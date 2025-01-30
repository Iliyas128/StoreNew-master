package tests

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var testDB *mongo.Database
var cartCollection *mongo.Collection
var loginCollection *mongo.Collection

// Настройка тестовой базы данных
func setupTestDB() {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}
	testDB = client.Database("test_store")
	cartCollection = testDB.Collection("cart")
	loginCollection = testDB.Collection("users")
}

// Очистка коллекций перед тестами
func clearCollections() {
	cartCollection.DeleteMany(context.Background(), bson.D{})
	loginCollection.DeleteMany(context.Background(), bson.D{})
}

