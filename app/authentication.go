package main

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

//response sent to and from the client during authentication
type jsonAuthResponse struct {
	Register bool   `json:"register"`
	Name     string `json:"name"`
	UniqueId string `json:"id"`
}

type bsonSubscription struct {
	Id          string `bson:"_id"`
	PublisherId string `bson:"publisher_id"`
}
type bSONClient struct {
	Id            string             `bson:"_id"`
	Name          string             `bson:"name"`
	Subscriptions []bsonSubscription `bson:"subscriptions"`
}

func requestAuthentication(client *clientConnection) (bool, error) {
	successError := errorSuccess{
		errorChannel:   make(chan error),
		successChannel: make(chan bool),
	}

	//request to send message to client asking them to authenticate
	go client.send(jsonCommunication{
		Action:  "authenticate",
		Message: "Please authenticate",
	}, successError)

	select {
	case err := <-successError.errorChannel: //failed to request authentication
		return false, err
	case <-successError.successChannel: //success!
		return true, nil
	}
}

func getClientAuthenticationResponse(client *clientConnection) (*jsonAuthResponse, error) {
	var message string

	//attempt to receive message from the client, timeout after 30 seconds
	timeout := time.After(time.Second * 30)
	select {
	case message = <-client.receiveChannel:
	case <-timeout:
		client.send(jsonCommunication{
			Action:  "authentication_failed",
			Message: "Authentication timed out",
		}, errorSuccess{})
		return nil, errors.New("authentication timed out")
	}
	authResponse := jsonAuthResponse{}
	err := json.Unmarshal([]byte(message), &authResponse) //parse response
	if err != nil {
		//failed parsing response
		client.send(jsonCommunication{
			Action:  "authentication_failed",
			Message: "Failed authentication",
		}, errorSuccess{})
		return nil, err
	}
	return &authResponse, nil
}

func registerNewClient(client *clientConnection, mongoManager *mongoManager, name string) (*bSONClient, error) {
	//open collection containing client details
	col := mongoManager.openCollection("message-broker", "clients")
	filter := bson.D{{Key: "name", Value: name}}
	num, err := mongoCount(col, filter)

	if num == 0 {

		//client is not in the collection so inserting a new record
		id := uuid.New().String()
		_, err := mongoInsertOne(col, bson.D{{Key: "_id", Value: id}, {Key: "name", Value: name}})
		if err != nil {
			//failed inserting the client record
			client.send(jsonCommunication{
				Action:  "authentication_failed",
				Message: "Failed creating client",
			}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
			return nil, err
		}

		//set the new ID on the client connection
		client.id = id

		//request to send message to the client with their new details
		client.send(jsonCommunication{
			Action: "authentication_successful",
			Data: jsonAuthResponse{
				Register: true,
				Name:     name,
				UniqueId: id,
			},
		}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response

		return &bSONClient{
			Id:   id,
			Name: client.name,
		}, nil
	} else if err != nil {
		//failed due to db error
		client.send(jsonCommunication{
			Action:  "authentication_failed",
			Message: "Error occurred",
		}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
		return nil, err
	}

	//client already exists
	client.send(jsonCommunication{
		Action:  "authentication_failed",
		Message: "Client already exists",
	}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
	return nil, errors.New("client exists")
}

//authenticate a client connection
func authenticate(client *clientConnection, mongoManager *mongoManager) (*bSONClient, error) {

	_, err := requestAuthentication(client)
	if err != nil {
		return nil, err
	}

	authResponse, err := getClientAuthenticationResponse(client)
	if err != nil {
		return nil, err
	}

	//open collection containing client details
	col := mongoManager.openCollection("message-broker", "clients")

	//if creating a new client
	if authResponse.Register {
		return registerNewClient(client, mongoManager, authResponse.Name)
	} else {

		//ID supplied so we're going to see if there is a valid client

		id := authResponse.UniqueId

		filter := bson.D{{Key: "_id", Value: id}}
		findProjection := bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: 1}, {Key: "subscriptions", Value: 1}}
		result := mongoFindOne(col, findProjection, filter)
		clientStruct := bSONClient{}
		err = result.Decode(&clientStruct)
		if err == mongo.ErrNoDocuments {

			//not found
			client.send(jsonCommunication{
				Action:  "authentication_failed",
				Message: "Incorrect credentials",
			}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
			return nil, errors.New("client not found")
		} else if err != nil {

			//db error
			client.send(jsonCommunication{
				Action:  "authentication_failed",
				Message: "Error occurred",
			}, errorSuccess{}) //not supplying any channels for error/success since we don't really need to block here to check the response
			return nil, err
		}

		//client found

		clientId := clientStruct.Id
		clientName := clientStruct.Name

		client.id = clientId
		client.name = clientName

		//create response for the user with the clients ID and name
		response := jsonAuthResponse{
			UniqueId: clientId,
			Name:     clientName,
		}
		successError := errorSuccess{
			errorChannel:   make(chan error),
			successChannel: make(chan bool),
		}

		//request to send success message to the clients
		go client.send(jsonCommunication{
			Action: "authentication_successful",
			Data:   response,
		}, successError) //this time we do want to check the response so we're supplying channels
		select {
		case <-successError.errorChannel: //failed to notify the front end :(
			return nil, errors.New("failed to notify success")
		case <-successError.successChannel: //success!
			return &clientStruct, nil
		}

	}

}
