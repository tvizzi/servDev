package policies

import (
	"github.com/gofiber/fiber/v2"
	"log"
	"serv/database"
	"serv/models"
)

func getAuthenticatesUser(c *fiber.Ctx) *models.User {
	userIDInterface := c.Locals("userID")
	if userIDInterface == nil {
		log.Println("Ошибка: userID отсутствует в контексте")
		return nil
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		log.Println("Ошибка: userID не может быть преобразован в uint")
		return nil
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		log.Printf("Ошибка получения пользователя: %v", err)
		return nil
	}

	return &user
}

func IsModeratorByID(userID int) bool {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		log.Printf("Ошибка при загрузке пользователя с ID %d: %v", userID, err)
		return false
	}
	return user.Role == "moderator"
}

func IsReaderByID(userID int) bool {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		log.Printf("Ошибка при загрузке пользователя с ID %d: %v", userID, err)
		return false
	}
	return user.Role == "reader"
}
