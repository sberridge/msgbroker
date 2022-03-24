package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
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
type jsonCommunication struct {
	Action  string      `json:"action"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

//struct sent to the send channel with the message to be sent + the error/success response channels
type sendRequest struct {
	message      interface{}
	errorSuccess errorSuccess
}

type confirmMessageData struct {
	Id             string `json:"id"`              //id of the message being confirmed
	SubscriptionID string `json:"subscription_id"` //id of the subscription the message was received on
}
type confirmRequestData struct {
	Messages []confirmMessageData `json:"messages"` //slice of messages being confirmed from the client including the message ID and the subscription id
}
type confirmRequest struct {
	Action  string             `json:"action"`
	Message string             `json:"message"`
	Data    confirmRequestData `json:"data"`
}

func handleConfirmMessage(message string, client *clientConnection) {
	confirmRequest := confirmRequest{}
	err := json.Unmarshal([]byte(message), &confirmRequest)
	if err != nil {
		client.send(jsonCommunication{
			Action:  "failed_confirmation",
			Message: "Invalid json format",
		}, errorSuccess{})
		return
	}
	confirmMessagesStruct := subscriptionManagerConfirmation{
		messages:               confirmRequest.Data.Messages,
		numberConfirmedChannel: make(chan int),
	}
	client.subscriptionManager.confirmChannel <- &confirmMessagesStruct
	confirmed := <-confirmMessagesStruct.numberConfirmedChannel
	client.send(jsonCommunication{
		Action: "messages_confirmed",
		Data: map[string]int{
			"confirmed": confirmed,
		},
	}, errorSuccess{})
}
func handleClientMessage(message string, client *clientConnection, mongoManager *mongoManager) {
	jsonMsg := jsonCommunication{}
	err := json.Unmarshal([]byte(message), &jsonMsg)
	if err != nil {
		client.send(jsonCommunication{
			Action:  "invalid_message",
			Message: "The message sent was incorrectly formatted",
		}, errorSuccess{})
		return
	}
	switch jsonMsg.Action {
	case "confirm_messages": //request to confirm that the client received a set of messages from a subscription
		handleConfirmMessage(message, client)
	}
}

//loop running in a goroutine to handle messages coming from the client via the websocket
func clientMessagesLoop(client *clientConnection, mongoManager *mongoManager) {
	closed := false
	for {
		select {
		case message := <-client.receiveChannel: //received a message from the client
			handleClientMessage(message, client, mongoManager)
		case <-client.receiveClosedChannel:
			closed = true
			fmt.Println("receive loop stop")
		}
		if closed {
			break
		}
	}
}

//loop running in a goroutine to handle deleting expired messages
func handleExpiredMessages(mongoManager *mongoManager) {

	for {
		filter := bson.D{
			{Key: "ttl", Value: bson.D{
				{Key: "$lt", Value: time.Now().Unix()},
				{Key: "$ne", Value: 0},
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
			publisherID:             sub.PublisherId,
			clientID:                client.id,
			cancelChannel:           make(chan bool),
			messagesChannel:         make(chan []jsonMessageItem),
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
