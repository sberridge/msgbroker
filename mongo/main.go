package bezmongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoService struct {
	connection *mongo.Client
}

func (mongoService *MongoService) OpenCollection(database string, collection string) *mongo.Collection {
	return mongoService.connection.Database(database).Collection(collection)
}

func StartMongo() (*MongoService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	mongoOptions := options.Client()
	//mongoOptions = mongoOptions.ApplyURI("mongodb://localhost:27017")
	mongoOptions = mongoOptions.ApplyURI("mongodb://message_broker_db:27017")
	/* creds := options.Credential{
		Username: "",
		Password: "",
	}
	mongoOptions.Auth = &creds */
	client, err := mongo.Connect(ctx, mongoOptions)
	if err != nil {
		return nil, err
	}

	manager := MongoService{
		connection: client,
	}

	return &manager, nil
}

func FindOne(collection *mongo.Collection, projection bson.D, filter bson.D) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	findOptions := options.FindOne().SetProjection(projection)
	return collection.FindOne(ctx, filter, findOptions)
}

func FindMany(collection *mongo.Collection, options *options.FindOptions, filter bson.D) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return collection.Find(ctx, filter, options)
}

func Count(collection *mongo.Collection, filter bson.D) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return collection.CountDocuments(ctx, filter)
}

func InsertOne(collection *mongo.Collection, row bson.D) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return collection.InsertOne(ctx, row)
}

func DeleteMany(collection *mongo.Collection, filter bson.D) (*mongo.DeleteResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return collection.DeleteMany(ctx, filter)
}

func UpdateMany(collection *mongo.Collection, filter bson.D, update bson.D) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return collection.UpdateMany(ctx, filter, update)
}

func UpdateOne(collection *mongo.Collection, filter bson.D, update bson.D) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return collection.UpdateOne(ctx, filter, update)
}

func Aggregate(collection *mongo.Collection, stages []bson.D) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pipeline := mongo.Pipeline{}
	for _, stage := range stages {
		pipeline = append(pipeline, stage)
	}
	return collection.Aggregate(ctx, pipeline)
}
