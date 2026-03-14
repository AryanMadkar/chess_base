package controllers

import (
	"chess/services"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func GameSocket(c *websocket.Conn) {
	defer services.Manager.LeaveAll(c)

	gameID := c.Query("gameId")
	if gameID != "" {
		services.Manager.Join("game:"+gameID, c)
	}

	playerID := c.Query("playerId")
	if playerID != "" {
		services.Manager.Join("player:"+playerID, c)
	}

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		var data map[string]string
		if err := json.Unmarshal(msg, &data); err != nil {
			res, _ := json.Marshal(fiber.Map{
				"type":  "error",
				"error": "invalid websocket payload",
			})
			_ = c.WriteMessage(websocket.TextMessage, res)
			continue
		}
		switch data["type"] {
		case "subscribe-game":
			subscribeGameID := data["gameId"]
			if subscribeGameID == "" {
				subscribeGameID = data["gameid"]
			}
			if subscribeGameID != "" {
				services.Manager.Join("game:"+subscribeGameID, c)
			}

		case "queue":
			playerID := data["playerId"]
			if playerID == "" {
				playerID = data["playerid"]
			}
			if playerID == "" {
				res, _ := json.Marshal(fiber.Map{
					"type":  "error",
					"error": "playerId is required",
				})
				_ = c.WriteMessage(websocket.TextMessage, res)
				continue
			}

			p1, p2, matched := services.Matchmaker.Join(playerID)
			if matched {
				game, err := services.StartMatch(p1, p2)
				if err != nil {
					res, _ := json.Marshal(fiber.Map{
						"type":  "error",
						"error": "failed to create match",
					})
					_ = c.WriteMessage(websocket.TextMessage, res)
					continue
				}

				payload, _ := json.Marshal(fiber.Map{
					"type": "match-found",
					"game": game,
				})

				services.Manager.Broadcast("player:"+p1, payload)
				services.Manager.Broadcast("player:"+p2, payload)
				services.Manager.Broadcast("game:"+game.GameID, payload)
			} else {
				queuedRes, _ := json.Marshal(fiber.Map{
					"type":        "queue-waiting",
					"queueLength": services.Matchmaker.QueueLength(),
				})
				_ = c.WriteMessage(websocket.TextMessage, queuedRes)
			}
		}
	}
}
