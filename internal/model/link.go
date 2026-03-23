package model

import "time"

// Link is a persisted short URL mapping.
type Link struct {
	ID        uint64    `gorm:"primaryKey"`
	Code      string    `gorm:"size:32;uniqueIndex;not null"`
	TargetURL string    `gorm:"size:2048;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (Link) TableName() string {
	return "links"
}
