package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"url-shortener/internal/repository"
	"url-shortener/internal/shortcode"
)

const redisKeyPrefix = "u:v1:"

// Shortener coordinates Postgres persistence and Redis cache-aside for lookups.
type Shortener struct {
	repo *repository.Link
	rdb  *redis.Client
}

// NewShortener builds a shortener service.
func NewShortener(repo *repository.Link, rdb *redis.Client) *Shortener {
	return &Shortener{repo: repo, rdb: rdb}
}

// CreateResult is returned after a successful shorten.
type CreateResult struct {
	Code      string
	TargetURL string
	ShortPath string // e.g. /s/abc
}

// Create stores a new short link and warms Redis.
func (s *Shortener) Create(ctx context.Context, targetURL, customAlias string) (*CreateResult, error) {
	targetURL = strings.TrimSpace(targetURL)
	customAlias = strings.TrimSpace(customAlias)

	if customAlias != "" {
		if err := validateCode(customAlias); err != nil {
			return nil, BadInput(err.Error())
		}
		if err := s.repo.Create(ctx, customAlias, targetURL); err != nil {
			if repository.IsUniqueViolation(err) {
				return nil, ErrConflict
			}
			return nil, err
		}
		_ = s.rdb.Set(ctx, cacheKey(customAlias), targetURL, 0).Err()
		return &CreateResult{
			Code:      customAlias,
			TargetURL: targetURL,
			ShortPath: "/s/" + customAlias,
		}, nil
	}

	for range 16 {
		code, err := shortcode.Random(shortcode.DefaultLength)
		if err != nil {
			return nil, err
		}
		if err := s.repo.Create(ctx, code, targetURL); err != nil {
			if repository.IsUniqueViolation(err) {
				continue
			}
			return nil, err
		}
		_ = s.rdb.Set(ctx, cacheKey(code), targetURL, 0).Err()
		return &CreateResult{
			Code:      code,
			TargetURL: targetURL,
			ShortPath: "/s/" + code,
		}, nil
	}
	return nil, fmt.Errorf("exhausted unique code retries")
}

// Resolve returns the target URL for a short code (Redis then Postgres).
func (s *Shortener) Resolve(ctx context.Context, code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", ErrNotFound
	}

	key := cacheKey(code)
	u, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		return u, nil
	}
	if err != redis.Nil {
		return "", err
	}

	link, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	_ = s.rdb.Set(ctx, key, link.TargetURL, 0).Err()
	return link.TargetURL, nil
}

// Lookup returns code and target for API responses (no stats).
func (s *Shortener) Lookup(ctx context.Context, code string) (*CreateResult, error) {
	u, err := s.Resolve(ctx, code)
	if err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	return &CreateResult{
		Code:      code,
		TargetURL: u,
		ShortPath: "/s/" + code,
	}, nil
}

func cacheKey(code string) string {
	return redisKeyPrefix + code
}

func validateCode(s string) error {
	if len(s) < 3 || len(s) > 32 {
		return fmt.Errorf("custom code must be between 3 and 32 characters")
	}
	for _, r := range s {
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
