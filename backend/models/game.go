package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// const (
// 	player1 = "white"
// 	player2 = "black"
// )

type Game struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GameID    string             `bson:"gameId" json:"gameId"`
	Player1   string             `bson:"player1,omitempty" json:"player1,omitempty"`
	Player2   string             `bson:"player2,omitempty" json:"player2,omitempty"`
	Board     string             `bson:"board" json:"board"`
	Turn      string             `bson:"turn" json:"turn"`
	Status    string             `bson:"status" json:"status"`
	Moves     []string           `bson:"moves" json:"moves"`
	CreatedAt int64              `bson:"createdAt" json:"createdAt"`
	Winner    string             `bson:"winner,omitempty" json:"winner,omitempty"`
	Reason    string             `bson:"reason,omitempty" json:"reason,omitempty"`
}
