package main

import (
	"github.com/gofiber/fiber/v2"
	"log"
	"regexp"
)

type AuthController struct{}

// отдает страницу регистрации через метод
func (ctrl *AuthController) Create(c *fiber.Ctx) error {
	csrfToken := c.Locals("csrf")
	log.Println("CSRF Token:", csrfToken)

	return render(c, "signin", fiber.Map{
		"Title": "Регистрация",
		"CSRF":  csrfToken, // генерируем csrf токен
	})
}

// обработка данных с формы
func (ctrl *AuthController) Registration(c *fiber.Ctx) error {
	type RegistrationForm struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var form RegistrationForm

	// парс данных с формы
	if err := c.BodyParser(&form); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	// валидация
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	if form.Name == "" || form.Email == "" || form.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Все поля обязательны для заполнения"})
	} else if !regexp.MustCompile(emailRegex).MatchString(form.Email) {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный формат email"})
	}

	// ответ в джсоне
	return c.JSON(fiber.Map{
		"message": "Регистрация успешна",
		"data": fiber.Map{
			"name":     form.Name,
			"email":    form.Email,
			"password": form.Password,
		},
	})
}
