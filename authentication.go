package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type clientDoc struct {
	id   string
	name string
}

func authenticate(con *websocket.Conn, authSuccessChan chan *jSONAuthResponse, authErrorChan chan error) {

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
	if authResponse.Register {
		name := authResponse.Name
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
		if err != nil {
			fmt.Println(err)
		}
		defer client.Disconnect(ctx)
		col := client.Database("message-broker").Collection("clients")
		filter := bson.D{primitive.E{Key: "name", Value: name}}
		clientResult := clientDoc{}

		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err = col.FindOne(ctx, filter).Decode(&clientResult)

		if err == mongo.ErrNoDocuments {
			fmt.Println("no results yo")
			id := uuid.New().String()
			res, err := col.InsertOne(ctx, bson.D{primitive.E{Key: "id", Value: id}, primitive.E{Key: "name", Value: name}})
			if err != nil {
				fmt.Println(err.Error())
				authErrorChan <- err
				return
			}
			fmt.Println(res)
		} else if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println(string(message))
}
