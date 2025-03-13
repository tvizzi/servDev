package mail

import (
	"os"
	"strconv"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	FromAddr string
	FromName string
}

func LoadSMTPConfig() SMTPConfig {
	port, _ := strconv.Atoi(os.Getenv("MAIL_PORT"))
	return SMTPConfig{
		Host:     os.Getenv("MAIL_HOST"),
		Port:     strconv.Itoa(port),
		Username: os.Getenv("MAIL_USERNAME"),
		Password: os.Getenv("MAIL_PASSWORD"),
		FromAddr: os.Getenv("MAIL_FROM_ADDRESS"),
		FromName: os.Getenv("MAIL_FROM_NAME"),
	}
}
