package main

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"html/template"
	"io"
	"log"
	"os"
	"serv/controllers"
	"serv/database"
	"serv/jobs"
	"serv/models"
	"serv/policies"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Article struct {
	ID           int    `json:"-"`
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

	// Загружаем все шаблоны из папки views
	templates, err := template.New("").Funcs(funcMap).ParseGlob(pattern)
	if err != nil {
		log.Fatalf("Ошибка при загрузке шаблонов: %v", err)
	}

	return &TemplateEngine{
		templates: templates,
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
	authToken := c.Cookies("auth_token")
	auth := authToken != ""

	isModerator := false
	if auth {
		var user models.User
		if err := database.DB.Where("auth_token = ?", authToken).First(&user).Error; err == nil {
			isModerator = user.Role == "moderator"
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

	notification := c.Query("notification")

	return c.Render("layout", fiber.Map{
		"Title":        "Главная",
		"Page":         "home",
		"Auth":         auth,
		"Articles":     articles,
		"IsModerator":  isModerator,
		"Notification": notification,
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

	var articles []Article
	if err := json.NewDecoder(file).Decode(&articles); err != nil {
		log.Printf("Ошибка при декодировании JSON: %v", err)
		return c.Status(500).SendString("Ошибка декодирования данных")
	}

	for index, article := range articles {
		if id == fmt.Sprintf("%d", index+1) {
			return c.Render("layout", fiber.Map{
				"Title":   "Галерея",
				"Page":    "gallery",
				"Article": article,
			})
		}
	}

	return c.Status(404).SendString("Статья не найдена")
}

func authMiddleware(c *fiber.Ctx) error {
	authToken := c.Cookies("auth_token")
	if authToken == "" {
		log.Println("auth_token отсутствует")
		return c.Status(401).JSON(fiber.Map{"error": "Неавторизованный доступ"})
	}

	var user models.User
	if err := database.DB.Where("auth_token = ?", authToken).First(&user).Error; err != nil {
		log.Printf("Ошибка поиска пользователя по токену: %v", err)
		return c.Status(401).JSON(fiber.Map{"error": "Неверный токен или пользователь не найден"})
	}

	c.Locals("userID", user.ID)
	return c.Next()
}

func ModeratorMiddleware(c *fiber.Ctx) error {
	userIDInterface := c.Locals("userID")
	userID, ok := userIDInterface.(uint)
	if !ok {
		log.Println("Ошибка: userID имеет неверный тип")
		return c.Status(401).JSON(fiber.Map{"error": "Неверный идентификатор пользователя"})
	}

	if userID == 0 {
		return c.Status(403).JSON(fiber.Map{"error": "Недостаточно прав"})
	}

	if !policies.IsModeratorByID(int(userID)) {
		return c.Status(403).JSON(fiber.Map{"error": "Недостаточно прав"})
	}

	return c.Next()
}

func ProcessQueue() {
	for {
		var job models.Job
		if err := database.DB.Where("reserved_at IS NULL AND available_at <= ?", time.Now()).
			Order("id ASC").First(&job).Error; err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		job.ReservedAt = new(time.Time)
		*job.ReservedAt = time.Now()
		job.Attempts++
		database.DB.Save(&job)

		var veryLongJob jobs.VeryLongJob
		if err := json.Unmarshal([]byte(job.Payload), &veryLongJob); err != nil {
			log.Printf("Ошибка декодирования задания ID %d: %v", job.ID, err)
			database.DB.Delete(&job)
			continue
		}

		if err := veryLongJob.Handle(); err != nil {
			log.Printf("Ошибка выполнения задания ID %d: %v", job.ID, err)
			if job.Attempts >= 3 {
				database.DB.Delete(&job)
				log.Printf("Задание ID %d удалено после 3 неудачных попыток", job.ID)
			} else {
				job.ReservedAt = nil
				database.DB.Save(&job)
			}
			continue
		}

		database.DB.Delete(&job)
		log.Printf("Задание ID %d успешно выполнено и удалено", job.ID)
	}
}

func main() {
	database.ConnectDB()
	time.Sleep(5 * time.Second)
	database.Migrate()
	database.SeedArticles()
	database.SeedRoles()

	go ProcessQueue()

	engine := NewTemplateEngine("./views/*.html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(logger.New())

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

	controller := &Controller{}
	authController := &controllers.AuthController{}

	app.Get("/gallery/:id", controller.Gallery)

	app.Get("/signup", func(c *fiber.Ctx) error {
		csrfToken := c.Locals("csrf")
		return c.Render("signin", fiber.Map{
			"Title": "Регистрация",
			"CSRF":  csrfToken,
		})
	})

	app.Post("/signup", authController.Registration)

	app.Get("/signin", func(c *fiber.Ctx) error {
		csrfToken := c.Locals("csrf")
		return c.Render("login", fiber.Map{
			"Title": "Авторизация",
			"CSRF":  csrfToken,
		})
	})

	app.Get("/", controller.Index)
	app.Post("/login", authController.Login)
	app.Get("/logout", authController.Logout)

	app.Get("/about", func(c *fiber.Ctx) error {
		auth := c.Cookies("auth_token") != ""
		return c.Render("layout", fiber.Map{
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
		return c.Render("layout", fiber.Map{
			"Title":    "Контакты",
			"Page":     "contacts",
			"Contacts": contacts,
			"Auth":     auth,
		})
	})

	app.Get("/articles/edit/:id", authMiddleware, ModeratorMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		var article models.Article

		if err := database.DB.First(&article, id).Error; err != nil {
			return c.Status(404).SendString("Статья не найдена")
		}

		return c.Render("edit_article", fiber.Map{
			"Title":     "Редактировать статью",
			"Article":   article,
			"CSRFToken": c.Locals("csrf"),
		})
	})

	app.Post("/articles/edit/:id", authMiddleware, ModeratorMiddleware, controllers.UpdateArticle)

	app.Delete("/articles/:id", authMiddleware, ModeratorMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")

		if err := database.DB.Delete(&models.Article{}, id).Error; err != nil {
			log.Printf("Ошибка удаления статьи с ID %s: %v", id, err)
			return c.Status(404).JSON(fiber.Map{"error": "Статья не найдена"})
		}

		log.Printf("Статья с ID %s успешно удалена", id)
		return c.SendStatus(204)
	})

	app.Get("/api/user", func(c *fiber.Ctx) error {
		authToken := c.Cookies("auth_token")
		if authToken == "" {
			return c.Status(401).JSON(fiber.Map{"authenticated": false})
		}

		var user models.User
		if err := database.DB.Where("auth_token = ?", authToken).First(&user).Error; err != nil {
			return c.Status(401).JSON(fiber.Map{"authenticated": false})
		}

		role := user.Role
		if role == "" {
			role = "user"
		}

		return c.JSON(fiber.Map{
			"authenticated": true,
			"user": fiber.Map{
				"id":    user.ID,
				"email": user.Email,
				"roles": role,
			},
		})
	})

	app.Post("/articles/:id/comments", authMiddleware, func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uint)
		if userID == 0 {
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
