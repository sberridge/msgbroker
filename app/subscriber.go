package main

import (
	"errors"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type subscription struct {
	id              string
	publisherId     string
	clientId        string
	cancelChannel   chan bool
	messagesChannel chan []publishMessageRequestData
}

func (sub *subscription) loop(mongoManager *mongoManager) {
	closed := false
	for {
		//collection := mongoManager.connection.Database("message-broker").Collection("clients")

		messages := []publishMessageRequestData{}

		select {
		case sub.messagesChannel <- messages:
		case <-sub.cancelChannel:
			closed = true
		}
		if closed {
			break
		}
	}

}

func subscribe(owner *clientConnection, mongoManager *mongoManager, publisherId string) (*subscription, error) {
	col := mongoManager.connection.Database("message-broker").Collection("clients")
	filter := bson.D{primitive.E{Key: "id", Value: owner.id},
		primitive.E{Key: "subscriptions.publisher_id", Value: publisherId},
	}
	result, _ := mongoFindOne(col, bson.D{primitive.E{Key: "id", Value: 1}}, filter)

	if _, e := result["id"]; e {
		return nil, errors.New("already subscribed")
	}

	filter = bson.D{primitive.E{Key: "id", Value: owner.id}}
	id := uuid.New().String()
	update := bson.D{primitive.E{Key: "$push", Value: bson.D{
		primitive.E{Key: "subscriptions", Value: bson.D{
			primitive.E{Key: "id", Value: id},
			primitive.E{Key: "publisher_id", Value: publisherId},
		}},
	}}}

	_, err := mongoUpdateOne(col, filter, update)

	if err != nil {
		return nil, err
	}

	sub := subscription{
		id:              id,
		publisherId:     publisherId,
		clientId:        owner.id,
		cancelChannel:   make(chan bool),
		messagesChannel: make(chan []publishMessageRequestData),
	}
	return &sub, nil

}
