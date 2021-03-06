package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoManager struct {
	connection *mongo.Client
}

func (mongoManager *mongoManager) openCollection(database string, collection string) *mongo.Collection {
	return mongoManager.connection.Database(database).Collection(collection)
}

func startMongo() (*mongoManager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://message_broker_db:27017"))
	//client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, err
	}
	manager := mongoManager{
		connection: client,
	}
	return &manager, nil
}

func mongoFindOne(collection *mongo.Collection, projection bson.D, filter bson.D) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	findOptions := options.FindOne().SetProjection(projection)
	return collection.FindOne(ctx, filter, findOptions)
}

func mongoFindMany(collection *mongo.Collection, findOptions *options.FindOptions, filter bson.D) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return collection.Find(ctx, filter, findOptions)
}

func mongoUpdateMany(collection *mongo.Collection, filter bson.D, update bson.D) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	updateResult, err := collection.UpdateMany(ctx, filter, update)
	return updateResult, err
}

func mongoDeleteMany(collection *mongo.Collection, filter bson.D) (*mongo.DeleteResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	deleteResult, err := collection.DeleteMany(ctx, filter)
	return deleteResult, err
}
