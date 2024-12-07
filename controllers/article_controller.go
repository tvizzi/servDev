package controllers

import (
	"github.com/gofiber/fiber/v2"
	"log"
	"serv/database"
	"serv/models"
)

func CreateArticle(c *fiber.Ctx) error {
	var article models.Article

	if err := c.BodyParser(&article); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	result := database.DB.Create(&article)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка при сохранении статьи"})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Статья успешно создана",
		"article": article,
	})
}

func ListArticlesPage(c *fiber.Ctx) error {
	var articles []models.Article
	result := database.DB.Find(&articles)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка при получении данных"})
	}

	err := c.Render("articles", fiber.Map{
		"Title":    "Список новостей",
		"Articles": articles,
	})
	if err != nil {
		log.Printf("Ошибка рендеринга шаблона: %v", err)
		return c.Status(500).SendString("Ошибка рендеринга")
	}

	return nil
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
		"Title":   "Детальная страница",
		"Article": article,
	})
	if err != nil {
		log.Printf("Ошибка рендеринга шаблона: %v", err)
		return c.Status(500).SendString("Ошибка рендеринга")
	}

	return nil
}
