package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func authenticate(client *clientConnection, authSuccessChan chan bool, authErrorChan chan error) {

	authMessage := jSONCommunication{
		Action:  "authenticate",
		Message: "Please authenticate",
	}

	successError := errorSuccess{
		errorChannel:   make(chan error),
		successChannel: make(chan bool),
	}

	go client.send(authMessage, successError)

	select {
	case err := <-successError.errorChannel:
		authErrorChan <- err
		return
	case <-successError.successChannel:

	}
	var message string
	timeout := time.After(time.Second * 30)
	select {
	case message = <-client.receiveChannel:
	case <-timeout:
		client.send(jSONCommunication{
			Action:  "authentication failed",
			Message: "Authentication timed out",
		}, errorSuccess{})
		authErrorChan <- errors.New("authentication timed out")
		return
	}

	authResponse := jSONAuthResponse{}
	err := json.Unmarshal([]byte(message), &authResponse)
	if err != nil {
		client.send(jSONCommunication{
			Action:  "authentication failed",
			Message: "Failed authentication",
		}, errorSuccess{})
		authErrorChan <- err
		return
	}

	mongoClient, err := mongoConnect()

	if err != nil {
		fmt.Println(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	defer mongoClient.Disconnect(ctx)
	col := mongoClient.Database("message-broker").Collection("clients")

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
				client.send(jSONCommunication{
					Action:  "authentication failed",
					Message: "Failed creating client",
				}, errorSuccess{})
				authErrorChan <- err
				return
			}
			client.id = id
			client.send(jSONCommunication{
				Action: "authentication successful",
				Data: jSONAuthResponse{
					Register: true,
					Name:     name,
					UniqueId: id,
				},
			}, errorSuccess{})

			authSuccessChan <- true
			return
		} else if err != nil {
			client.send(jSONCommunication{
				Action:  "authentication failed",
				Message: "Error occurred",
			}, errorSuccess{})
			authErrorChan <- err
			return
		}
		client.send(jSONCommunication{
			Action:  "authentication failed",
			Message: "Client already exists",
		}, errorSuccess{})
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
			client.send(jSONCommunication{
				Action:  "authentication failed",
				Message: "Incorrect credentials",
			}, errorSuccess{})
			authErrorChan <- errors.New("client not found")
			return
		} else if err != nil {
			client.send(jSONCommunication{
				Action:  "authentication failed",
				Message: "Error occurred",
			}, errorSuccess{})
			authErrorChan <- err
			return
		}

		clientId := clientResult["id"]
		clientName := clientResult["name"]

		response := jSONAuthResponse{}
		switch v := clientId.(type) {
		case string:
			client.id = v
			response.UniqueId = v
		}
		switch v := clientName.(type) {
		case string:
			client.name = v
			response.Name = v
		}
		go client.send(jSONCommunication{
			Action: "authentication successful",
			Data:   response,
		}, successError)
		select {
		case <-successError.errorChannel:
			authErrorChan <- errors.New("failed to notify success")
		case <-successError.successChannel:
			authSuccessChan <- true
		}

	}

}
