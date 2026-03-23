package model

import "time"

// Link is a persisted short URL mapping.
type Link struct {
	ID           uint64     `gorm:"primaryKey"`
	Code         string     `gorm:"size:32;uniqueIndex;not null"`
	TargetURL    string     `gorm:"size:2048;not null"`
	PasswordHash string     `gorm:"size:255"` // bcrypt; empty = no password
	ExpiresAt    *time.Time `gorm:"index"`
	ClickCount   uint64     `gorm:"not null;default:0"`
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
}

func (Link) TableName() string {
	return "links"
}

// HasPassword reports whether visiting the link requires a password.
func (l *Link) HasPassword() bool {
	return l.PasswordHash != ""
}

// IsExpired reports whether the link is past its optional expiry.
func (l *Link) IsExpired(now time.Time) bool {
	if l.ExpiresAt == nil {
		return false
	}
	return !now.Before(*l.ExpiresAt)
}
