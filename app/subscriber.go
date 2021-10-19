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
	id                   string
	publisherId          string
	clientId             string
	waitingConfirmation  []string //potential issue here
	cancelChannel        chan bool
	cancelConfirmChannel chan bool
	messagesChannel      chan []messageItem
	confirmedChannel     chan string
}

type messageItem struct {
	Id             string `json:"id"`
	PublisherId    string `json:"publisher_id"`
	SubscriptionId string `json:"subscription_id"`
	Payload        string `json:"payload"`
}

func (sub *subscription) confirmLoop(mongoManager *mongoManager) {
	closed := false
	for {
		select {
		case confirmedId := <-sub.confirmedChannel:
			collection := mongoManager.connection.Database("message-broker").Collection("publisher_messages")
			filter := bson.D{
				primitive.E{Key: "_id", Value: confirmedId},
			}
			update := bson.D{
				primitive.E{Key: "received_by", Value: bson.D{
					primitive.E{Key: "$push", Value: sub.clientId},
				}},
			}
			_, err := mongoUpdateOne(collection, filter, update)

			if err == nil {
				for i, v := range sub.waitingConfirmation {
					if v == confirmedId {
						sub.waitingConfirmation[i] = sub.waitingConfirmation[len(sub.waitingConfirmation)-1]
						sub.waitingConfirmation = sub.waitingConfirmation[:len(sub.waitingConfirmation)-1]
						break
					}
				}
			}

		case <-sub.cancelConfirmChannel:
			closed = true
		}
		if closed {
			break
		}
	}
}

func (sub *subscription) loop(mongoManager *mongoManager) {
	closed := false
	for {
		collection := mongoManager.connection.Database("message-broker").Collection("publisher_messages")
		filter := bson.D{
			primitive.E{Key: "publisher_id", Value: sub.publisherId},
			primitive.E{Key: "received_by", Value: bson.D{
				primitive.E{Key: "$nin", Value: []string{sub.clientId}},
			}},
		}
		if len(sub.waitingConfirmation) > 0 {
			filter = append(filter, primitive.E{Key: "_id", Value: bson.D{
				primitive.E{Key: "$nin", Value: sub.waitingConfirmation},
			}})
		}

		projection := bson.D{
			primitive.E{Key: "publisher_id", Value: 1},
			primitive.E{Key: "payload", Value: 1},
			primitive.E{Key: "date_created", Value: 1},
			primitive.E{Key: "ttl", Value: 1},
		}
		results, err := mongoFindMany(collection, options.Find().SetProjection(projection).SetSort(bson.D{primitive.E{Key: "date_created", Value: 1}}).SetLimit(10), filter)
		messages := []messageItem{}
		if err != nil {
			fmt.Println(err.Error())
			//todo: error logging?
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			for {
				hasRow := results.Next(ctx)
				if !hasRow {
					break
				}
				row := results.Current
				msg := messageItem{
					Id:             row.Lookup("_id").String(),
					PublisherId:    row.Lookup("publisher_id").String(),
					Payload:        row.Lookup("payload").String(),
					SubscriptionId: sub.id,
				}
				sub.waitingConfirmation = append(sub.waitingConfirmation, msg.Id)

				messages = append(messages, msg)
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
		<-time.After(time.Second * 5)
	}

}

func subscribe(owner *clientConnection, mongoManager *mongoManager, publisherId string) (*subscription, error) {
	col := mongoManager.connection.Database("message-broker").Collection("clients")
	filter := bson.D{primitive.E{Key: "id", Value: owner.id},
		primitive.E{Key: "subscriptions.publisher_id", Value: publisherId},
	}
	result, _ := mongoCount(col, filter)

	if result > 0 {
		return nil, errors.New("already subscribed")
	}

	filter = bson.D{primitive.E{Key: "_id", Value: owner.id}}
	id := uuid.New().String()
	update := bson.D{primitive.E{Key: "$push", Value: bson.D{
		primitive.E{Key: "subscriptions", Value: bson.D{
			primitive.E{Key: "_id", Value: id},
			primitive.E{Key: "publisher_id", Value: publisherId},
		}},
	}}}

	_, err := mongoUpdateOne(col, filter, update)

	if err != nil {
		return nil, err
	}

	sub := subscription{
		id:                   id,
		publisherId:          publisherId,
		clientId:             owner.id,
		cancelChannel:        make(chan bool),
		confirmedChannel:     make(chan string),
		cancelConfirmChannel: make(chan bool),
		messagesChannel:      make(chan []messageItem),
	}
	return &sub, nil

}
