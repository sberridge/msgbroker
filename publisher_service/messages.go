package main

import (
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/sberridge/bezmongo"
	"go.mongodb.org/mongo-driver/bson"
)

const messagesCollection = "publisher_messages"

type publishMessageRequest struct {
	Ttl     int64  `json:"ttl"`     //time to live in seconds
	Payload string `json:"payload"` //payload of the message
}

func handlePublishMessage(body io.ReadCloser, mongo *bezmongo.MongoService, authId string, pubId string) []byte {

	failedMessage := "failed to publish message"

	bytes, err := readBody(body)
	if err != nil {
		return createMessageResponse(false, failedMessage)
	}

	requestData := publishMessageRequest{}

	err = json.Unmarshal(bytes, &requestData)

	if err != nil {
		return createMessageResponse(false, failedMessage)
	}

	owned, err := checkOwnsPublisher(pubId, authId, mongo)

	if err != nil {
		return createMessageResponse(false, failedMessage)
	}

	if !owned {
		return createMessageResponse(false, "publisher not found")
	}

	timeToExpire := int64(0)
	if requestData.Ttl > 0 {
		timeToExpire = time.Now().Unix() + requestData.Ttl
	}

	messagesCollection := mongo.OpenCollection(messageBrokerDb, messagesCollection)
	_, err = bezmongo.InsertOne(messagesCollection, bson.D{
		{Key: "_id", Value: uuid.New().String()},
		{Key: "publisher_id", Value: pubId},
		{Key: "payload", Value: requestData.Payload},
		{Key: "date_created", Value: time.Now()},
		{Key: "ttl", Value: timeToExpire},
	})

	if err != nil {
		return createMessageResponse(false, failedMessage)
	}

	return createMessageResponse(true, "message published")
}
