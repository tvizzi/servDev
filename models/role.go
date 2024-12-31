package models

type Role struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"size:255;not null;unique"`
	Users []User `gorm:"many2many:user_roles;"`
}
