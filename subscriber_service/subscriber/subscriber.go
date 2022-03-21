package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/sberridge/bezmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const publisherCollection = "publishers"

func handleGetSubscriptions(id string, mongo *bezmongo.MongoService) []byte {
	collection := mongo.OpenCollection(messageBrokerDb, clientsCollection)
	filter := bson.D{{Key: "_id", Value: id}}
	projection := bson.D{{Key: "subscriptions", Value: 1}}
	res := bezmongo.FindOne(collection, projection, filter)
	fmt.Println(res)
	return createMessageResponse(true, "")
}

func checkPublisherExists(collection *mongo.Collection, id string) bool {
	filter := bson.D{{Key: "_id", Value: id}}
	count, err := bezmongo.Count(collection, filter)
	if err != nil {
		return true
	}
	return count > 0
}

type subscribeRequest struct {
	PublisherID string `json:"publisher_id"`
}

func handleSubscribe(body io.ReadCloser, id string, mongo *bezmongo.MongoService) []byte {
	failMessage := "failed to subscribe"
	bytes, err := readBody(body)
	if err != nil {
		return createMessageResponse(false, failMessage)
	}
	request := subscribeRequest{}
	err = json.Unmarshal(bytes, &request)
	if err != nil {
		return createMessageResponse(false, failMessage)
	}

	pubCollection := mongo.OpenCollection(messageBrokerDb, publisherCollection)
	if !checkPublisherExists(pubCollection, request.PublisherID) {
		return createMessageResponse(false, failMessage)
	}

	clientCollection := mongo.OpenCollection(messageBrokerDb, clientsCollection)

	filter := bson.D{{Key: "_id", Value: id},
		{Key: "subscriptions.publisher_id", Value: request.PublisherID},
	}
	result, _ := bezmongo.Count(clientCollection, filter)

	if result > 0 {
		return createMessageResponse(false, "already subscribed")
	}

	filter = bson.D{{Key: "_id", Value: id}}
	subid := uuid.New().String()
	update := bson.D{{Key: "$push", Value: bson.D{
		{Key: "subscriptions", Value: bson.D{
			{Key: "_id", Value: subid},
			{Key: "publisher_id", Value: request.PublisherID},
		}},
	}}}

	_, err = bezmongo.UpdateOne(clientCollection, filter, update)

	if err != nil {
		return createMessageResponse(false, failMessage)
	}

	return createMessageResponse(true, "subscribed")
}
