package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
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

	// Pending click counters (buffered before flush to Postgres).
	redisClickKeyPrefix = "u:clk:"
	redisClickDirtySet  = "u:clk:dirty"

	clickFlushInterval = 2 * time.Second
	clickFlushBatch    = 500

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

func clickRedisKey(code string) string {
	return redisClickKeyPrefix + code
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

	pending := s.pendingClicks(ctx, code)

	res := &LookupResult{
		Code:              link.Code,
		ShortPath:         "/s/" + link.Code,
		ClickCount:        link.ClickCount + pending,
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

func (s *Shortener) pendingClicks(ctx context.Context, code string) uint64 {
	v, err := s.rdb.Get(ctx, clickRedisKey(code)).Result()
	if err == redis.Nil || v == "" {
		return 0
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// RecordClick buffers a redirect click in Redis (one RTT). Counts are flushed to Postgres
// periodically by RunClickFlush; Lookup adds pending Redis counts to the stored total.
func (s *Shortener) RecordClick(ctx context.Context, code string) {
	code = strings.TrimSpace(code)
	if code == "" {
		return
	}
	pipe := s.rdb.Pipeline()
	pipe.Incr(ctx, clickRedisKey(code))
	pipe.SAdd(ctx, redisClickDirtySet, code)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("click buffer redis: %v", err)
	}
}

// RunClickFlush periodically moves buffered click counts from Redis into Postgres until ctx is done.
func (s *Shortener) RunClickFlush(ctx context.Context) {
	t := time.NewTicker(clickFlushInterval)
	defer t.Stop()
	defer s.drainAllPendingClicks(context.Background())

	s.flushClickBatch(context.Background(), clickFlushBatch)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.flushClickBatch(context.Background(), clickFlushBatch)
		}
	}
}

func (s *Shortener) drainAllPendingClicks(ctx context.Context) {
	for {
		n := s.flushClickBatch(ctx, clickFlushBatch)
		if n == 0 {
			return
		}
	}
}

// flushClickBatch pops up to maxCodes entries from the dirty set and persists their counters.
// Returns how many codes were processed (including those with nothing to flush).
func (s *Shortener) flushClickBatch(ctx context.Context, maxCodes int) int {
	processed := 0
	for range maxCodes {
		code, err := s.rdb.SPop(ctx, redisClickDirtySet).Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			log.Printf("click flush spop: %v", err)
			break
		}
		processed++
		s.flushOneClickKey(ctx, code)
	}
	return processed
}

func (s *Shortener) flushOneClickKey(ctx context.Context, code string) {
	nStr, err := s.rdb.GetDel(ctx, clickRedisKey(code)).Result()
	if err == redis.Nil || nStr == "" {
		return
	}
	n, err := strconv.ParseUint(nStr, 10, 64)
	if err != nil || n == 0 {
		return
	}
	if err := s.repo.IncrementClicksBy(ctx, code, n); err != nil {
		log.Printf("click flush db %s: %v", code, err)
		_, _ = s.rdb.IncrBy(ctx, clickRedisKey(code), int64(n)).Result()
		_, _ = s.rdb.SAdd(ctx, redisClickDirtySet, code).Result()
		return
	}
	exists, _ := s.rdb.Exists(ctx, clickRedisKey(code)).Result()
	if exists > 0 {
		_, _ = s.rdb.SAdd(ctx, redisClickDirtySet, code).Result()
	}
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
