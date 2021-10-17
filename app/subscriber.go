package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type subscription struct {
	id              string
	publisherId     string
	clientId        string
	cancelChannel   chan bool
	messagesChannel chan []messageItem
}

type messageItem struct {
	Id          string `json:"id"`
	PublisherId string `json:"publisher_id"`
	Payload     string `json:"payload"`
}

func (sub *subscription) loop(mongoManager *mongoManager) {
	closed := false
	for {
		collection := mongoManager.connection.Database("message-broker").Collection("publisher_messages")
		filter := bson.D{primitive.E{Key: "publisher_id", Value: sub.publisherId}}
		projection := bson.D{
			primitive.E{Key: "publisher_id", Value: 1},
			primitive.E{Key: "payload", Value: 1},
			primitive.E{Key: "date_created", Value: 0},
			primitive.E{Key: "ttl", Value: 1},
		}
		results, err := mongoFindMany(collection, options.Find().SetProjection(projection).SetSort(bson.D{primitive.E{Key: "date_created", Value: 1}}).SetLimit(10), filter)
		fmt.Println(results)
		messages := []messageItem{}
		if err != nil {
			fmt.Println(err.Error())
			//todo: error logging?
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			for {
				row := results.Current
				messages = append(messages, messageItem{
					Id:          row.Lookup("id").String(),
					PublisherId: row.Lookup("publisher_id").String(),
					Payload:     row.Lookup("payload").String(),
				})
				if !results.Next(ctx) {
					break
				}
			}
		}

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
		messagesChannel: make(chan []messageItem),
	}
	return &sub, nil

}
