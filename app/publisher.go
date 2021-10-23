package main

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type jsonPublisher struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	OwnerID string `json:"owner_id"`
}

func checkPublisherExists(name string, mongoManager *mongoManager) (bool, error) {

	//open collection containing publisher details
	col := mongoManager.connection.Database("message-broker").Collection("publishers")
	filter := bson.D{{Key: "name", Value: name}}

	count, err := mongoCount(col, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func registerPublisher(owner *clientConnection, data newPublisherRequestData, mongoManager *mongoManager) (newId string, err error) {

	//open collection containing publisher details
	col := mongoManager.connection.Database("message-broker").Collection("publishers")

	newId = uuid.New().String()
	_, err = mongoInsertOne(col, bson.D{{Key: "_id", Value: newId}, {Key: "name", Value: data.Name}, {Key: "owner_id", Value: owner.id}})

	if err != nil {
		return "", err
	}
	return
}

func newPublisher(owner *clientConnection, data newPublisherRequestData, mongoManager *mongoManager) (*jsonPublisher, error) {
	exists, err := checkPublisherExists(data.Name, mongoManager)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("publisher exists")
	}
	newId, err := registerPublisher(owner, data, mongoManager)
	if err != nil {
		return nil, err
	}
	publisher := jsonPublisher{
		Id:      newId,
		Name:    data.Name,
		OwnerID: owner.id,
	}
	return &publisher, nil
}

func publishMessage(owner *clientConnection, data publishMessageRequestData, mongoManager *mongoManager) (bool, error) {

	col := mongoManager.connection.Database("message-broker").Collection("publishers")
	filter := bson.D{{Key: "_id", Value: data.PublisherID}, {Key: "owner_id", Value: owner.id}}

	count, err := mongoCount(col, filter)
	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, errors.New("publisher not found")
	}

	timeToExpire := int64(0)
	if data.Ttl > 0 {
		timeToExpire = time.Now().Unix() + data.Ttl
	}

	messagesCollection := mongoManager.connection.Database("message-broker").Collection("publisher_messages")
	_, err = mongoInsertOne(messagesCollection, bson.D{
		{Key: "_id", Value: uuid.New().String()},
		{Key: "publisher_id", Value: data.PublisherID},
		{Key: "payload", Value: data.Payload},
		{Key: "date_created", Value: time.Now()},
		{Key: "ttl", Value: timeToExpire},
	})

	if err != nil {
		return false, err
	}
	return true, nil
}

type bsonPublisher struct {
	Id      string `bson:"_id"`
	Name    string `bson:"name"`
	OwnerID string `bson:"owner_id"`
}

func getPublishers(owner *clientConnection, mongoManager *mongoManager) ([]jsonPublisher, error) {
	col := mongoManager.connection.Database("message-broker").Collection("publishers")
	filter := bson.D{{Key: "owner_id", Value: owner.id}}

	results, err := mongoFindMany(col, options.Find().SetProjection(bson.D{
		{Key: "_id", Value: 1},
		{Key: "name", Value: 1},
	}), filter)

	if err != nil {
		return nil, err
	}
	bpublishers := []bsonPublisher{}
	publishers := []jsonPublisher{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	results.All(ctx, &bpublishers)
	for _, p := range bpublishers {
		publishers = append(publishers, jsonPublisher(p))
	}
	return publishers, nil

}
