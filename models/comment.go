package models

import "time"

type Comment struct {
	ID        uint      `gorm:"primaryKey"`
	Content   string    `gorm:"type:text;not null"`
	UserID    uint      `gorm:"not null"`
	User      User      `gorm:"foreignKey:UserID"`
	ArticleID uint      `gorm:"not null"`
	Article   Article   `gorm:"foreignKey:ArticleID"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
