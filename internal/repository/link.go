package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"url-shortener/internal/model"
)

// Link persists and loads short links.
type Link struct {
	db *gorm.DB
}

// NewLink creates a link repository.
func NewLink(db *gorm.DB) *Link {
	return &Link{db: db}
}

// Create inserts a new link. Caller must ensure code uniqueness.
func (r *Link) Create(ctx context.Context, link *model.Link) error {
	return r.db.WithContext(ctx).Create(link).Error
}

// FindByCode returns the link or gorm.ErrRecordNotFound.
func (r *Link) FindByCode(ctx context.Context, code string) (*model.Link, error) {
	var link model.Link
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&link).Error
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// IncrementClicks adds one to click_count for the given code.
func (r *Link) IncrementClicks(ctx context.Context, code string) error {
	res := r.db.WithContext(ctx).Model(&model.Link{}).
		Where("code = ?", code).
		UpdateColumn("click_count", gorm.Expr("click_count + ?", 1))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IsUniqueViolation reports Postgres unique constraint violations (23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}
