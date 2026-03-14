package services

import (
	"chess/config"
	"chess/models"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/notnil/chess"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	turnWhite      = "white"
	turnBlack      = "black"
	statusWaiting  = "waiting"
	statusActive   = "active"
	statusFinished = "finished"
)

func gamesCollection() (*mongo.Collection, error) {
	if config.DB == nil {
		return nil, errors.New("database unavailable")
	}
	return config.DB.Collection("games"), nil
}

func CreateGame() (*models.Game, error) {
	ctx := context.TODO()
	loc, _ := time.LoadLocation("Asia/Kolkata")
	game := models.Game{
		GameID:    uuid.New().String(),
		Board:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		Turn:      turnWhite,
		Status:    statusWaiting,
		CreatedAt: time.Now().In(loc).Unix(),
	}
	collection, err := gamesCollection()
	if err != nil {
		return nil, err
	}
	insertResult, err := collection.InsertOne(ctx, game)
	if err != nil {
		return nil, err
	}
	if oid, ok := insertResult.InsertedID.(primitive.ObjectID); ok {
		game.ID = oid
	}
	return &game, nil
}

func JoinGame(gameID string, playerID string) (*models.Game, string, error) {
	collection, err := gamesCollection()
	if err != nil {
		return nil, "", err
	}
	filter := bson.M{"gameId": gameID}

	var game models.Game
	err = collection.FindOne(context.TODO(), filter).Decode(&game)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, "", errors.New("game not found")
		}
		return nil, "", err
	}
	role := ""
	if game.Player1 == "" {
		game.Player1 = playerID
		role = "White"
	} else if game.Player2 == "" {
		game.Player2 = playerID
		role = "Black"
		game.Status = statusActive
	} else {
		return nil, "", errors.New("game full")
	}
	_, err = collection.UpdateOne(
		context.TODO(),
		filter,
		bson.M{"$set": bson.M{
			"player1": game.Player1,
			"player2": game.Player2,
			"status":  game.Status,
		}},
	)
	if err != nil {
		return nil, "", err
	}
	return &game, role, nil
}

func GetGame(gameID string) (*models.Game, error) {
	collection, err := gamesCollection()
	if err != nil {
		return nil, err
	}
	var game models.Game
	err = collection.FindOne(context.TODO(), bson.M{"gameId": gameID}).Decode(&game)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("game not found")
		}
		return nil, err
	}
	return &game, nil
}

func deleteGameByID(gameID string) error {
	collection, err := gamesCollection()
	if err != nil {
		return err
	}
	_, err = collection.DeleteOne(context.TODO(), bson.M{"gameId": gameID})
	return err
}

func applyMove(game *models.Game, playerID, moveStr string) error {
	if game.Status != statusActive {
		return errors.New("game not active")
	}

	turn := strings.ToLower(strings.TrimSpace(game.Turn))
	if turn == turnWhite && game.Player1 != playerID || turn == turnBlack && game.Player2 != playerID {
		return errors.New("not your turn")
	}

	fenOpt, err := chess.FEN(game.Board)
	if err != nil {
		return err
	}

	chessGame := chess.NewGame(fenOpt)
	move, err := chess.UCINotation{}.Decode(chessGame.Position(), moveStr)
	if err != nil {
		return errors.New("invalid move format")
	}

	if err := chessGame.Move(move); err != nil {
		return errors.New("illegal move")
	}
	outcome := chessGame.Outcome()
	method := chessGame.Method()
	if outcome != chess.NoOutcome {
		game.Status = statusFinished
		switch outcome {
		case chess.WhiteWon:
			game.Winner = "white"
		case chess.BlackWon:
			game.Winner = "black"
		default:
			game.Winner = "draw"
		}
		switch method {
		case chess.Checkmate:
			game.Reason = "checkmate"
		case chess.Stalemate:
			game.Reason = "stalemate"
		case chess.DrawOffer:
			game.Reason = "draw"
		default:
			game.Reason = "finished"
		}
	}

	game.Board = chessGame.Position().String()
	game.Moves = append(game.Moves, moveStr)
	if turn == turnWhite {
		game.Turn = turnBlack
	} else {
		game.Turn = turnWhite
	}

	if chessGame.Outcome() != chess.NoOutcome {
		game.Status = statusFinished
	}

	return nil
}

func MakeMove(gameID, playerID, moveStr string) (*models.Game, error) {
	collection, err := gamesCollection()
	if err != nil {
		return nil, err
	}
	var game models.Game
	err = collection.FindOne(context.TODO(), bson.M{"gameId": gameID}).Decode(&game)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("game not found")
		}
		return nil, err
	}
	if game.Status == "finished" {
		return nil, errors.New("game already finished")
	}

	if err := applyMove(&game, playerID, moveStr); err != nil {
		return nil, err
	}

	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"gameId": gameID},
		bson.M{"$set": bson.M{
			"board":  game.Board,
			"moves":  game.Moves,
			"turn":   game.Turn,
			"status": game.Status,
			"winner": game.Winner,
			"reason": game.Reason,
		}},
	)
	if err != nil {
		return nil, err
	}

	if game.Status == statusFinished {
		if err := deleteGameByID(gameID); err != nil {
			return nil, err
		}
	}

	return &game, nil
}

func ResignGame(gameID, playerID string) (*models.Game, error) {
	collection, err := gamesCollection()
	if err != nil {
		return nil, err
	}
	var game models.Game
	err = collection.FindOne(context.TODO(), bson.M{"gameId": gameID}).Decode(&game)
	if err != nil {
		return nil, err
	}
	if game.Status == statusFinished {
		return nil, errors.New("game already finished")
	}
	game.Status = statusFinished
	game.Reason = "resign"
	if game.Player1 == playerID {
		game.Winner = game.Player2
	} else {
		game.Winner = game.Player1
	}
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"gameId": gameID},
		bson.M{"$set": bson.M{
			"status": game.Status,
			"winner": game.Winner,
			"reason": game.Reason,
		}},
	)
	if err != nil {
		return nil, err
	}

	if err := deleteGameByID(gameID); err != nil {
		return nil, err
	}

	return &game, nil
}
