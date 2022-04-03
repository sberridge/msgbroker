package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gorilla/sessions"
	"github.com/sberridge/bezmongo"
	"go.mongodb.org/mongo-driver/bson"
)

type authRequest struct {
	UniqueId string `json:"id"`
}

type bsonClient struct {
	Id   string `bson:"_id"`
	Name string `bson:"name"`
}

const clientsCollection = "clients"

type authResponseData struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
type authResponse struct {
	Success bool             `json:"success"`
	Data    authResponseData `json:"data"`
}

func handleAuth(body io.ReadCloser, bmongo *bezmongo.MongoService, session *sessions.Session) []byte {

	bytes, err := readBody(body)
	failedAuthMessage := "Authentication failed"
	if err != nil {
		fmt.Println(err)
		return createMessageResponse(false, failedAuthMessage)

	}

	requestBody := authRequest{}

	err = json.Unmarshal(bytes, &requestBody)
	if err != nil {
		fmt.Println(err)
		return createMessageResponse(false, failedAuthMessage)
	}

	id := requestBody.UniqueId

	clientStruct, err := getClient(id, bmongo)

	if err != nil {
		return createMessageResponse(false, failedAuthMessage)
	}
	response, err := json.Marshal(authResponse{
		Success: true,
		Data: authResponseData{
			Id:   clientStruct.Id,
			Name: clientStruct.Name,
		},
	})
	if err != nil {
		return createMessageResponse(false, failedAuthMessage)
	}
	session.Values["auth_id"] = clientStruct.Id
	return response
}

type checkAuthResponseData struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
type checkAuthResponse struct {
	Success bool                  `json:"success"`
	Data    checkAuthResponseData `json:"data"`
}

func getClient(clientId string, mongo *bezmongo.MongoService) (bsonClient, error) {
	col := mongo.OpenCollection(messageBrokerDb, clientsCollection)
	filter := bson.D{{Key: "_id", Value: clientId}}
	findProjection := bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: 1}}
	result := bezmongo.FindOne(col, findProjection, filter)
	clientStruct := bsonClient{}
	err := result.Decode(&clientStruct)
	if err != nil {
		return bsonClient{}, err
	}
	return clientStruct, nil
}

func handleCheckAuth(ses *sessions.Session, mongo *bezmongo.MongoService) []byte {
	id, authed := checkAuth(ses)
	if authed {
		client, err := getClient(id, mongo)
		if err != nil {
			return createMessageResponse(false, "failed checking auth")
		}
		response, err := json.Marshal(checkAuthResponse{
			Success: true,
			Data: checkAuthResponseData{
				Id:   id,
				Name: client.Name,
			},
		})
		if err != nil {
			return createMessageResponse(false, "failed checking auth")
		}
		return response
	}
	return createMessageResponse(false, "not authed")
}
