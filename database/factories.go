package database

import (
	"github.com/bxcodec/faker/v3"
	"log"
	"serv/models"
)

func SeedArticles() {
	for i := 0; i < 10; i++ {
		article := models.Article{
			Title:   faker.Sentence(),
			Content: faker.Paragraph(),
		}
		result := DB.Create(&article)
		if result.Error != nil {
			log.Printf("Ошибка при добавлении статьи: %v", result.Error)
		}
	}
	log.Println("Успешно")
}
