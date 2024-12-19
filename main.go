package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/gorm"
	"html/template"
	"io"
	"log"
	"os"
	"serv/controllers"
	"serv/database"
	"serv/models"

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
	file, err := os.Open("articles.json")
	if err != nil {
		log.Printf("Ошибка при открытии файла: %v", err)
		return c.Status(500).SendString("Ошибка при чтении данных")
	}
	defer file.Close() // скипнем эту ошибку

	var articles []Article
	if err := json.NewDecoder(file).Decode(&articles); err != nil {
		log.Printf("Ошибка при декодировании JSON: %v", err)
		return c.Status(500).SendString("Ошибка декодирования данных")
	}

	for i := range articles {
		articles[i].ID = i + 1
	}

	return render(c, "layout", fiber.Map{
		"Title":    "Главная",
		"Page":     "home",
		"Articles": articles,
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

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(&models.Article{})
	if err != nil {
		log.Fatal("Migration Failed", err)
	}
	fmt.Println("Migration completed")
}

func multiSourceCSRFCheck(c *fiber.Ctx) error {
	token := c.Get("X-CSRF-Token")
	if token == "" {
		token = c.FormValue("_csrf")
	}

	if token == "" || token != c.Cookies("csrf_") {
		log.Println("CSRF Error: Token missing or invalid")
		return c.Status(fiber.StatusForbidden).SendString("Forbidden - Invalid CSRF token")
	}

	return c.Next()
}

func main() {
	database.ConnectDB()
	database.Migrate()
	database.SeedArticles()

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

	app.Use(multiSourceCSRFCheck)

	app.Static("/img", "./img")

	// Контроллеры
	controller := &Controller{}
	authController := &AuthController{}

	// Маршруты
	app.Get("/", controller.Index)
	app.Get("/gallery/:id", controller.Gallery)

	// Маршруты AuthController
	app.Get("/signin", authController.Create)
	app.Post("/signin", authController.Registration)

	// Страницы о нас и компания
	app.Get("/about", func(c *fiber.Ctx) error {
		return render(c, "layout", fiber.Map{
			"Title": "О нас",
			"Page":  "about",
		})
	})

	app.Get("/contacts", func(c *fiber.Ctx) error {
		contacts := map[string]string{
			"Phone":   "112",
			"Email":   "da@gmail.com",
			"Address": "ERFHERFHRFHERJFHEFJEHFEJ",
		}
		return render(c, "layout", fiber.Map{
			"Title":    "Контакты",
			"Page":     "contacts",
			"Contacts": contacts,
		})
	})

	app.Get("/articles/edit/:id", func(c *fiber.Ctx) error {
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

	app.Post("/articles/edit/:id", func(c *fiber.Ctx) error {
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

	app.Delete("/articles/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		if err := database.DB.Delete(&models.Article{}, id).Error; err != nil {
			log.Printf("Ошибка удаления статьи с ID %s: %v", id, err)
			return c.Status(404).JSON(fiber.Map{"error": "Статья не найдена"})
		}

		log.Printf("Статья с ID %s успешно удалена", id)
		return c.SendStatus(204)
	})

	app.Get("/articles/:id", controllers.RenderArticlePage)
	app.Get("/articles", controllers.ListArticlesPage)
	app.Post("/articles", controllers.CreateArticle)
	app.Put("/articles/:id", controllers.UpdateArticle)

	log.Fatal(app.Listen(":3000"))
}
