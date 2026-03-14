package config

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var DB *mongo.Database

func ConnectDB() {
	mongoURI := strings.TrimSpace(os.Getenv("MONGO_URI"))
	if mongoURI == "" {
		log.Println("MONGO_URI is not set; starting without database")
		return
	}

	dbName := strings.TrimSpace(os.Getenv("MONGO_DB_NAME"))
	if dbName == "" {
		dbName = "chess"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))

	if err != nil {
		log.Printf("Mongo connection error: %v", err)
		return
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer pingCancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		log.Printf("Mongo ping error: %v", err)
		return
	}

	DB = client.Database(dbName)

	log.Printf("MongoDB connected (db=%s)", dbName)
}
