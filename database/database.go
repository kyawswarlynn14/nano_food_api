package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DBSet() *mongo.Client {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("MONGODB_URI environment variable not set")
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Println("failed to connect to mongodb")
		return nil
	}
	fmt.Println("Successfully Connected to the mongodb")
	return client
}

var Client *mongo.Client = DBSet()

func NanoFoodData(client *mongo.Client, CollectionName string) *mongo.Collection {
	return client.Database("nano_food").Collection(CollectionName)
}

var UserCollection *mongo.Collection = NanoFoodData(Client, "users")
var BranchCollection *mongo.Collection = NanoFoodData(Client, "branches")
var CategoryCollection *mongo.Collection = NanoFoodData(Client, "categories")
var MenuCollection *mongo.Collection = NanoFoodData(Client, "menus")
var AddOnCollection *mongo.Collection = NanoFoodData(Client, "add_ons")
var TableCollection *mongo.Collection = NanoFoodData(Client, "tables")
var OrderCollection *mongo.Collection = NanoFoodData(Client, "orders")
var SaleCollection *mongo.Collection = NanoFoodData(Client, "sales")
