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

func handleAuth(body io.ReadCloser, bmongo *bezmongo.MongoService, session *sessions.Session, responseChannel chan []byte) {

	bytes, err := readBody(body)
	failedAuthMessage := "Authentication failed"
	if err != nil {
		fmt.Println(err)
		responseChannel <- createMessageResponse(false, failedAuthMessage)
		return
	}

	requestBody := authRequest{}

	err = json.Unmarshal(bytes, &requestBody)
	if err != nil {
		fmt.Println(err)
		responseChannel <- createMessageResponse(false, failedAuthMessage)
		return
	}

	col := bmongo.OpenCollection("message-broker", "clients")
	id := requestBody.UniqueId

	filter := bson.D{{Key: "_id", Value: id}}
	findProjection := bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: 1}, {Key: "subscriptions", Value: 1}}
	result := bezmongo.FindOne(col, findProjection, filter)
	clientStruct := bsonClient{}
	err = result.Decode(&clientStruct)
	if err != nil {
		responseChannel <- createMessageResponse(false, failedAuthMessage)
		return
	}
	session.Values["auth_id"] = clientStruct.Id
	responseChannel <- createMessageResponse(true, fmt.Sprintf("Authenticated as %s", clientStruct.Name))
}
