package database

import (
	"log"
	"serv/models"
)

func Migrate() {
	err := DB.AutoMigrate(&models.Article{}, &models.User{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}
	log.Println("Миграция успешна")
}
