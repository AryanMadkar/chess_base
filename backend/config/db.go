package config

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database

func ConnectDB() {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb+srv://aradhyamadkar10_db_user:Ashlesha3462@cluster0drum.lzmremr.mongodb.net/?appName=Cluster0drum"))

	if err != nil {
		log.Fatal("Mongo connection error:", err)
	}

	DB = client.Database("chess")

	log.Println("MongoDB connected")
}