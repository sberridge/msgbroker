package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type subscription struct {
	id                      string
	publisherID             string
	clientID                string
	cancelChannel           chan bool
	messagesChannel         chan []jsonMessageItem
	receiveConfirmedChannel chan *subscriptionMessagesConfirmation
}

type jsonMessageItem struct {
	Id             string `json:"id"`
	PublisherID    string `json:"publisher_id"`
	SubscriptionID string `json:"subscription_id"`
	Payload        string `json:"payload"`
}

type bsonMessage struct {
	Id             string `bson:"_id"`
	PublisherID    string `bson:"publisher_id"`
	SubscriptionID string `bson:"subscription_id"`
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
			{Key: "publisher_id", Value: sub.publisherID},
			{Key: "received_by", Value: bson.D{
				{Key: "$nin", Value: []string{sub.clientID}},
			}},
		}

		projection := bson.D{
			{Key: "publisher_id", Value: 1},
			{Key: "payload", Value: 1},
			{Key: "date_created", Value: 1},
			{Key: "ttl", Value: 1},
		}
		results, err := mongoFindMany(collection, options.Find().SetProjection(projection).SetSort(bson.D{{Key: "date_created", Value: 1}}).SetLimit(10), filter)
		messages := []jsonMessageItem{}
		if err != nil {
			fmt.Println(err.Error())
			//todo: error logging?
		} else {
			bsonMessages := []bsonMessage{}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			results.All(ctx, &bsonMessages)
			for _, message := range bsonMessages {
				messageItem := jsonMessageItem(message)
				messageItem.SubscriptionID = sub.id
				messages = append(messages, messageItem)
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
		if len(messages) > 0 {
			var confirmation *subscriptionMessagesConfirmation
			select {
			case confirmation = <-sub.receiveConfirmedChannel:
			case <-sub.cancelChannel:
				closed = true
			}
			if closed {
				break
			}

			collection := mongoManager.connection.Database("message-broker").Collection("publisher_messages")
			filter := bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "$in", Value: confirmation.messages},
				}},
			}
			update := bson.D{
				{Key: "$push", Value: bson.D{
					{Key: "received_by", Value: sub.clientID},
				}},
			}
			res, err := mongoUpdateMany(collection, filter, update)
			confirmed := 0
			if err == nil {
				confirmed = int(res.ModifiedCount)
			}
			confirmation.confirmedChannel <- confirmed

		}

		<-time.After(time.Second * 2)
	}

}

func subscribe(owner *clientConnection, mongoManager *mongoManager, publisherID string) (*subscription, error) {

	publisherCol := mongoManager.connection.Database("message-broker").Collection("publishers")
	publisherFilter := bson.D{{Key: "_id", Value: publisherID}}
	result, _ := mongoCount(publisherCol, publisherFilter)
	if result == 0 {
		return nil, errors.New("publisher not found")
	}

	col := mongoManager.connection.Database("message-broker").Collection("clients")
	filter := bson.D{{Key: "_id", Value: owner.id},
		{Key: "subscriptions.publisher_id", Value: publisherID},
	}
	result, _ = mongoCount(col, filter)

	if result > 0 {
		return nil, errors.New("already subscribed")
	}

	filter = bson.D{{Key: "_id", Value: owner.id}}
	id := uuid.New().String()
	update := bson.D{{Key: "$push", Value: bson.D{
		{Key: "subscriptions", Value: bson.D{
			{Key: "_id", Value: id},
			{Key: "publisher_id", Value: publisherID},
		}},
	}}}

	_, err := mongoUpdateOne(col, filter, update)

	if err != nil {
		return nil, err
	}

	sub := subscription{
		id:                      id,
		publisherID:             publisherID,
		clientID:                owner.id,
		cancelChannel:           make(chan bool),
		receiveConfirmedChannel: make(chan *subscriptionMessagesConfirmation),
		messagesChannel:         make(chan []jsonMessageItem),
	}
	return &sub, nil

}
