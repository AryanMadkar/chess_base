package controllers

import (
	"chess/services"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func broadcastGameUpdate(gameID string, game any) {
	msg, err := json.Marshal(fiber.Map{
		"type": "game-update",
		"game": game,
	})
	if err != nil {
		return
	}
	services.Manager.Broadcast("game:"+gameID, msg)
}

func CreateGame(c *fiber.Ctx) error {
	game, err := services.CreateGame()
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "database unavailable") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "database unavailable",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(game)
}

func JoinGame(c *fiber.Ctx) error {
	type Body struct {
		GameID   string `json:"gameId"`
		PlayerID string `json:"playerId"`
	}
	var body Body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}
	if body.PlayerID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "playerId is required"})
	}
	if body.GameID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "gameId is required"})
	}
	game, role, err := services.JoinGame(body.GameID, body.PlayerID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	broadcastGameUpdate(game.GameID, game)
	return c.JSON(fiber.Map{
		"game": game,
		"role": role,
	})
}

func GetGameState(c *fiber.Ctx) error {
	gameID := c.Query("gameId")
	if gameID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "gameId is required"})
	}

	game, err := services.GetGame(gameID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(game)
}

func ResignGame(c *fiber.Ctx) error {
	type Body struct {
		GameID   string `json:"gameId"`
		PlayerID string `json:"playerId"`
	}
	var body Body

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	game, err := services.ResignGame(body.GameID, body.PlayerID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	broadcastGameUpdate(game.GameID, game)
	return c.JSON(game)
}

func MakeMove(c *fiber.Ctx) error {
	type Body struct {
		GameID   string `json:"gameId"`
		PlayerID string `json:"playerId"`
		Move     string `json:"move"`
	}
	var body Body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if body.GameID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "gameId is required"})
	}
	if body.PlayerID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "playerId is required"})
	}
	if body.Move == "" {
		return c.Status(400).JSON(fiber.Map{"error": "move is required"})
	}
	game, err := services.MakeMove(body.GameID, body.PlayerID, body.Move)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	broadcastGameUpdate(game.GameID, game)
	return c.JSON(game)
}
