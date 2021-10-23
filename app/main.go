package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//struct for reusable success/error channel responses
type errorSuccess struct {
	successChannel chan bool
	errorChannel   chan error
}

//struct to define JSON messages sent to and from the client
type jSONCommunication struct {
	Action  string      `json:"action"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

//struct sent to the send channel with the message to be sent + the error/success response channels
type sendRequest struct {
	message      interface{}
	errorSuccess errorSuccess
}

type newPublisherRequestData struct {
	Name string `json:"name"`
}
type newPublisherRequest struct {
	Action  string                  `json:"action"`
	Message string                  `json:"message"`
	Data    newPublisherRequestData `json:"data"`
}

type publishMessageRequestData struct {
	Publisher_id string `json:"publisher_id"`
	Ttl          int64  `json:"ttl"`
	Payload      string `json:"payload"`
}
type publishMessageRequest struct {
	Action  string                    `json:"action"`
	Message string                    `json:"message"`
	Data    publishMessageRequestData `json:"data"`
}

type subscribeRequestData struct {
	Publisher_id string `json:"publisher_id"`
}
type subscribeRequest struct {
	Action  string               `json:"action"`
	Message string               `json:"message"`
	Data    subscribeRequestData `json:"data"`
}

type confirmMessageData struct {
	Id             string `json:"id"`
	SubscriptionId string `json:"subscription_id"`
}
type confirmRequestData struct {
	Messages []confirmMessageData `json:"messages"`
}
type confirmRequest struct {
	Action  string             `json:"action"`
	Message string             `json:"message"`
	Data    confirmRequestData `json:"data"`
}

func clientMessagesLoop(client *clientConnection, mongoManager *mongoManager) {
	closed := false
	for {
		select {
		case message := <-client.receiveChannel:
			jsonMsg := jSONCommunication{}
			err := json.Unmarshal([]byte(message), &jsonMsg)
			if err != nil {
				client.send(jSONCommunication{
					Action:  "invalid_message",
					Message: "The message sent was incorrectly formatted",
				}, errorSuccess{})
				break
			}
			switch jsonMsg.Action {
			case "register_publisher":
				newPublisherRequest := newPublisherRequest{}
				err := json.Unmarshal([]byte(message), &newPublisherRequest)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_registering_publisher",
						Message: "Invalid json format",
					}, errorSuccess{})
					break
				}
				publisher, err := newPublisher(client, newPublisherRequest.Data, mongoManager)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_registering_publisher",
						Message: err.Error(),
					}, errorSuccess{})
				} else {
					client.send(jSONCommunication{
						Action: "publisher_registered",
						Data:   publisher,
					}, errorSuccess{})
				}
			case "get_publishers":
				publishers, err := getPublishers(client, mongoManager)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_fetching_publishers",
						Message: err.Error(),
					}, errorSuccess{})
					break
				}
				client.send(jSONCommunication{
					Action: "your_publishers",
					Data:   publishers,
				}, errorSuccess{})
			case "publish_message":
				publishMessageRequest := publishMessageRequest{}
				err := json.Unmarshal([]byte(message), &publishMessageRequest)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_publishing_message",
						Message: "Invalid json format",
					}, errorSuccess{})
					break
				}
				publishedMessage, err := publishMessage(client, publishMessageRequest.Data, mongoManager)
				if publishedMessage {

					client.send(jSONCommunication{
						Action:  "message_published",
						Message: "Message published",
					}, errorSuccess{})
					fmt.Println("done")
				} else {
					client.send(jSONCommunication{
						Action:  "failed_publishing_message",
						Message: err.Error(),
					}, errorSuccess{})
				}
			case "subscribe":
				subscribeRequest := subscribeRequest{}
				err := json.Unmarshal([]byte(message), &subscribeRequest)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_subscribing",
						Message: "Invalid json format",
					}, errorSuccess{})
					break
				}
				subscription, err := subscribe(client, mongoManager, subscribeRequest.Data.Publisher_id)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_subscribing",
						Message: err.Error(),
					}, errorSuccess{})
				} else {
					client.send(jSONCommunication{
						Action:  "subscribed",
						Message: "Subscribed",
					}, errorSuccess{})
					client.subscriptionManager.newSubscriptionChannel <- subscription
				}
			case "confirm_messages":
				confirmRequest := confirmRequest{}
				err := json.Unmarshal([]byte(message), &confirmRequest)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_confirmation",
						Message: "Invalid json format",
					}, errorSuccess{})
					break
				}
				confirmMessagesStruct := subscriptionManagerConfirmation{
					messages:               confirmRequest.Data.Messages,
					numberConfirmedChannel: make(chan int),
				}
				client.subscriptionManager.confirmChannel <- &confirmMessagesStruct
				confirmed := <-confirmMessagesStruct.numberConfirmedChannel
				client.send(jSONCommunication{
					Action: "messages_confirmed",
					Data: map[string]int{
						"confirmed": confirmed,
					},
				}, errorSuccess{})
			}
		case <-client.receiveClosedChannel:
			closed = true
			fmt.Println("receive loop stop")
		}
		if closed {
			break
		}
	}
}

func handleExpiredMessages(mongoManager *mongoManager) {

	for {
		filter := bson.D{
			primitive.E{Key: "ttl", Value: bson.D{
				primitive.E{Key: "$lt", Value: time.Now().Unix()},
				primitive.E{Key: "$ne", Value: 0},
			},
			},
		}

		col := mongoManager.connection.Database("message-broker").Collection("publisher_messages")
		_, err := mongoDeleteMany(col, filter)
		if err != nil {
			fmt.Println(err.Error())
		}
		<-time.After(time.Second * 30)
	}
}

//channels for the connection manager
type connectionManagerChannels struct {
	newConnection  chan *clientConnection //receive new client connections
	lostConnection chan *clientConnection //channel to remove closed client connections
}

//manage client connections
func connectionManager(channels connectionManagerChannels) {
	//map to store open connections
	connections := make(map[string]*clientConnection)
	for {
		select {
		case newCon := <-channels.newConnection: //received a new client connection, add it to the map
			connections[newCon.id] = newCon
		case lostCon := <-channels.lostConnection: //lost a client connection, remove it from the map
			delete(connections, lostCon.id)
		}
	}
}

//handle setting up and authenticating a new client connection
func handleConnection(con *websocket.Conn, managerChannels connectionManagerChannels, mongoManager *mongoManager) {
	client := clientConnection{
		id:                   uuid.New().String(),
		connection:           con,
		receiveChannel:       make(chan string),
		sendChannel:          make(chan sendRequest),
		receiveClosedChannel: make(chan bool),
		sendClosedChannel:    make(chan bool),
	}

	//start the receive messages loop
	go client.receiveLoop(managerChannels)

	//start the send message loop
	go client.sendLoop()

	//authenticate the client connection
	bsonClient, err := authenticate(&client, mongoManager)

	if err != nil {
		client.close()
		return
	}

	//add authed client to the manager
	managerChannels.newConnection <- &client
	subManager := subscriptionManager{
		subscriptions:             map[string]*subscription{},
		newSubscriptionChannel:    make(chan *subscription),
		confirmChannel:            make(chan *subscriptionManagerConfirmation),
		removeSubscriptionChannel: make(chan string),
		cancelReceiveChannel:      make(chan bool),
		cancelManagerChannel:      make(chan bool),
		sendToClientChannel:       client.sendChannel,
	}
	client.subscriptionManager = &subManager

	go client.subscriptionManager.managerLoop(mongoManager)

	for _, sub := range bsonClient.Subscriptions {
		client.subscriptionManager.newSubscriptionChannel <- &subscription{
			id:                      sub.Id,
			publisherId:             sub.PublisherId,
			clientId:                client.id,
			cancelChannel:           make(chan bool),
			messagesChannel:         make(chan []messageItem),
			receiveConfirmedChannel: make(chan *subscriptionMessagesConfirmation),
		}
	}

	//start receiving messages from the client
	go clientMessagesLoop(&client, mongoManager)
}

func main() {
	//channels for the client manager
	channels := connectionManagerChannels{
		newConnection:  make(chan *clientConnection),
		lostConnection: make(chan *clientConnection),
	}

	mongoManager, err := startMongo()
	if err != nil {
		fmt.Printf("Failed opening mongo connection, %s", err.Error())
	}
	//start the client manager
	go connectionManager(channels)

	go handleExpiredMessages(mongoManager)

	//route to open a websocket connection
	http.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		//hijack the request and turn it into a websocket connection
		upgrader.CheckOrigin = func(r *http.Request) bool {

			return true
		}
		con, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			//start handling the connection
			go handleConnection(con, channels, mongoManager)
		}
	})

	http.ListenAndServe(":8001", nil)
}
