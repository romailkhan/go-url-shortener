package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"url-shortener/internal/model"
	"url-shortener/internal/repository"
	"url-shortener/internal/shortcode"
)

const (
	redisKeyPrefix = "u:v2:"

	maxLinkLifetime = 366 * 24 * time.Hour
	minPasswordLen  = 8
)

// cacheEntry is stored in Redis for links without a password.
type cacheEntry struct {
	U string `json:"u"`
	X int64  `json:"x"`
}

// CreateInput holds optional password and lifetime for a new short link.
type CreateInput struct {
	TargetURL   string
	CustomAlias string
	Password    string
	ExpiresIn   time.Duration
}

// CreateResult is returned after a successful shorten.
type CreateResult struct {
	Code              string
	TargetURL         string
	ShortPath         string
	ExpiresAt         *time.Time
	PasswordProtected bool
}

// LookupResult is API metadata for a link.
type LookupResult struct {
	Code              string
	TargetURL         string
	ShortPath         string
	ClickCount        uint64
	ExpiresAt         *time.Time
	PasswordProtected bool
	CreatedAt         time.Time
}

// Shortener coordinates Postgres persistence and Redis cache-aside for lookups.
type Shortener struct {
	repo *repository.Link
	rdb  *redis.Client
}

// NewShortener builds a shortener service.
func NewShortener(repo *repository.Link, rdb *redis.Client) *Shortener {
	return &Shortener{repo: repo, rdb: rdb}
}

// Create stores a new short link and warms Redis when safe to cache.
func (s *Shortener) Create(ctx context.Context, in CreateInput) (*CreateResult, error) {
	targetURL := strings.TrimSpace(in.TargetURL)
	customAlias := strings.TrimSpace(in.CustomAlias)
	password := strings.TrimSpace(in.Password)

	if password != "" && len(password) < minPasswordLen {
		return nil, BadInput(fmt.Sprintf("password must be at least %d characters", minPasswordLen))
	}

	var expiresAt *time.Time
	if in.ExpiresIn > 0 {
		if in.ExpiresIn > maxLinkLifetime {
			return nil, BadInput(fmt.Sprintf("expires_in must be at most %v", maxLinkLifetime))
		}
		t := time.Now().UTC().Add(in.ExpiresIn)
		expiresAt = &t
	}

	var hash string
	if password != "" {
		b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hash = string(b)
	}

	if customAlias != "" {
		if err := validateCode(customAlias); err != nil {
			return nil, BadInput(err.Error())
		}
		return s.createWithCode(ctx, customAlias, targetURL, hash, expiresAt)
	}

	for range 16 {
		code, err := shortcode.Random(shortcode.DefaultLength)
		if err != nil {
			return nil, err
		}
		res, err := s.createWithCode(ctx, code, targetURL, hash, expiresAt)
		if err != nil {
			if errors.Is(err, ErrConflict) {
				continue
			}
			return nil, err
		}
		return res, nil
	}
	return nil, fmt.Errorf("exhausted unique code retries")
}

func (s *Shortener) createWithCode(ctx context.Context, code, targetURL, passwordHash string, expiresAt *time.Time) (*CreateResult, error) {
	link := &model.Link{
		Code:         code,
		TargetURL:    targetURL,
		PasswordHash: passwordHash,
		ExpiresAt:    expiresAt,
	}
	if err := s.repo.Create(ctx, link); err != nil {
		if repository.IsUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}

	s.warmCache(ctx, link)

	return &CreateResult{
		Code:              code,
		TargetURL:         targetURL,
		ShortPath:         "/s/" + code,
		ExpiresAt:         expiresAt,
		PasswordProtected: passwordHash != "",
	}, nil
}

func (s *Shortener) warmCache(ctx context.Context, link *model.Link) {
	if link.HasPassword() {
		return
	}
	now := time.Now()
	if link.IsExpired(now) {
		return
	}
	key := cacheKey(link.Code)
	ce := cacheEntry{U: link.TargetURL, X: expiryUnix(link)}
	b, err := json.Marshal(ce)
	if err != nil {
		return
	}
	ttl := redisTTL(link, now)
	_ = s.rdb.Set(ctx, key, b, ttl).Err()
}

func expiryUnix(link *model.Link) int64 {
	if link.ExpiresAt == nil {
		return 0
	}
	return link.ExpiresAt.Unix()
}

func redisTTL(link *model.Link, now time.Time) time.Duration {
	if link.ExpiresAt == nil {
		return 0
	}
	d := link.ExpiresAt.Sub(now)
	if d < time.Second {
		return time.Second
	}
	return d
}

func cacheKey(code string) string {
	return redisKeyPrefix + code
}

// Resolve returns the target URL after optional password and expiry checks.
func (s *Shortener) Resolve(ctx context.Context, code, password string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", ErrNotFound
	}
	now := time.Now().UTC()
	key := cacheKey(code)

	raw, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		if u, ok := s.parseCache(ctx, raw, now, key); ok {
			return u, nil
		}
	} else if err != redis.Nil {
		return "", err
	}

	link, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}

	if link.IsExpired(now) {
		return "", ErrExpired
	}

	if err := s.checkPassword(link, password); err != nil {
		return "", err
	}

	s.warmCache(ctx, link)
	return link.TargetURL, nil
}

func (s *Shortener) parseCache(ctx context.Context, raw string, now time.Time, key string) (string, bool) {
	if len(raw) > 0 && raw[0] == '{' {
		var ce cacheEntry
		if json.Unmarshal([]byte(raw), &ce) != nil {
			return "", false
		}
		if ce.X > 0 && now.Unix() >= ce.X {
			_, _ = s.rdb.Del(ctx, key).Result()
			return "", false
		}
		return ce.U, true
	}
	// Legacy plain URL (v1 cache)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw, true
	}
	return "", false
}

func (s *Shortener) checkPassword(link *model.Link, password string) error {
	if !link.HasPassword() {
		return nil
	}
	if password == "" {
		return ErrPasswordRequired
	}
	if bcrypt.CompareHashAndPassword([]byte(link.PasswordHash), []byte(password)) != nil {
		return ErrPasswordInvalid
	}
	return nil
}

// Lookup returns metadata; target URL is omitted unless password (if any) is valid.
func (s *Shortener) Lookup(ctx context.Context, code, password string) (*LookupResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrNotFound
	}
	now := time.Now().UTC()

	link, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if link.IsExpired(now) {
		return nil, ErrExpired
	}

	res := &LookupResult{
		Code:              link.Code,
		ShortPath:         "/s/" + link.Code,
		ClickCount:        link.ClickCount,
		ExpiresAt:         link.ExpiresAt,
		PasswordProtected: link.HasPassword(),
		CreatedAt:         link.CreatedAt,
	}

	if link.HasPassword() {
		if err := s.checkPassword(link, password); err != nil {
			return nil, err
		}
	}

	res.TargetURL = link.TargetURL
	return res, nil
}

// RecordClick increments the persisted click counter (best-effort for redirects).
func (s *Shortener) RecordClick(ctx context.Context, code string) {
	code = strings.TrimSpace(code)
	if code == "" {
		return
	}
	_ = s.repo.IncrementClicks(ctx, code)
}

func validateCode(alias string) error {
	if len(alias) < 3 || len(alias) > 32 {
		return fmt.Errorf("custom code must be between 3 and 32 characters")
	}
	for _, r := range alias {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return fmt.Errorf("custom code may only use letters, digits, hyphen, and underscore")
		}
	}
	return nil
}
