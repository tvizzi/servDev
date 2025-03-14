package database

import (
	"log"
	"serv/models"
)

func Migrate() {
	DB.Migrator().DropTable("roles", "user_roles", &models.Job{})

	err := DB.AutoMigrate(&models.Article{}, &models.User{}, &models.Comment{}, &models.Job{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	if !DB.Migrator().HasColumn(&models.User{}, "AuthToken") {
		err = DB.Migrator().AddColumn(&models.User{}, "AuthToken")
		if err != nil {
			log.Fatalf("Ошибка добавления колонки AuthToken: %v", err)
		}
		log.Println("Колонка AuthToken успешно добавлена")
	}

	if !DB.Migrator().HasColumn(&models.User{}, "AuthToken") {
		err = DB.Migrator().AddColumn(&models.User{}, "AuthToken")
		if err != nil {
			log.Fatalf("Ошибка добавления колонки AuthToken: %v", err)
		}
		log.Println("Колонка AuthToken успешно добавлена")
	}

	log.Println("Миграция успешна")
}
