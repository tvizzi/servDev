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

// Структура для статьи
type Article struct {
	ID           int    `json:"-"` // Не экспортируется в JSON
	Date         string `json:"date"`
	Name         string `json:"name"`
	PreviewImage string `json:"preview_image"`
	FullImage    string `json:"full_image"`
	ShortDesc    string `json:"shortDesc"`
	Desc         string `json:"desc"`
}

// Контроллер для обработки запросов
type Controller struct{}

type TemplateEngine struct {
	templates *template.Template
}

func NewTemplateEngine(pattern string) *TemplateEngine {
	return &TemplateEngine{
		templates: template.Must(template.ParseGlob(pattern)),
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
	// Открываем JSON-файл
	file, err := os.Open("articles.json")
	if err != nil {
		log.Printf("Ошибка при открытии файла: %v", err)
		return c.Status(500).SendString("Ошибка при чтении данных")
	}
	defer file.Close()

	// Парсим данные
	var articles []Article
	if err := json.NewDecoder(file).Decode(&articles); err != nil {
		log.Printf("Ошибка при декодировании JSON: %v", err)
		return c.Status(500).SendString("Ошибка декодирования данных")
	}

	// Добавляем ID к каждой статье
	for i := range articles {
		articles[i].ID = i + 1
	}

	// Рендерим главную страницу
	return render(c, "layout", fiber.Map{
		"Title":    "Главная",
		"Page":     "home",
		"Articles": articles,
	})
}

func (ctrl *Controller) Gallery(c *fiber.Ctx) error {
	// Получаем ID статьи
	id := c.Params("id")

	// Открываем JSON-файл
	file, err := os.Open("articles.json")
	if err != nil {
		log.Printf("Ошибка при открытии файла: %v", err)
		return c.Status(500).SendString("Ошибка при чтении данных")
	}
	defer file.Close()

	// Парсим данные
	var articles []Article
	if err := json.NewDecoder(file).Decode(&articles); err != nil {
		log.Printf("Ошибка при декодировании JSON: %v", err)
		return c.Status(500).SendString("Ошибка декодирования данных")
	}

	// Находим статью по ID
	for index, article := range articles {
		if id == fmt.Sprintf("%d", index+1) {
			// Рендерим галерею
			return render(c, "layout", fiber.Map{
				"Title":   "Галерея",
				"Page":    "gallery",
				"Article": article,
			})
		}
	}

	// Если статья не найдена
	return c.Status(404).SendString("Статья не найдена")
}

// Функция для рендера HTML
func render(c *fiber.Ctx, name string, data fiber.Map) error {
	tmpl, err := template.ParseFiles("./views/" + name + ".html")
	if err != nil {
		log.Printf("Ошибка при парсинге шаблона: %v", err)
		return c.Status(500).SendString("Ошибка шаблона")
	}

	// Используем bytes.Buffer для хранения результата рендера
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
		KeyLookup:      "form:_csrf",
		CookieName:     "csrf_",
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
		ContextKey:     "csrf",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Println("CSRF Error:", err)
			return c.Status(fiber.StatusForbidden).SendString("Forbidden - Invalid CSRF token")
		},
	}))

	// Обработка статических файлов
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

	// Страницы "О нас" и "Контакты"
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

	app.Get("/articles/:id", controllers.RenderArticlePage)
	app.Get("/articles", controllers.ListArticlesPage)
	app.Post("/articles", controllers.CreateArticle)
	app.Put("/articles/:id", controllers.UpdateArticle)
	app.Delete("/articles/:id", controllers.DeleteArticle)

	// Запуск сервера
	log.Fatal(app.Listen(":3000"))
}
