package main

import (
	"chess/config"
	"chess/routes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
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
	go config.ConnectDB()
	app := fiber.New(
		fiber.Config{
			Prefork:     false,
			JSONEncoder: json.Marshal,
			JSONDecoder: json.Unmarshal,
		},
	)
	app.Use(compress.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://chess-base.vercel.app,http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000,http://127.0.0.1:3000",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		database := "disconnected"
		if config.DB != nil {
			database = "connected"
		}
		return c.JSON(fiber.Map{
			"status":   "ok",
			"database": database,
		})
	})

	routes.GameRoutes(app)

	frontendBuildDir := resolveFrontendBuildDir()
	app.Get("/", func(c *fiber.Ctx) error {
		if frontendBuildDir != "" {
			return c.SendFile(filepath.Join(frontendBuildDir, "index.html"))
		}
		return c.JSON(fiber.Map{
			"message": "chess backend running",
		})
	})

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("listening on :%s", port)
	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}
