package main

import (
	"encoding/json"
	"fmt"
	"net/http"

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

type clientConnection struct {
	id             string
	name           string
	connection     *websocket.Conn
	receiveChannel chan string
}

func (client *clientConnection) receiveLoop(managerChannels connectionManagerChannels) {
	for {
		_, message, err := client.connection.ReadMessage()
		if err != nil {
			managerChannels.lostConnection <- client
			return
		}
		client.receiveChannel <- string(message)
	}
}

func (client *clientConnection) send(message interface{}, errorSuccess errorSuccess) {
	msg, err := json.Marshal(message)
	if err != nil {
		if errorSuccess.errorChannel != nil {
			errorSuccess.errorChannel <- err
		}
		return
	}
	err = client.connection.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		if errorSuccess.errorChannel != nil {
			errorSuccess.errorChannel <- err
		}
		return
	}
	if errorSuccess.successChannel != nil {
		errorSuccess.successChannel <- true
	}
}

func (client *clientConnection) close() {
	client.connection.Close()
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

type jSONCommunication struct {
	Action  string      `json:"action"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jSONAuthResponse struct {
	Register bool   `json:"register"`
	Name     string `json:"name"`
	UniqueId string `json:"id"`
}

func handleConnection(con *websocket.Conn, managerChannels connectionManagerChannels) {
	client := clientConnection{
		id:             uuid.New().String(),
		connection:     con,
		receiveChannel: make(chan string),
	}

	go client.receiveLoop(managerChannels)

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
