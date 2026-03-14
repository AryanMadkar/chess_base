package main

import (
	"chess/config"
	"chess/routes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

func resolveFrontendBuildDir() string {
	candidates := []string{
		"public",
		"../public",
		"backend/public",
		"../backend/public",
	}

	for _, candidate := range candidates {
		indexPath := filepath.Join(candidate, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			return candidate
		}
	}

	return ""
}

func main() {
	config.ConnectDB()
	app := fiber.New(
		fiber.Config{
			Prefork:     false,
			JSONEncoder: json.Marshal,
			JSONDecoder: json.Unmarshal,
		},
	)
	app.Use(compress.New())

	routes.GameRoutes(app)

	frontendBuildDir := resolveFrontendBuildDir()
	if frontendBuildDir == "" {
		log.Println("frontend build not found, serving API only")
	} else {
		app.Static("/", frontendBuildDir, fiber.Static{
			Compress: true,
			MaxAge:   3600,
		})

		app.Get("/*", func(c *fiber.Ctx) error {
			path := c.Path()
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/ws") {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "route not found"})
			}
			return c.SendFile(filepath.Join(frontendBuildDir, "index.html"))
		})
	}

	log.Fatal(app.Listen(":3000"))
}
