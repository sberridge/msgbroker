package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func authenticate(con *websocket.Conn, authSuccessChan chan jSONAuthResponse, authErrorChan chan error) {

	errChan := make(chan error)
	successChan := make(chan bool)
	authMessage, _ := json.Marshal(jSONCommunication{
		Action:  "authenticate",
		Message: "Please authenticate",
	})

	go sendMessage(con, string(authMessage), successChan, errChan)

	select {
	case err := <-errChan:
		authErrorChan <- err
		fmt.Printf("errored requesting auth, %s", err.Error())
		return
	case <-successChan:

	}

	_, message, err := con.ReadMessage()
	if err != nil {
		fmt.Printf("errored receiving auth, %s", err.Error())
		return
	}
	authResponse := jSONAuthResponse{}
	err = json.Unmarshal(message, &authResponse)
	if err != nil {
		fmt.Printf("failed reading auth response, %s", err.Error())
		failResponse, _ := json.Marshal(jSONCommunication{
			Action:  "authentication failed",
			Message: "Failed authentication",
		})
		sendMessage(con, string(failResponse), nil, nil)
		authErrorChan <- err
		return
	}
	fmt.Println(authResponse)

	client, err := mongoConnect()

	if err != nil {
		fmt.Println(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	defer client.Disconnect(ctx)
	col := client.Database("message-broker").Collection("clients")

	if authResponse.Register {
		name := authResponse.Name

		filter := bson.D{primitive.E{Key: "name", Value: name}}
		clientResult := bson.M{}

		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		findProjection := bson.D{primitive.E{Key: "id", Value: 1}, primitive.E{Key: "name", Value: 1}, primitive.E{Key: "_id", Value: 0}}
		findOptions := options.FindOne().SetProjection(findProjection)
		err = col.FindOne(ctx, filter, findOptions).Decode(&clientResult)
		if err == mongo.ErrNoDocuments {
			id := uuid.New().String()
			_, err := col.InsertOne(ctx, bson.D{primitive.E{Key: "id", Value: id}, primitive.E{Key: "name", Value: name}})
			if err != nil {
				fmt.Println(err.Error())
				authErrorChan <- err
				return
			}
			response := jSONAuthResponse{
				Register: true,
				Name:     name,
				UniqueId: id,
			}
			authSuccessChan <- response
			return
		} else if err != nil {
			authErrorChan <- err
			return
		}
		authErrorChan <- errors.New("client exists")
	} else {
		id := authResponse.UniqueId

		filter := bson.D{primitive.E{Key: "id", Value: id}}
		clientResult := bson.M{}

		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		findProjection := bson.D{primitive.E{Key: "id", Value: 1}, primitive.E{Key: "name", Value: 1}, primitive.E{Key: "_id", Value: 0}}
		findOptions := options.FindOne().SetProjection(findProjection)
		err = col.FindOne(ctx, filter, findOptions).Decode(&clientResult)

		if err == mongo.ErrNoDocuments {
			authErrorChan <- errors.New("client not found")
			return
		} else if err != nil {
			authErrorChan <- err
			return
		}
		clientId := clientResult["id"]
		clientName := clientResult["name"]

		response := jSONAuthResponse{}
		switch v := clientId.(type) {
		case string:
			response.UniqueId = v
		}
		switch v := clientName.(type) {
		case string:
			response.Name = v
		}
		authSuccessChan <- response
	}

}
