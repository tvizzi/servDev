package controllers

import (
	"github.com/gofiber/fiber/v2"
	"log"
	"serv/database"
	"serv/jobs"
	"serv/models"
	"strconv"
)

func CreateArticle(c *fiber.Ctx) error {
	csrfToken := c.Locals("csrf")
	if csrfToken == nil {
		return c.Status(403).SendString("CSRF отсутствует")
	}
	log.Println("Полученый CSRF токен:", csrfToken)

	var article models.Article
	if err := c.BodyParser(&article); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	if article.Title == "" || len(article.Title) > 255 {
		return c.Status(400).JSON(fiber.Map{"error": "Название статьи не должно быть пустой"})
	}
	if article.Content == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Контент статьи обязателен"})
	}

	result := database.DB.Create(&article)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка при сохранении статьи"})
	}

	var readers []models.User
	if err := database.DB.Where("role = ?", "reader").Find(&readers).Error; err == nil {
		for _, reader := range readers {
			if reader.Email != "" {
				job := jobs.VeryLongJob{
					ArticleID: article.ID,
					Email:     reader.Email,
				}
				if err := job.Dispatch(); err != nil {
					log.Printf("Ошибка добавления задания в очередь для %s: %v", reader.Email, err)
				} else {
					log.Printf("Задание на отправку уведомления добавлено в очередь для %s", reader.Email)
				}
			}
		}
	} else {
		log.Printf("Читатели не найдены: %v", err)
	}

	return c.JSON(fiber.Map{
		"message": "Статья успешно создана",
		"id":      article.ID,
	})
}

func ListArticlesPage(c *fiber.Ctx) error {
	authToken := c.Cookies("auth_token")
	auth := authToken != ""

	isModerator := false
	if auth {
		var user models.User
		if err := database.DB.Where("auth_token = ?", authToken).First(&user).Error; err == nil {
			isModerator = user.Role == "moderator"
		}
	}

	log.Printf("User Moderator Status: %v", isModerator)

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	var articles []models.Article
	result := database.DB.Order("id DESC").Limit(limit).Offset(offset).Find(&articles)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка при получении данных"})
	}

	var total int64
	database.DB.Model(&models.Article{}).Count(&total)

	return c.Render("articles", fiber.Map{
		"Title":       "Список новостей",
		"Articles":    articles,
		"Page":        page,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
		"Total":       int(total),
		"CSRFToken":   c.Locals("csrf"),
		"IsModerator": isModerator,
	})
}

func UpdateArticle(c *fiber.Ctx) error {
	id := c.Params("id")
	var article models.Article

	if err := database.DB.First(&article, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Статья не найдена"})
	}

	if err := c.BodyParser(&article); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	if article.Title == "" || len(article.Title) > 255 {
		return c.Status(400).JSON(fiber.Map{"error": "Название статьи не должно быть пустым"})
	}

	if article.Content == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Контент статьи обязателен"})
	}

	database.DB.Save(&article)
	return c.JSON(fiber.Map{
		"message": "Статья успешно обновлена",
		"id":      article.ID,
	})
}

func RenderArticlePage(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		log.Printf("Неверный ID статьи: %v", err)
		return c.Status(400).SendString("Неверный ID статьи")
	}

	var article models.Article
	if err := database.DB.First(&article, id).Error; err != nil {
		log.Printf("Статья с ID %d не найдена: %v", id, err)
		return c.Status(404).SendString("Статья не найдена")
	}

	var comments []models.Comment
	if err := database.DB.Preload("User").Where("article_id = ?", article.ID).Find(&comments).Error; err != nil {
		log.Printf("Ошибка загрузки комментариев для статьи ID %d: %v", id, err)
		return c.Status(500).SendString("Ошибка загрузки комментариев")
	}

	authToken := c.Cookies("auth_token")
	auth := authToken != ""
	isModerator := false
	var currentUser models.User

	if auth {
		if err := database.DB.Where("auth_token = ?", authToken).First(&currentUser).Error; err != nil {
			log.Printf("Ошибка при получении пользователя по auth_token %s: %v", authToken, err)
			auth = false
		} else {
			isModerator = currentUser.Role == "moderator"
			log.Printf("Пользователь авторизован: Email=%s, Role=%s", currentUser.Email, currentUser.Role)
		}
	} else {
		log.Println("Пользователь не авторизован: auth_token отсутствует")
	}

	notification := c.Query("notification")

	err = c.Render("article", fiber.Map{
		"Title":        "Детальная страница",
		"Article":      article,
		"Comments":     comments,
		"Auth":         auth,
		"IsModerator":  isModerator,
		"CSRFToken":    c.Locals("csrf"),
		"CurrentUser":  currentUser,
		"Notification": notification,
	})
	if err != nil {
		log.Printf("Ошибка рендеринга шаблона для статьи ID %d: %v", id, err)
		return c.Status(500).SendString("Ошибка рендеринга")
	}

	return nil
}
