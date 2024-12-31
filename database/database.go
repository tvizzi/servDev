package database

import (
	"fmt"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
	"serv/models"
)

var DB *gorm.DB

func ConnectDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to db: ", err)
	}

	DB = db
	fmt.Println("Db connected")
}

func SeedRoles() {
	moderator := models.Role{Name: "moderator"}
	reader := models.Role{Name: "reader"}

	//проверяем есть ли роли

	var existingRoles []models.Role
	DB.Find(&existingRoles)
	if len(existingRoles) > 0 {
		fmt.Println("Роли существуют, сиды запущены")
		return
	}

	if err := DB.Create(&moderator).Error; err != nil {
		log.Fatalf("Ошибка при добавлении роли модератора: %v", err)
	}

	if err := DB.Create(&reader).Error; err != nil {
		log.Fatalf("Ошибка при добавлении роли читателя: %v", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("megapassword"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Ошибка при хэшировании пароля: %v", err)
	}

	moderatorUser := models.User{
		Name:     "Moderator",
		Email:    "moder@gmail.com",
		Password: string(hashedPassword),
		Roles:    []models.Role{moderator},
	}

	if err := DB.Create(&moderatorUser).Error; err != nil {
		log.Fatalf("Ошибка при добавлении пользователя-модератора: %v", err)
	}

	fmt.Println("Роли и пользователь-модератор успешно добавлены")
}
