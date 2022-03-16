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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type bsonPublisher struct {
	Id   string `bson:"_id"`
	Name string `bson:"name"`
}
type jsonPublisher struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
type publishersResult struct {
	Success    bool            `json:"success"`
	Publishers []jsonPublisher `json:"publishers"`
}

func handleGetPublications(mongo *bezmongo.MongoService, id string, query url.Values, responseChannel chan []byte) {
	collection := mongo.OpenCollection("message-broker", "publishers")
	filter := bson.D{{Key: "owner_id", Value: id}}
	findProjection := bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: 1}}
	results, err := bezmongo.FindMany(collection, options.Find().SetProjection(findProjection), filter)
	if err != nil {
		responseChannel <- createMessageResponse(false, "failed fetching publications")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bResults := []bsonPublisher{}
	jResults := []jsonPublisher{}
	results.All(ctx, &bResults)
	for _, p := range bResults {
		jResults = append(jResults, jsonPublisher(p))
	}
	result, err := json.Marshal(publishersResult{
		Success:    true,
		Publishers: jResults,
	})
	if err != nil {
		responseChannel <- createMessageResponse(false, "failed fetching publications")
		return
	}
	responseChannel <- result
}

type createPublisherRequest struct {
	Name string `json:"name"`
}

func checkPublicationExists(name string, id string, mongo *bezmongo.MongoService) bool {
	collection := mongo.OpenCollection("message-broker", "publishers")
	filter := bson.D{{Key: "owner_id", Value: id}, {Key: "name", Value: name}}
	count, err := bezmongo.Count(collection, filter)
	if err != nil {
		return true
	}
	return count > 0
}

func registerPublisher(name string, id string, mongo *bezmongo.MongoService) (string, error) {
	col := mongo.OpenCollection("message-broker", "publishers")

	newId := uuid.New().String()
	_, err := bezmongo.InsertOne(col, bson.D{{Key: "_id", Value: newId}, {Key: "name", Value: name}, {Key: "owner_id", Value: id}})

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

func handleCreatePublication(body io.ReadCloser, id string, mongo *bezmongo.MongoService, responseChannel chan []byte) {
	publisherFailedMessage := "create publication failed"
	bytes, err := readBody(body)
	if err != nil {
		responseChannel <- createMessageResponse(false, publisherFailedMessage)
		return
	}

	publisherRequest := createPublisherRequest{}
	err = json.Unmarshal(bytes, &publisherRequest)
	if err != nil {
		responseChannel <- createMessageResponse(false, publisherFailedMessage)
		return
	}

	if checkPublicationExists(publisherRequest.Name, id, mongo) {
		responseChannel <- createMessageResponse(false, "publication already exists")
		return
	}

	publisherId, err := registerPublisher(publisherRequest.Name, id, mongo)

	if err != nil {
		responseChannel <- createMessageResponse(false, publisherFailedMessage)
		return
	}

	response, err := json.Marshal(createPublisherSuccessResponse{
		Success: true,
		Row: createPublisherResponse{
			Id:   publisherId,
			Name: publisherRequest.Name,
		},
	})

	if err != nil {
		responseChannel <- createMessageResponse(false, publisherFailedMessage)
		return
	}

	responseChannel <- response

}
