package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"io"
	"log"
	"os"
	"serv/controllers"
	"serv/database"
	"serv/models"
	"serv/policies"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Article struct {
	ID           int    `json:"-"` // Не экспортируется в JSON
	Date         string `json:"date"`
	Name         string `json:"name"`
	PreviewImage string `json:"preview_image"`
	FullImage    string `json:"full_image"`
	ShortDesc    string `json:"shortDesc"`
	Desc         string `json:"desc"`
}

type Controller struct{}

type TemplateEngine struct {
	templates *template.Template
}

func NewTemplateEngine(pattern string) *TemplateEngine {
	funcMap := template.FuncMap{
		"mul": func(a, b int) int { return a * b },
		"toInt": func(v interface{}) int {
			switch val := v.(type) {
			case string:
				i, _ := strconv.Atoi(val)
				return i
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			default:
				return 0
			}
		},
	}

	return &TemplateEngine{
		templates: template.Must(template.New("").Funcs(funcMap).ParseGlob(pattern)),
	}
}

func (t *TemplateEngine) Render(w io.Writer, name string, data interface{}, layout ...string) error {
	err := t.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("Ошибка рендеринга шаблона %s: %v", name, err)
		return fmt.Errorf("ошибка рендеринга шаблона: %w", err)
	}
	return nil
}

func (t *TemplateEngine) Load() error {
	return nil
}

func (ctrl *Controller) Index(c *fiber.Ctx) error {
	auth := c.Cookies("auth_token") != ""

	isModerator := false
	if auth {
		if userIDInterface := c.Locals("userID"); userIDInterface != nil {
			if userID, ok := userIDInterface.(int); ok {
				isModerator = policies.IsModeratorByID(userID)
			}
		}
	}

	file, err := os.Open("articles.json")
	if err != nil {
		log.Printf("Ошибка при открытии файла: %v", err)
		return c.Status(500).SendString("Ошибка при чтении данных")
	}
	defer file.Close()

	var articles []Article
	if err := json.NewDecoder(file).Decode(&articles); err != nil {
		log.Printf("Ошибка при декодировании JSON: %v", err)
		return c.Status(500).SendString("Ошибка декодирования данных")
	}

	for i := range articles {
		articles[i].ID = i + 1
	}

	return render(c, "layout", fiber.Map{
		"Title":       "Главная",
		"Page":        "home",
		"Auth":        auth,
		"Articles":    articles,
		"IsModerator": isModerator,
	})
}

func (ctrl *Controller) Gallery(c *fiber.Ctx) error {
	id := c.Params("id")

	file, err := os.Open("articles.json")
	if err != nil {
		log.Printf("Ошибка при открытии файла: %v", err)
		return c.Status(500).SendString("Ошибка при чтении данных")
	}
	defer file.Close()

	// Парсим
	var articles []Article
	if err := json.NewDecoder(file).Decode(&articles); err != nil {
		log.Printf("Ошибка при декодировании JSON: %v", err)
		return c.Status(500).SendString("Ошибка декодирования данных")
	}

	// ищем статью по айди
	for index, article := range articles {
		if id == fmt.Sprintf("%d", index+1) {
			// rend gall
			return render(c, "layout", fiber.Map{
				"Title":   "Галерея",
				"Page":    "gallery",
				"Article": article,
			})
		}
	}

	return c.Status(404).SendString("Статья не найдена")
}

// кастом func для обработки html т.к fiber больше не поддерживает обработку html)
func render(c *fiber.Ctx, name string, data fiber.Map) error {
	tmpl, err := template.ParseFiles("./views/" + name + ".html")
	if err != nil {
		log.Printf("Ошибка при парсинге шаблона: %v", err)
		return c.Status(500).SendString("Ошибка шаблона")
	}

	// Используем буф для хранения результата рендера
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Printf("Ошибка при рендеринге шаблона: %v", err)
		return c.Status(500).SendString("Ошибка рендеринга")
	}

	return c.Type("html", "utf-8").Send(buf.Bytes())
}

func ModeratorMiddleware(c *fiber.Ctx) error {
	userIDInterface := c.Locals("userID")
	var userID uint

	if id, ok := userIDInterface.(int); ok {
		userID = uint(id)
	} else if id, ok := userIDInterface.(uint); ok {
		userID = id
	} else {
		log.Println("Ошибка: userID не может быть преобразован в uint или int")
		return c.Status(401).JSON(fiber.Map{"error": "Неверный идентификатор пользователя"})
	}

	if userID == 0 {
		log.Println("Ошибка: userID равен 0")
		return c.Status(403).JSON(fiber.Map{"error": "Недостаточно прав"})
	}

	if !policies.IsModeratorByID(int(userID)) { // Обновлено для передачи userID в IsModerator
		return c.Status(403).JSON(fiber.Map{"error": "Недостаточно прав"})
	}

	return c.Next()
}

func authMiddleware(c *fiber.Ctx) error {
	authToken := c.Cookies("auth_token")
	log.Printf("Полученный auth_token: %s", authToken)

	if authToken == "" {
		log.Println("auth_token отсутствует")
		return c.Status(401).JSON(fiber.Map{"error": "Неавторизованный доступ"})
	}

	var user models.User
	if err := database.DB.Where("auth_token = ?", authToken).First(&user).Error; err != nil {
		log.Printf("Ошибка поиска пользователя по токену: %v", err)
		return c.Status(401).JSON(fiber.Map{"error": "Неверный токен или пользователь не найден"})
	}

	log.Printf("Пользователь найден: ID=%d, Name=%s", user.ID, user.Name)
	c.Locals("userID", user.ID)
	return c.Next()
}

func main() {
	database.ConnectDB()
	database.Migrate()
	database.SeedArticles()
	database.SeedRoles()

	engine := NewTemplateEngine("./views/*.html")

	// Создаем приложение Fiber
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Middleware для логов
	app.Use(logger.New())

	// CSRF Middleware
	app.Use(csrf.New(csrf.Config{
		KeyLookup:      "header:X-CSRF-Token",
		CookieName:     "csrf_",
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
		ContextKey:     "csrf",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Println("CSRF Error:", err)
			return c.Status(fiber.StatusForbidden).SendString("Forbidden - Invalid CSRF token")
		},
	}))

	app.Static("/img", "./img")

	// Контроллеры
	controller := &Controller{}
	authController := &AuthController{}

	// Маршруты
	app.Get("/gallery/:id", controller.Gallery)

	// Маршруты AuthController
	app.Get("/signup", func(c *fiber.Ctx) error {
		csrfToken := c.Locals("csrf")
		return render(c, "signin", fiber.Map{
			"Title": "Регистрация",
			"CSRF":  csrfToken,
		})
	})

	app.Post("/signup", func(c *fiber.Ctx) error {
		type SignupForm struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		var form SignupForm
		if err := c.BodyParser(&form); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Неверный формат данных"})
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

		if err := database.DB.Create(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Ошибка при сохранении пользователя"})
		}

		token, err := createAuthToken(user.ID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Ошибка создания токена"})
		}

		c.Cookie(&fiber.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			HTTPOnly: true,
		})

		return c.Redirect("/")
	})

	app.Get("/signin", func(c *fiber.Ctx) error {
		csrfToken := c.Locals("csrf")
		return render(c, "login", fiber.Map{
			"Title": "Авторизация",
			"CSRF":  csrfToken,
		})
	})

	app.Get("/", controller.Index)
	app.Post("/signin", authController.Registration)
	app.Post("/login", authController.Login)
	app.Get("logout", authController.Logout)

	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("Доступ разрешен")
	})
	app.Get("/about", func(c *fiber.Ctx) error {
		auth := c.Cookies("auth_token") != ""
		return render(c, "layout", fiber.Map{
			"Title": "О нас",
			"Page":  "about",
			"Auth":  auth,
		})
	})

	app.Get("/contacts", func(c *fiber.Ctx) error {
		contacts := map[string]string{
			"Phone":   "112",
			"Email":   "da@gmail.com",
			"Address": "ERFHERFHRFHERJFHEFJEHFEJ",
		}

		auth := c.Cookies("auth_token") != ""
		return render(c, "layout", fiber.Map{
			"Title":    "Контакты",
			"Page":     "contacts",
			"Contacts": contacts,
			"Auth":     auth,
		})
	})

	// Для модеров
	app.Get("/articles/edit/:id", ModeratorMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		var article models.Article

		if err := database.DB.First(&article, id).Error; err != nil {
			return c.Status(404).SendString("Статья не найдена")
		}

		return render(c, "edit_article", fiber.Map{
			"Title":     "Редактировать статью",
			"Article":   article,
			"CSRFToken": c.Locals("csrf"),
		})
	})

	app.Post("/articles/edit/:id", ModeratorMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		var article models.Article

		if err := database.DB.First(&article, id).Error; err != nil {
			return c.Status(404).SendString("Статья не найдена")
		}

		var updateData struct {
			Title   string `form:"title"`
			Content string `form:"content"`
		}

		if err := c.BodyParser(&updateData); err != nil {
			log.Printf("Ошибка парсинга формы: %v", err)
			return c.Status(400).JSON(fiber.Map{"error": "Неверный формат данных"})
		}

		log.Printf("Редактируем статью с ID %s. Новые данные: Title=%s, Content=%s", id, updateData.Title, updateData.Content)

		article.Title = updateData.Title
		article.Content = updateData.Content

		if err := database.DB.Save(&article).Error; err != nil {
			log.Printf("Ошибка сохранения статьи: %v", err)
			return c.Status(500).SendString("Ошибка при обновлении статьи")
		}

		return c.Redirect("/articles")
	})

	app.Delete("/articles/:id", ModeratorMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")

		if err := database.DB.Delete(&models.Article{}, id).Error; err != nil {
			log.Printf("Ошибка удаления статьи с ID %s: %v", id, err)
			return c.Status(404).JSON(fiber.Map{"error": "Статья не найдена"})
		}

		log.Printf("Статья с ID %s успешно удалена", id)
		return c.SendStatus(204)
	})

	app.Post("/login", func(c *fiber.Ctx) error {
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

		token, err := createAuthToken(user.ID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Ошибка создания токена"})
		}

		c.Cookie(&fiber.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			HTTPOnly: true,
		})

		return c.JSON(fiber.Map{"message": "Успешный вход"})
	})

	// коменты

	app.Post("/articles/:id/comments", authMiddleware, func(c *fiber.Ctx) error {
		userIDInterface := c.Locals("userID")
		userID, ok := userIDInterface.(uint)
		if !ok {
			if userIDInt, ok := userIDInterface.(int); ok {
				userID = uint(userIDInt)
			} else {
				log.Println("Ошибка: userID не является uint или int")
				return c.Status(401).JSON(fiber.Map{"error": "Неверный идентификатор пользователя"})
			}
		}

		if userID == 0 {
			log.Println("Ошибка: userID равен 0")
			return c.Status(403).JSON(fiber.Map{"error": "Недостаточно прав"})
		}

		if !policies.IsReaderByID(int(userID)) {
			return c.Status(403).JSON(fiber.Map{"error": "Недостаточно прав"})
		}

		articleID := c.Params("id")
		var article models.Article
		if err := database.DB.First(&article, articleID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Статья не найдена"})
		}

		type CommentData struct {
			Content string `json:"content"`
		}
		var commentData CommentData
		if err := c.BodyParser(&commentData); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Неверные данные"})
		}

		comment := models.Comment{
			Content:   commentData.Content,
			UserID:    userID,
			ArticleID: article.ID,
		}
		if err := database.DB.Create(&comment).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Ошибка при сохранении комментария"})
		}

		return c.JSON(fiber.Map{"message": "Комментарий добавлен", "comment": comment})
	})

	app.Get("/articles/:id", controllers.RenderArticlePage)
	app.Get("/articles", controllers.ListArticlesPage)
	app.Post("/articles", authMiddleware, controllers.CreateArticle)
	app.Put("/articles/:id", authMiddleware, controllers.UpdateArticle)

	log.Fatal(app.Listen(":3000"))
}
