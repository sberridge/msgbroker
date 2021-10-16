package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoManager struct {
	connection *mongo.Client
}

func startMongo() (*mongoManager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, err
	}
	manager := mongoManager{
		connection: client,
	}
	return &manager, nil
}

func mongoFindOne(collection *mongo.Collection, projection bson.D, filter bson.D) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	findProjection := bson.D{primitive.E{Key: "id", Value: 1}, primitive.E{Key: "name", Value: 1}, primitive.E{Key: "_id", Value: 0}}
	findOptions := options.FindOne().SetProjection(findProjection)
	result := bson.M{}
	err := collection.FindOne(ctx, filter, findOptions).Decode(&result)
	return result, err
}

func mongoInsertOne(collection *mongo.Collection, row bson.D) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return collection.InsertOne(ctx, row)
}
