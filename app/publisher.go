package main

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type publisher struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Owner_id string `json:"owner_id"`
}

func checkPublisherExists(name string, mongoManager *mongoManager) (bool, error) {

	//open collection containing publisher details
	col := mongoManager.connection.Database("message-broker").Collection("publishers")
	filter := bson.D{primitive.E{Key: "name", Value: name}}
	findProjection := bson.D{primitive.E{Key: "id", Value: 1}, primitive.E{Key: "name", Value: 1}, primitive.E{Key: "_id", Value: 0}}

	_, err := mongoFindOne(col, findProjection, filter)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		} else {
			return false, err
		}
	}
	return false, nil
}

func registerPublisher(owner *clientConnection, data newPublisherRequestData, mongoManager *mongoManager) (newId string, err error) {

	//open collection containing publisher details
	col := mongoManager.connection.Database("message-broker").Collection("publishers")

	newId = uuid.New().String()
	_, err = mongoInsertOne(col, bson.D{primitive.E{Key: "id", Value: newId}, primitive.E{Key: "name", Value: data.Name}, primitive.E{Key: "owner_id", Value: owner.id}})

	if err != nil {
		return "", err
	}
	return
}

func newPublisher(owner *clientConnection, data newPublisherRequestData, mongoManager *mongoManager) (*publisher, error) {
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
	publisher := publisher{
		Id:       newId,
		Name:     data.Name,
		Owner_id: owner.id,
	}
	return &publisher, nil
}

func publishMessage(owner *clientConnection, data publishMessageRequestData, mongoManager *mongoManager) (bool, error) {

	col := mongoManager.connection.Database("message-broker").Collection("publishers")
	filter := bson.D{primitive.E{Key: "id", Value: data.Publisher_id}, primitive.E{Key: "owner_id", Value: owner.id}}
	findProjection := bson.D{primitive.E{Key: "id", Value: 1}, primitive.E{Key: "name", Value: 1}, primitive.E{Key: "_id", Value: 0}}

	_, err := mongoFindOne(col, findProjection, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, errors.New("publisher not found")
		} else {
			return false, err
		}
	}

	timeToExpire := int64(0)
	if data.Ttl > 0 {
		timeToExpire = time.Now().Unix() + data.Ttl
	}

	_, err = mongoUpdateOne(col, filter, bson.D{
		primitive.E{Key: "$push", Value: bson.D{
			primitive.E{Key: "messages", Value: bson.D{
				primitive.E{Key: "id", Value: uuid.New().String()},
				primitive.E{Key: "payload", Value: data.Payload},
				primitive.E{Key: "date_created", Value: time.Now()},
				primitive.E{Key: "ttl", Value: timeToExpire},
			},
			},
		},
		},
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
