package controllers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"log"
	"regexp"
	"serv/database"
	"serv/models"
	"time"
)

type AuthController struct{}

func (ctrl *AuthController) Create(c *fiber.Ctx) error {
	csrfToken := c.Locals("csrf")
	log.Println("CSRF Token:", csrfToken)

	return c.Render("signin", fiber.Map{
		"Title": "Регистрация",
		"CSRF":  csrfToken,
	})
}

func (ctrl *AuthController) Registration(c *fiber.Ctx) error {
	log.Println("Registration POST запрос получен")

	type RegistrationForm struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var form RegistrationForm
	log.Println("Попытка парса")

	if err := c.BodyParser(&form); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	if form.Name == "" || form.Email == "" || form.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Все поля обязательны для заполнения"})
	} else if !regexp.MustCompile(emailRegex).MatchString(form.Email) {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный формат email"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка сервера"})
	}

	user := models.User{
		Name:     form.Name,
		Email:    form.Email,
		Password: string(hashedPassword),
	}

	result := database.DB.Create(&user)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка при сохранении пользователя"})
	}

	token, err := CreateAuthToken(user.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка создания токена"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
	})

	return c.Redirect("/?notification=Регистрация успешна")
}

func CreateAuthToken(userID uint) (string, error) {
	token := fmt.Sprintf("user_%d_token", userID)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return "", err
	}

	user.AuthToken = token
	if err := database.DB.Save(&user).Error; err != nil {
		return "", err
	}

	return token, nil
}

func (a *AuthController) Login(c *fiber.Ctx) error {
	type LoginForm struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var form LoginForm
	if err := c.BodyParser(&form); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	var user models.User
	if err := database.DB.Where("email = ?", form.Email).First(&user).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный email или пароль"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(form.Password)); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный email или пароль"})
	}

	token, err := CreateAuthToken(user.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка создания токена"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	return c.Redirect("/?notification=Вход успешен")
}

func (ctrl *AuthController) Logout(c *fiber.Ctx) error {
	c.ClearCookie("auth_token")
	c.Locals("csrf", nil)
	// Редирект без уведомления в JSON, уведомление будет на главной
	return c.Redirect("/?notification=Выход успешен")
}
