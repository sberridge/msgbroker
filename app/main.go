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

type errorSuccess struct {
	successChannel chan bool
	errorChannel   chan error
}

type jSONCommunication struct {
	Action  string      `json:"action"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type jSONAuthResponse struct {
	Register bool   `json:"register"`
	Name     string `json:"name"`
	UniqueId string `json:"id"`
}

type sendRequest struct {
	message      interface{}
	errorSuccess errorSuccess
}

type clientConnection struct {
	id                   string
	name                 string
	connection           *websocket.Conn
	sendChannel          chan sendRequest
	sendClosedChannel    chan bool
	receiveClosedChannel chan bool
	receiveChannel       chan string
}

func (client *clientConnection) receiveLoop(managerChannels connectionManagerChannels) {
	for {
		_, message, err := client.connection.ReadMessage()
		if err != nil {
			managerChannels.lostConnection <- client
			client.close()
			return
		}
		client.receiveChannel <- string(message)
	}
}

func (client *clientConnection) sendLoop() {
	closed := false
	for {
		select {
		case msg := <-client.sendChannel:
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
		case <-client.sendClosedChannel:
			closed = true
		}
		if closed {
			break
		}
	}
}

func (client *clientConnection) send(message interface{}, customErrorSuccess errorSuccess) {
	thisErrorSuccess := errorSuccess{
		errorChannel:   make(chan error),
		successChannel: make(chan bool),
	}
	client.sendChannel <- sendRequest{
		message:      message,
		errorSuccess: thisErrorSuccess,
	}

	select {
	case err := <-thisErrorSuccess.errorChannel:
		if customErrorSuccess.errorChannel != nil {
			customErrorSuccess.errorChannel <- err
		}
	case <-thisErrorSuccess.successChannel:
		if customErrorSuccess.successChannel != nil {
			customErrorSuccess.successChannel <- true
		}
	}
}

func (client *clientConnection) close() {
	client.connection.Close()
	timeout := time.After(time.Second * 5)
	select {
	case client.receiveClosedChannel <- true:
	case <-timeout:
	}

	timeout = time.After(time.Second * 5)
	select {
	case client.sendClosedChannel <- true:
	case <-timeout:
	}

}

type connectionManagerChannels struct {
	newConnection  chan *clientConnection
	lostConnection chan *clientConnection
}

func connectionManager(channels connectionManagerChannels) {
	connections := make(map[string]*clientConnection)
	for {
		select {
		case newCon := <-channels.newConnection:
			connections[newCon.id] = newCon
		case lostCon := <-channels.lostConnection:
			delete(connections, lostCon.id)
		}
	}
}

func handleClientMessages(client *clientConnection) {
	closed := false
	for {
		select {
		case message := <-client.receiveChannel:
			jsonMsg := jSONCommunication{}
			err := json.Unmarshal([]byte(message), &jsonMsg)
			if err != nil {
				client.send(jSONCommunication{
					Action:  "invalid message",
					Message: "The message sent was incorrectly formatted",
				}, errorSuccess{})
				continue
			}
			fmt.Println(message)
		case <-client.receiveClosedChannel:
			closed = true
		}
		if closed {
			break
		}
	}
}

func handleConnection(con *websocket.Conn, managerChannels connectionManagerChannels) {
	client := clientConnection{
		id:                   uuid.New().String(),
		connection:           con,
		receiveChannel:       make(chan string),
		sendChannel:          make(chan sendRequest),
		receiveClosedChannel: make(chan bool),
		sendClosedChannel:    make(chan bool),
	}

	go client.receiveLoop(managerChannels)
	go client.sendLoop()

	authSuccessChan := make(chan bool)
	authErrorChan := make(chan error)

	go authenticate(&client, authSuccessChan, authErrorChan)

	select {
	case <-authSuccessChan:
	case <-authErrorChan:
		client.close()
		return
	}

	managerChannels.newConnection <- &client
	go handleClientMessages(&client)
}

func main() {
	channels := connectionManagerChannels{
		newConnection:  make(chan *clientConnection),
		lostConnection: make(chan *clientConnection),
	}
	go connectionManager(channels)
	http.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		con, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			go handleConnection(con, channels)
		}
	})

	http.ListenAndServe(":8001", nil)
}
