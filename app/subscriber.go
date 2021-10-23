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
	id                      string
	publisherId             string
	clientId                string
	cancelChannel           chan bool
	messagesChannel         chan []messageItem
	receiveConfirmedChannel chan *subscriptionMessagesConfirmation
}

type messageItem struct {
	Id             string `json:"id"`
	PublisherId    string `json:"publisher_id"`
	SubscriptionId string `json:"subscription_id"`
	Payload        string `json:"payload"`
}

type bsonMessage struct {
	Id             string `bson:"_id"`
	PublisherId    string `bson:"publisher_id"`
	SubscriptionId string `bson:"subscription_id"`
	Payload        string `bson:"payload"`
}

type subscriptionMessagesConfirmation struct {
	messages         []string
	confirmedChannel chan int
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
			bsonMessages := []bsonMessage{}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			results.All(ctx, &bsonMessages)
			for _, message := range bsonMessages {
				messageItem := messageItem(message)
				messageItem.SubscriptionId = sub.id
				messages = append(messages, messageItem)
			}
		}
		if len(messages) > 0 {
			select {
			case sub.messagesChannel <- messages:
			case <-sub.cancelChannel:
				closed = true
			}
			if closed {
				break
			}
			confirmation := <-sub.receiveConfirmedChannel

			collection := mongoManager.connection.Database("message-broker").Collection("publisher_messages")
			filter := bson.D{
				primitive.E{Key: "_id", Value: bson.D{
					primitive.E{Key: "$in", Value: confirmation.messages},
				}},
			}
			update := bson.D{
				primitive.E{Key: "$push", Value: bson.D{
					primitive.E{Key: "received_by", Value: sub.clientId},
				}},
			}
			res, err := mongoUpdateMany(collection, filter, update)
			if err != nil {
				confirmation.confirmedChannel <- 0
			} else {
				confirmation.confirmedChannel <- int(res.ModifiedCount)
			}

		}

		<-time.After(time.Second * 5)
	}

}

func subscribe(owner *clientConnection, mongoManager *mongoManager, publisherId string) (*subscription, error) {

	publisherCol := mongoManager.connection.Database("message-broker").Collection("publishers")
	publisherFilter := bson.D{primitive.E{Key: "_id", Value: publisherId}}
	result, _ := mongoCount(publisherCol, publisherFilter)
	if result == 0 {
		return nil, errors.New("publisher not found")
	}

	col := mongoManager.connection.Database("message-broker").Collection("clients")
	filter := bson.D{primitive.E{Key: "_id", Value: owner.id},
		primitive.E{Key: "subscriptions.publisher_id", Value: publisherId},
	}
	result, _ = mongoCount(col, filter)

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
		id:                      id,
		publisherId:             publisherId,
		clientId:                owner.id,
		cancelChannel:           make(chan bool),
		receiveConfirmedChannel: make(chan *subscriptionMessagesConfirmation),
		messagesChannel:         make(chan []messageItem),
	}
	return &sub, nil

}
