package main

import (
	"encoding/json"
	"io"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/sberridge/bezmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type registerRequest struct {
	Name string `json:"name"`
}
type registerSuccessResponse struct {
	Success bool             `json:"success"`
	Row     registerResponse `json:"row"`
}
type registerResponse struct {
	Id string `json:"id"`
}

func checkExistingRegistration(collection *mongo.Collection, name string) bool {
	filter := bson.D{{Key: "name", Value: name}}
	result, err := bezmongo.Count(collection, filter)
	return err == nil && result == 0
}

func createRegistration(collection *mongo.Collection, name string) (string, error) {
	id := uuid.New().String()
	_, err := bezmongo.InsertOne(collection, bson.D{{Key: "_id", Value: id}, {Key: "name", Value: name}})
	if err != nil {
		return "", err
	}
	return id, nil
}

func handleRegistration(body io.ReadCloser, mongo *bezmongo.MongoService, session *sessions.Session) []byte {
	registrationFailedMessage := "registration failed"
	bytes, err := readBody(body)
	if err != nil {
		return createMessageResponse(false, registrationFailedMessage)
	}
	requestBody := registerRequest{}
	err = json.Unmarshal(bytes, &requestBody)

	if err != nil {
		return createMessageResponse(false, registrationFailedMessage)
	}

	col := mongo.OpenCollection("message-broker", "clients")

	if !checkExistingRegistration(col, requestBody.Name) {
		return createMessageResponse(false, registrationFailedMessage)
	}

	id, err := createRegistration(col, requestBody.Name)

	if err != nil {
		return createMessageResponse(false, registrationFailedMessage)
	}

	res, _ := json.Marshal(registerSuccessResponse{
		Success: true,
		Row: registerResponse{
			Id: id,
		},
	})
	return res
}
