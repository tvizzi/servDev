package models

import "time"

type Article struct {
	ID          uint      `gorm:"primaryKey"`
	Title       string    `gorm:"size:255;not null"`
	Content     string    `gorm:"type:text;not null"`
	PublishedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
