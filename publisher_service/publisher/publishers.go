package main

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/sberridge/bezmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type bsonPublisher struct {
	Id      string `bson:"_id"`
	Name    string `bson:"name"`
	OwnerID string `bson:"owner_id"`
}
type jsonPublisher struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
type publishersResult struct {
	Success    bool            `json:"success"`
	Publishers []jsonPublisher `json:"publishers"`
}

const publisherCollection = "publishers"

func handleGetPublications(mongo *bezmongo.MongoService, id string, query url.Values) []byte {
	collection := mongo.OpenCollection(messageBrokerDb, publisherCollection)
	filter := bson.D{{Key: "owner_id", Value: id}}
	findProjection := bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: 1}}
	results, err := bezmongo.FindMany(collection, options.Find().SetProjection(findProjection), filter)
	if err != nil {
		return createMessageResponse(false, "failed fetching publications")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bResults := []bsonPublisher{}
	jResults := []jsonPublisher{}
	results.All(ctx, &bResults)
	for _, p := range bResults {
		jResults = append(jResults, jsonPublisher{
			Id:   p.Id,
			Name: p.Name,
		})
	}
	result, err := json.Marshal(publishersResult{
		Success:    true,
		Publishers: jResults,
	})
	if err != nil {
		return createMessageResponse(false, "failed fetching publications")
	}
	return result
}

type createPublisherRequest struct {
	Name string `json:"name"`
}

func checkPublisherExists(collection *mongo.Collection, name string, id string) bool {
	filter := bson.D{{Key: "owner_id", Value: id}, {Key: "name", Value: name}}
	count, err := bezmongo.Count(collection, filter)
	if err != nil {
		return true
	}
	return count > 0
}

func registerPublisher(collection *mongo.Collection, name string, id string) (string, error) {
	newId := uuid.New().String()
	_, err := bezmongo.InsertOne(collection, bson.D{{Key: "_id", Value: newId}, {Key: "name", Value: name}, {Key: "owner_id", Value: id}})

	if err != nil {
		return "", err
	}
	return newId, nil
}

type createPublisherSuccessResponse struct {
	Success bool                    `json:"success"`
	Row     createPublisherResponse `json:"row"`
}
type createPublisherResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func handleCreatePublisher(body io.ReadCloser, id string, mongo *bezmongo.MongoService) []byte {
	publisherFailedMessage := "create publisher failed"
	bytes, err := readBody(body)
	if err != nil {
		return createMessageResponse(false, publisherFailedMessage)
	}

	publisherRequest := createPublisherRequest{}
	err = json.Unmarshal(bytes, &publisherRequest)
	if err != nil {
		return createMessageResponse(false, publisherFailedMessage)
	}

	collection := mongo.OpenCollection(messageBrokerDb, publisherCollection)

	if checkPublisherExists(collection, publisherRequest.Name, id) {
		return createMessageResponse(false, "publisher already exists")
	}

	publisherId, err := registerPublisher(collection, publisherRequest.Name, id)

	if err != nil {
		return createMessageResponse(false, publisherFailedMessage)
	}

	response, err := json.Marshal(createPublisherSuccessResponse{
		Success: true,
		Row: createPublisherResponse{
			Id:   publisherId,
			Name: publisherRequest.Name,
		},
	})

	if err != nil {
		return createMessageResponse(false, publisherFailedMessage)
	}

	return response

}

func checkOwnsPublisher(pubId string, ownerId string, mongo *bezmongo.MongoService) (bool, error) {
	filter := bson.D{{Key: "_id", Value: pubId}, {Key: "owner_id", Value: ownerId}}
	collection := mongo.OpenCollection(messageBrokerDb, publisherCollection)
	count, err := bezmongo.Count(collection, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func deletePublisher(pubId string, mongo *bezmongo.MongoService) (bool, error) {
	filter := bson.D{{Key: "_id", Value: pubId}}
	collection := mongo.OpenCollection(messageBrokerDb, publisherCollection)
	result, err := bezmongo.DeleteMany(collection, filter)
	if err != nil {
		return false, err
	}
	return result.DeletedCount > 0, nil
}

func deleteAllPublisherMessages(pubId string, mongo *bezmongo.MongoService) (int64, error) {
	filter := bson.D{{Key: "publisher_id", Value: pubId}}
	collection := mongo.OpenCollection(messageBrokerDb, "publisher_messages")
	result, err := bezmongo.DeleteMany(collection, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func deleteExistingSubscriptions(pubId string, mongo *bezmongo.MongoService) (int64, error) {
	filter := bson.D{
		{Key: "subscriptions", Value: bson.D{
			{Key: "$elemMatch", Value: bson.D{
				{Key: "publisher_id", Value: pubId},
			}},
		}},
	}
	update := bson.D{{Key: "$pull", Value: bson.D{{Key: "subscriptions", Value: bson.D{{Key: "publisher_id", Value: pubId}}}}}}
	collection := mongo.OpenCollection(messageBrokerDb, "clients")
	result, err := bezmongo.UpdateMany(collection, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func handleDeletePublisher(pubId string, ownerId string, mongo *bezmongo.MongoService) []byte {
	deletePublisherFailedMessage := "delete publisher failed"
	owned, err := checkOwnsPublisher(pubId, ownerId, mongo)
	if err != nil {
		return createMessageResponse(false, deletePublisherFailedMessage)
	}

	if !owned {
		return createMessageResponse(false, "publisher not found")
	}

	deletedPublisher, err := deletePublisher(pubId, mongo)

	if err != nil || !deletedPublisher {
		return createMessageResponse(false, deletePublisherFailedMessage)
	}

	go deleteAllPublisherMessages(pubId, mongo)

	go deleteExistingSubscriptions(pubId, mongo)

	return createMessageResponse(true, "publisher deleted")
}

type subscribersResult struct {
	Success     bool            `json:"success"`
	Subscribers []jsonPublisher `json:"subscribers"`
}

func getPublisherSubscribers(pubId string, mongo *bezmongo.MongoService) []byte {
	filter := bson.D{{Key: "subscriptions", Value: bson.D{{Key: "$elemMatch", Value: bson.D{{Key: "publisher_id", Value: pubId}}}}}}
	projection := bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: 1}}
	collection := mongo.OpenCollection(messageBrokerDb, "clients")
	results, err := bezmongo.FindMany(collection, options.Find().SetProjection(projection), filter)
	if err != nil {
		return createMessageResponse(false, "failed finding subscribers")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bResults := []bsonPublisher{}
	jResults := []jsonPublisher{}
	results.All(ctx, &bResults)
	for _, p := range bResults {
		jResults = append(jResults, jsonPublisher{
			p.Id,
			p.Name,
		})
	}
	result, err := json.Marshal(subscribersResult{
		Success:     true,
		Subscribers: jResults,
	})
	if err != nil {
		return createMessageResponse(false, "failed fetching publications")
	}
	return result
}

func handleGetPublisherSubscribers(pubId string, ownerId string, mongo *bezmongo.MongoService) []byte {
	deletePublisherFailedMessage := "get subscribers failed"
	owned, err := checkOwnsPublisher(pubId, ownerId, mongo)
	if err != nil {
		return createMessageResponse(false, deletePublisherFailedMessage)
	}
	if !owned {
		return createMessageResponse(false, "publisher not found")
	}
	return getPublisherSubscribers(pubId, mongo)
}
