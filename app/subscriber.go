package main

import (
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type subscription struct {
	id              string
	publisherId     string
	clientId        string
	cancelChannel   chan bool
	messagesChannel chan []publishMessageRequestData
}

func (sub *subscription) loop(*mongoManager) {
	closed := false
	for {
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
		primitive.E{Key: "subscriptions", Value: bson.D{
			primitive.E{Key: "publisher_id", Value: publisherId},
		},
		},
	}
	_, err := mongoFindOne(col, bson.D{}, filter)
	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	filter = bson.D{primitive.E{Key: "id", Value: owner.id}}
	id := uuid.New().String()
	update := bson.D{primitive.E{Key: "$push", Value: bson.D{
		primitive.E{Key: "subscriptions", Value: bson.D{
			primitive.E{Key: "id", Value: id},
			primitive.E{Key: "publisher_id", Value: publisherId},
		}},
	}}}

	_, err = mongoUpdateOne(col, filter, update)

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
