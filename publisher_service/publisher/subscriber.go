package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/sberridge/bezmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type bsonClientSubscriptions struct {
	Id            string `bson:"_id"`
	Name          string `bson:"name"`
	Subscriptions []struct {
		Id          string `bson:"_id"`
		PublisherId string `bson:"publisher_id"`
	} `bson:"subscriptions"`
	Publishers []bsonPublisher
}
type jsonSubscriptionResultPublisher struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	OwnerID string `json:"owner_id"`
}
type jsonSubscriptionResult struct {
	Id        string                          `json:"id"`
	Publisher jsonSubscriptionResultPublisher `json:"publisher"`
}
type subscriptionsResult struct {
	Success       bool                     `json:"success"`
	Subscriptions []jsonSubscriptionResult `json:"subscriptions"`
}

func findPublisherInList(id string, publishers []bsonPublisher) (bool, bsonPublisher) {
	for _, publisher := range publishers {
		if publisher.Id == id {
			return true, publisher
		}
	}
	return false, bsonPublisher{}
}

func handleGetSubscriptions(id string, mongo *bezmongo.MongoService) []byte {
	fmt.Println(id)
	collection := mongo.OpenCollection(messageBrokerDb, clientsCollection)
	filter := bson.D{{Key: "$match", Value: bson.D{
		{Key: "_id", Value: id},
	}}}
	lookup := bson.D{{Key: "$lookup", Value: bson.D{
		{Key: "from", Value: "publishers"},
		{Key: "localField", Value: "subscriptions.publisher_id"},
		{Key: "foreignField", Value: "_id"},
		{Key: "as", Value: "publishers"},
	}}}
	stages := []bson.D{
		filter,
		lookup,
	}
	res, err := bezmongo.Aggregate(collection, stages)
	if err != nil {
		return createMessageResponse(false, "failed fetching subscriptions")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	bResults := []bsonClientSubscriptions{}

	if res.RemainingBatchLength() == 0 {
		return createMessageResponse(false, "failed fetching subscriptions")
	}
	res.All(ctx, &bResults)

	subscriptions := []jsonSubscriptionResult{}
	for _, subscription := range bResults[0].Subscriptions {
		found, publisherDetails := findPublisherInList(subscription.PublisherId, bResults[0].Publishers)
		if !found {
			continue
		}
		subscriptions = append(subscriptions, jsonSubscriptionResult{
			Id:        subscription.Id,
			Publisher: jsonSubscriptionResultPublisher(publisherDetails),
		})
	}

	result, err := json.Marshal(subscriptionsResult{
		Success:       true,
		Subscriptions: subscriptions,
	})
	if err != nil {
		return createMessageResponse(false, "failed fetching subscriptions")
	}
	return result
}

func checkPublisherIDExists(collection *mongo.Collection, id string) bool {
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
	if !checkPublisherIDExists(pubCollection, request.PublisherID) {
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

func checkOwnsSubscription(subscriptionId string, ownerId string, mongo *bezmongo.MongoService) (bool, error) {
	collection := mongo.OpenCollection(messageBrokerDb, clientsCollection)
	filter := bson.D{
		{Key: "_id", Value: ownerId},
		{Key: "subscriptions", Value: bson.D{
			{Key: "$elemMatch", Value: bson.D{
				{Key: "_id", Value: subscriptionId},
			}},
		}},
	}
	count, err := bezmongo.Count(collection, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func unsubscribe(subscriptionId string, ownerId string, mongo *bezmongo.MongoService) (bool, error) {
	collection := mongo.OpenCollection(messageBrokerDb, clientsCollection)
	filter := bson.D{
		{Key: "_id", Value: ownerId},
	}
	update := bson.D{
		{Key: "$pull", Value: bson.D{
			{Key: "subscriptions", Value: bson.D{
				{Key: "_id", Value: subscriptionId},
			}},
		}},
	}
	count, err := bezmongo.UpdateOne(collection, filter, update)
	if err != nil {
		return false, err
	}
	return count.ModifiedCount > 0, nil
}

func handleDeleteSubscription(subscriptionId string, body io.ReadCloser, id string, mongo *bezmongo.MongoService) []byte {
	deleteSubscriptionFailMessage := "delete subscription failed"
	owned, err := checkOwnsSubscription(subscriptionId, id, mongo)
	if err != nil {
		return createMessageResponse(false, deleteSubscriptionFailMessage)
	}
	if !owned {
		return createMessageResponse(false, "subscription not found")
	}

	unsubscribed, err := unsubscribe(subscriptionId, id, mongo)
	if err != nil || !unsubscribed {
		return createMessageResponse(false, deleteSubscriptionFailMessage)
	}
	return createMessageResponse(true, "unsubscribed")
}
