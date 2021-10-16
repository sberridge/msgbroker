package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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

	if customErrorSuccess.errorChannel != nil || customErrorSuccess.successChannel != nil {
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

}

type newProviderData struct {
	Name string `json:"name"`
}
type newProviderRequest struct {
	Action  string          `json:"action"`
	Message string          `json:"message"`
	Data    newProviderData `json:"data"`
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
			case "register_provider":
				newProviderRequest := newProviderRequest{}
				err := json.Unmarshal([]byte(message), &newProviderRequest)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_registering_provider",
						Message: "Invalid json format",
					}, errorSuccess{})
				}
				provider, err := newProvider(client, newProviderRequest.Data, mongoManager)
				if err != nil {
					client.send(jSONCommunication{
						Action:  "failed_registering_provider",
						Message: err.Error(),
					}, errorSuccess{})
				} else {
					client.send(jSONCommunication{
						Action: "provider_registered",
						Data:   provider,
					}, errorSuccess{})
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
