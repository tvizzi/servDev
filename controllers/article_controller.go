package controllers

import (
	"github.com/gofiber/fiber/v2"
	"log"
	"serv/database"
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
		return c.Status(400).JSON(fiber.Map{"error": "контент статьи обязателен"})
	}

	result := database.DB.Create(&article)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка при сохранении статьи"})
	}

	var total int64
	database.DB.Model(&models.Article{}).Count(&total)

	page := 1
	limit := 10
	offset := (page - 1) * limit

	var articles []models.Article
	database.DB.Order("id DESC").Limit(limit).Offset(offset).Find(&articles)

	return c.Render("articles", fiber.Map{
		"Title":     "Список новостей",
		"Articles":  articles,
		"Page":      page,
		"PrevPage":  page - 1,
		"NextPage":  page + 1,
		"Total":     int(total),
		"CSRFToken": c.Locals("csrf"),
	})
}

func ListArticlesPage(c *fiber.Ctx) error {
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
		"Title":     "Список новостей",
		"Articles":  articles,
		"Page":      page,
		"PrevPage":  page - 1,
		"NextPage":  page + 1,
		"Total":     int(total),
		"CSRFToken": c.Locals("csrf"),
	})
}

func GetArticleByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var article models.Article

	result := database.DB.First(&article, id)
	if result.Error != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Статья не найдена"})
	}

	return c.JSON(article)
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
		return c.Status(400).JSON(fiber.Map{"error": "название статьи не должны быть пустым"})
	}

	if article.Content == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Контент статьи обязателен"})
	}

	database.DB.Save(&article)
	return c.JSON(article)
}

func DeleteArticle(c *fiber.Ctx) error {
	id := c.Params("id")
	var article models.Article

	if err := database.DB.Delete(&article, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Ошибка удаления"})
	}

	return c.SendStatus(204)
}

func RenderArticlePage(c *fiber.Ctx) error {
	id := c.Params("id")
	var article models.Article

	result := database.DB.First(&article, id)
	if result.Error != nil {
		return c.Status(404).SendString("Статья не найдена")
	}

	err := c.Render("article", fiber.Map{
		"Title":     "Детальная страница",
		"Article":   article,
		"CSRFToken": c.Locals("csrf"),
	})
	if err != nil {
		log.Printf("Ошибка рендеринга шаблона: %v", err)
		return c.Status(500).SendString("Ошибка рендеринга")
	}

	return nil
}
