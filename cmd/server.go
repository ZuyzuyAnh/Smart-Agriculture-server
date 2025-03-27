package main

import (
	"context"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.ERROR)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	mongoClient, ctx, cancel, err := connectMongoDB(os.Getenv("MONGO_URI"))
	if err != nil {
		panic(err)
	}

	defer closeMongoDB(mongoClient, ctx, cancel)
}

func connectMongoDB(uri string) (*mongo.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	return client, ctx, cancel, err
}

func closeMongoDB(client *mongo.Client, ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}


