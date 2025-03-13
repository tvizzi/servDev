package mail

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"serv/models"
)

type Mailer struct {
	config SMTPConfig
}

func NewMailer() *Mailer {
	return &Mailer{config: LoadSMTPConfig()}
}

func (m *Mailer) SendNewArticleNotification(article *models.Article, moderatorEmail string) error {
	// Формирование текстового сообщения
	subject := "Новая статья добавлена"
	body := fmt.Sprintf("Здравствуйте!\n\nВ системе добавлена новая статья:\n- Название: %s\n- ID: %d",
		article.Title, article.ID)

	// Формирование письма
	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	to := []string{moderatorEmail}
	msg := []byte("To: " + moderatorEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	// Установка SSL-соединения
	addr := m.config.Host + ":" + m.config.Port
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.config.Host})
	if err != nil {
		log.Printf("Ошибка подключения к SMTP-серверу: %v", err)
		return err
	}
	defer conn.Close()

	// Создание SMTP-клиента поверх TLS-соединения
	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		log.Printf("Ошибка создания SMTP-клиента: %v", err)
		return err
	}
	defer client.Close()

	// Аутентификация
	if err = client.Auth(auth); err != nil {
		log.Printf("Ошибка аутентификации: %v", err)
		return err
	}

	// Отправка письма
	if err = client.Mail(m.config.FromAddr); err != nil {
		log.Printf("Ошибка установки отправителя: %v", err)
		return err
	}
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			log.Printf("Ошибка установки получателя %s: %v", recipient, err)
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		log.Printf("Ошибка открытия тела письма: %v", err)
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		log.Printf("Ошибка записи тела письма: %v", err)
		return err
	}
	err = w.Close()
	if err != nil {
		log.Printf("Ошибка закрытия тела письма: %v", err)
		return err
	}

	// Завершение соединения
	err = client.Quit()
	if err != nil {
		log.Printf("Ошибка завершения соединения: %v", err)
		return err
	}

	log.Printf("Письмо успешно отправлено на %s", moderatorEmail)
	return nil
}
