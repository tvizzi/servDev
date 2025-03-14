package models

import "time"

type Job struct {
	ID          uint   `gorm:"primaryKey"`
	Queue       string `gorm:"size:255;not null;default:'default'"`
	Payload     string `gorm:"type:text;not null"`
	Attempts    uint   `gorm:"default:0"`
	ReservedAt  *time.Time
	AvailableAt time.Time `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}
