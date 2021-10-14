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

type clientConnection struct {
	id         string
	name       string
	connection *websocket.Conn
}

type channels struct {
	newConnection    chan clientConnection
	lostConnection   chan clientConnection
	broadcastChannel chan string
}

func sendMessage(con *websocket.Conn, message string, successChan chan bool, errChan chan error) {
	err := con.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil && errChan != nil {
		errChan <- err
	} else if successChan != nil {
		successChan <- true
	}
}

func connectionManager(channels channels) {
	connections := make(map[string]clientConnection)
	for {
		select {
		case newCon := <-channels.newConnection:
			connections[newCon.id] = newCon
		case lostCon := <-channels.lostConnection:
			delete(connections, lostCon.id)
		case message := <-channels.broadcastChannel:
			for _, client := range connections {
				go sendMessage(client.connection, message, nil, nil)
			}
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

func handleConnection(con *websocket.Conn, channels channels) {
	client := clientConnection{
		id:         uuid.New().String(),
		connection: con,
	}

	authSuccessChan := make(chan jSONAuthResponse)
	authErrorChan := make(chan error)

	go authenticate(con, authSuccessChan, authErrorChan)

	timeout := time.After(time.Second * 30)

	select {
	case authResponse := <-authSuccessChan:
		client.id = authResponse.UniqueId
		client.name = authResponse.Name
		successResponse, _ := json.Marshal(jSONCommunication{
			Action:  "authentication successful",
			Message: "",
			Data:    authResponse,
		})
		sendMessage(con, string(successResponse), nil, nil)
	case err := <-authErrorChan:
		fmt.Printf("errored authenticating, %s", err.Error())
		if err.Error() == "client exists" {
			failResponse, _ := json.Marshal(jSONCommunication{
				Action:  "authentication failed",
				Message: "Client already exists",
			})
			sendMessage(con, string(failResponse), nil, nil)
		}
		con.Close()
		return
	case <-timeout:
		fmt.Println("authentication timed out")
		failResponse, _ := json.Marshal(jSONCommunication{
			Action:  "authentication failed",
			Message: "Authentication timed out",
		})
		sendMessage(con, string(failResponse), nil, nil)
		con.Close()
		return
	}

	channels.newConnection <- client
	for {
		_, message, err := con.ReadMessage()
		if err != nil {
			fmt.Println(err.Error())
			channels.lostConnection <- client
			return
		}
		fmt.Println(string(message))
	}
}

func main() {
	channels := channels{
		newConnection:    make(chan clientConnection),
		lostConnection:   make(chan clientConnection),
		broadcastChannel: make(chan string),
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
