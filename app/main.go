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

//struct for managing a client connection
type clientConnection struct {
	id                   string           //unique ID of the client
	name                 string           //unique name of the client
	connection           *websocket.Conn  //websocket connection
	sendChannel          chan sendRequest //channel used to send message requests to the send loop
	sendClosedChannel    chan bool        //channel used to control exiting the send loop when the websocket connection closes
	receiveClosedChannel chan bool        //channel used to control exiting the receive loop when the websocket connection closes
	receiveChannel       chan string      //channel messages received in the receive loop are sent out on to be processed
	subscriptionManager  *subscriptionManager
}

type subscriptionManager struct {
	subscriptions             map[string]*subscription
	newSubscriptionChannel    chan *subscription
	removeSubscriptionChannel chan string
	cancelReceiveChannel      chan bool
	cancelManagerChannel      chan bool
}

func (subManager *subscriptionManager) receiveLoop() {
	closed := false
	for {
		for _, sub := range subManager.subscriptions {
			select {
			case messages := <-sub.messagesChannel:
				if len(messages) > 0 {

				}
			case <-subManager.cancelReceiveChannel:
				closed = true
			}
			if closed {
				break
			}
		}
		if closed {
			break
		}
	}
}

func (subManager *subscriptionManager) managerLoop(mongoManager *mongoManager) {
	closed := false
	for {
		select {
		case sub := <-subManager.newSubscriptionChannel:
			go sub.loop(mongoManager)
			subManager.subscriptions[sub.id] = sub
		case subId := <-subManager.removeSubscriptionChannel:
			timeout := time.After(30 * time.Second)
			select {
			case subManager.subscriptions[subId].cancelChannel <- true:
			case <-timeout:
			}
			delete(subManager.subscriptions, subId)
		case <-subManager.cancelManagerChannel:
			subManager.cancelReceiveChannel <- true
			for _, sub := range subManager.subscriptions {
				timeout := time.After(30 * time.Second)
				select {
				case sub.cancelChannel <- true:
				case <-timeout:
				}
			}
			closed = true
		}
		if closed {
			break
		}
	}
}

//loop for messages received via the websocket connection
func (client *clientConnection) receiveLoop(managerChannels connectionManagerChannels) {
	for {
		_, message, err := client.connection.ReadMessage()
		if err != nil {
			//errored so we've lost connection
			managerChannels.lostConnection <- client
			client.close()
			return
		}

		//got a message, send it out on the channel
		client.receiveChannel <- string(message)
	}
}

//loop to handle sending messages out via the websocket connection
func (client *clientConnection) sendLoop() {
	closed := false
	for {
		select {
		case msg := <-client.sendChannel: //received request to send out a message
			jsonMsg, err := json.Marshal(msg.message)
			if err != nil {
				msg.errorSuccess.errorChannel <- err
				return
			}
			err = client.connection.WriteMessage(websocket.TextMessage, jsonMsg)
			if err != nil {
				if msg.errorSuccess.errorChannel != nil {
					msg.errorSuccess.errorChannel <- err
				}
				return
			}
			if msg.errorSuccess.successChannel != nil {
				msg.errorSuccess.successChannel <- true
			}
		case <-client.sendClosedChannel: //received instruction to exit the loop as the websocket connection has closed
			closed = true
		}
		if closed {
			break
		}
	}
}

//request to send a message to the client
func (client *clientConnection) send(message interface{}, customErrorSuccess errorSuccess) {
	//create channels to receive response from the send loop
	thisErrorSuccess := errorSuccess{
		errorChannel:   make(chan error),
		successChannel: make(chan bool),
	}

	go func(errSuccess errorSuccess) {
		//send the message request to the send loop
		client.sendChannel <- sendRequest{
			message:      message,
			errorSuccess: errSuccess,
		}
	}(thisErrorSuccess)

	select {
	case err := <-thisErrorSuccess.errorChannel: //received an error
		if customErrorSuccess.errorChannel != nil {
			customErrorSuccess.errorChannel <- err //if the requester supplied us with a channel then send the error out
		}
	case <-thisErrorSuccess.successChannel: //successfully sent the message
		if customErrorSuccess.successChannel != nil {
			customErrorSuccess.successChannel <- true //if the requester supplied us with a channel then send the success response
		}
	}

}

//close the websocket connection
func (client *clientConnection) close() {
	client.connection.Close()
	timeout := time.After(time.Second * 5)
	select {
	case client.receiveClosedChannel <- true: //tell the receive loop to stop
	case <-timeout:
	}

	timeout = time.After(time.Second * 5)
	select {
	case client.sendClosedChannel <- true: //tell the send loop to stop
	case <-timeout:
	}

	timeout = time.After(time.Second * 5)
	select {
	case client.subscriptionManager.cancelManagerChannel <- true: //tell the sub manager to stop
	case <-timeout:
	}

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
				continue
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
			case "publish_message":
				publishMessageRequest := publishMessageRequest{}
				err := json.Unmarshal([]byte(message), &publishMessageRequest)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_publishing_message",
						Message: "Invalid json format",
					}, errorSuccess{})
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
			}
		case <-client.receiveClosedChannel:
			closed = true
		}
		if closed {
			break
		}
	}
}

func handleExpiredMessages(mongoManager *mongoManager) {

	for {
		filter := bson.D{
			primitive.E{Key: "messages", Value: bson.D{
				primitive.E{Key: "$all", Value: bson.A{
					bson.D{primitive.E{Key: "$elemMatch", Value: bson.D{
						primitive.E{Key: "ttl", Value: bson.D{
							primitive.E{Key: "$lt", Value: time.Now().Unix()},
							primitive.E{Key: "$ne", Value: 0},
						},
						},
					},
					},
					},
				},
				},
			},
			},
		}

		update := bson.D{
			primitive.E{Key: "$pull", Value: bson.D{
				primitive.E{Key: "messages", Value: bson.D{
					primitive.E{Key: "ttl", Value: bson.D{
						primitive.E{Key: "$lt", Value: time.Now().Unix()},
						primitive.E{Key: "$ne", Value: 0},
					},
					},
				},
				},
			},
			},
		}
		col := mongoManager.connection.Database("message-broker").Collection("publishers")
		_, err := mongoUpdateMany(col, filter, update)
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

	//authentication channels
	authChannels := errorSuccess{
		errorChannel:   make(chan error),
		successChannel: make(chan bool),
	}

	//authenticate the client connection
	go authenticate(&client, authChannels, mongoManager)

	select {
	case <-authChannels.successChannel: //authed!
	case <-authChannels.errorChannel: //not authed, close the connection
		client.close()
		return
	}

	//add authed client to the manager
	managerChannels.newConnection <- &client
	subManager := subscriptionManager{
		subscriptions:             map[string]*subscription{},
		newSubscriptionChannel:    make(chan *subscription),
		removeSubscriptionChannel: make(chan string),
		cancelReceiveChannel:      make(chan bool),
		cancelManagerChannel:      make(chan bool),
	}
	client.subscriptionManager = &subManager

	go client.subscriptionManager.managerLoop(mongoManager)
	go client.subscriptionManager.receiveLoop()

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
