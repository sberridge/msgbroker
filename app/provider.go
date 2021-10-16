package main

import (
	"errors"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type provider struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Owner_id string `json:"owner_id"`
}

func checkProviderExists(name string, mongoManager *mongoManager) (bool, error) {

	//open collection containing provider details
	col := mongoManager.connection.Database("message-broker").Collection("providers")
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

func registerProvider(owner *clientConnection, data newProviderData, mongoManager *mongoManager) (newId string, err error) {

	//open collection containing provider details
	col := mongoManager.connection.Database("message-broker").Collection("providers")

	newId = uuid.New().String()
	_, err = mongoInsertOne(col, bson.D{primitive.E{Key: "id", Value: newId}, primitive.E{Key: "name", Value: data.Name}, primitive.E{Key: "owner_id", Value: owner.id}})

	if err != nil {
		return "", err
	}
	return
}

func newProvider(owner *clientConnection, data newProviderData, mongoManager *mongoManager) (*provider, error) {
	exists, err := checkProviderExists(data.Name, mongoManager)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("provider exists")
	}
	newId, err := registerProvider(owner, data, mongoManager)
	if err != nil {
		return nil, err
	}
	provider := provider{
		Id:       newId,
		Name:     data.Name,
		Owner_id: owner.id,
	}
	return &provider, nil
}
