package routes

import (
	"chess/controllers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func GameRoutes(app *fiber.App) {
	app.Post("/api/game/create", controllers.CreateGame)
	app.Post("/api/game/join", controllers.JoinGame)
	app.Get("/api/game/state", controllers.GetGameState)
	app.Post("/api/game/move", controllers.MakeMove)
	app.Get("/ws/game", websocket.New(controllers.GameSocket))
	app.Post("/api/game/resign", controllers.ResignGame)
	app.Post("/api/matchmaking/join", controllers.JoinQueue)
}
