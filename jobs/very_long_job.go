package jobs

import (
	"encoding/json"
	"log"
	"serv/database"
	"serv/mail"
	"serv/models"
	"time"
)

type VeryLongJob struct {
	ArticleID uint   `json:"article_id"`
	Email     string `json:"email"`
}

func (j *VeryLongJob) Handle() error {
	var article models.Article
	if err := database.DB.First(&article, j.ArticleID).Error; err != nil {
		log.Printf("Ошибка загрузки статьи с ID %d: %v", j.ArticleID, err)
		return err
	}

	mailer := mail.NewMailer()
	err := mailer.SendNewArticleNotification(&article, j.Email)
	if err != nil {
		log.Printf("Ошибка отправки уведомления на %s: %v", j.Email, err)
		return err
	}

	log.Printf("Уведомление успешно отправлено на %s для статьи ID %d", j.Email, j.ArticleID)
	return nil

}

func (j *VeryLongJob) Dispatch() error {
	payload, err := json.Marshal(j)
	if err != nil {
		return err
	}

	job := models.Job{
		Queue:       "default",
		Payload:     string(payload),
		AvailableAt: time.Now(),
	}
	return database.DB.Create(&job).Error
}
