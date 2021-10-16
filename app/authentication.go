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

//response sent to and from the client during authentication
type jSONAuthResponse struct {
	Register bool   `json:"register"`
	Name     string `json:"name"`
	UniqueId string `json:"id"`
}

//authenticate a client connection
func authenticate(client *clientConnection, authChannels errorSuccess) {

	successError := errorSuccess{
		errorChannel:   make(chan error),
		successChannel: make(chan bool),
	}

	//request to send message to client asking them to authenticate
	go client.send(jSONCommunication{
		Action:  "authenticate",
		Message: "Please authenticate",
	}, successError)

	select {
	case err := <-successError.errorChannel: //failed to request authentication
		authChannels.errorChannel <- err
		return
	case <-successError.successChannel: //success!
	}

	var message string

	//attempt to receive message from the client, timeout after 30 seconds
	timeout := time.After(time.Second * 30)
	select {
	case message = <-client.receiveChannel:
	case <-timeout:
		client.send(jSONCommunication{
			Action:  "authentication failed",
			Message: "Authentication timed out",
		}, errorSuccess{})
		authChannels.errorChannel <- errors.New("authentication timed out")
		return
	}

	authResponse := jSONAuthResponse{}
	err := json.Unmarshal([]byte(message), &authResponse) //parse response
	if err != nil {
		//failed parsing response
		client.send(jSONCommunication{
			Action:  "authentication failed",
			Message: "Failed authentication",
		}, errorSuccess{})
		authChannels.errorChannel <- err
		return
	}

	//open mongodb connection
	mongoClient, err := mongoConnect()

	//failed to connect to mongo
	if err != nil {
		fmt.Println(err)
	}

	//setup closing the db connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	defer mongoClient.Disconnect(ctx)

	//open collection containing client details
	col := mongoClient.Database("message-broker").Collection("clients")

	//if creating a new client
	if authResponse.Register {

		//checking if client is already in the collection
		name := authResponse.Name

		filter := bson.D{primitive.E{Key: "name", Value: name}}
		clientResult := bson.M{}

		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		findProjection := bson.D{primitive.E{Key: "id", Value: 1}, primitive.E{Key: "name", Value: 1}, primitive.E{Key: "_id", Value: 0}}
		findOptions := options.FindOne().SetProjection(findProjection)
		err = col.FindOne(ctx, filter, findOptions).Decode(&clientResult)
		if err == mongo.ErrNoDocuments {

			//client is not in the collection so inserting a new record
			id := uuid.New().String()
			_, err := col.InsertOne(ctx, bson.D{primitive.E{Key: "id", Value: id}, primitive.E{Key: "name", Value: name}})
			if err != nil {

				//failed inserting the client record
				client.send(jSONCommunication{
					Action:  "authentication failed",
					Message: "Failed creating client",
				}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
				authChannels.errorChannel <- err
				return
			}

			//set the new ID on the client connection
			client.id = id

			//request to send message to the client with their new details
			client.send(jSONCommunication{
				Action: "authentication successful",
				Data: jSONAuthResponse{
					Register: true,
					Name:     name,
					UniqueId: id,
				},
			}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response

			authChannels.successChannel <- true
			return
		} else if err != nil {
			//failed due to db error
			client.send(jSONCommunication{
				Action:  "authentication failed",
				Message: "Error occurred",
			}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
			authChannels.errorChannel <- err
			return
		}

		//client already exists
		client.send(jSONCommunication{
			Action:  "authentication failed",
			Message: "Client already exists",
		}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
		authChannels.errorChannel <- errors.New("client exists")
	} else {

		//ID supplied so we're going to see if there is a valid client

		id := authResponse.UniqueId

		filter := bson.D{primitive.E{Key: "id", Value: id}}
		clientResult := bson.M{}

		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		findProjection := bson.D{primitive.E{Key: "id", Value: 1}, primitive.E{Key: "name", Value: 1}, primitive.E{Key: "_id", Value: 0}}
		findOptions := options.FindOne().SetProjection(findProjection)
		err = col.FindOne(ctx, filter, findOptions).Decode(&clientResult)

		if err == mongo.ErrNoDocuments {

			//not found
			client.send(jSONCommunication{
				Action:  "authentication failed",
				Message: "Incorrect credentials",
			}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
			authChannels.errorChannel <- errors.New("client not found")
			return
		} else if err != nil {

			//db error
			client.send(jSONCommunication{
				Action:  "authentication failed",
				Message: "Error occurred",
			}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
			authChannels.errorChannel <- err
			return
		}

		//client found

		clientId := clientResult["id"]
		clientName := clientResult["name"]

		//create response for the user with the clients ID and name
		response := jSONAuthResponse{}
		switch v := clientId.(type) {
		case string:
			client.id = v //updating the client connection struct with the clients ID
			response.UniqueId = v
		}
		switch v := clientName.(type) {
		case string:
			client.name = v //updating the client connection struct with the clients name
			response.Name = v
		}

		//request to send success message to the clients
		go client.send(jSONCommunication{
			Action: "authentication successful",
			Data:   response,
		}, successError) //this time we do want to check the response so we're supplying channels
		select {
		case <-successError.errorChannel: //failed to notify the front end :(
			authChannels.errorChannel <- errors.New("failed to notify success")
		case <-successError.successChannel: //success!
			authChannels.successChannel <- true
		}

	}

}
