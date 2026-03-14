package controllers

import (
	"chess/services"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

func JoinQueue(c *fiber.Ctx) error {

	type Body struct {
		PlayerID string `json:"playerId"`
	}

	var body Body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid"})
	}
	if body.PlayerID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "playerId is required"})
	}

	p1, p2, matched := services.Matchmaker.Join(body.PlayerID)

	if matched {

		game, err := services.StartMatch(p1, p2)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		payload, _ := json.Marshal(fiber.Map{
			"type": "match-found",
			"game": game,
		})
		services.Manager.Broadcast("player:"+p1, payload)
		services.Manager.Broadcast("player:"+p2, payload)
		services.Manager.Broadcast("game:"+game.GameID, payload)

		return c.JSON(fiber.Map{
			"matched": true,
			"game":    game,
		})
	}

	return c.JSON(fiber.Map{
		"matched":     false,
		"message":     "waiting for opponent",
		"queueLength": services.Matchmaker.QueueLength(),
	})
}
