package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

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

//loop for messages received via the websocket connection
func (client *clientConnection) receiveLoop(managerChannels connectionManagerChannels) {
	for {
		_, message, err := client.connection.ReadMessage()
		if err != nil {
			//errored so we've lost connection
			managerChannels.lostConnection <- client
			fmt.Println("lost connection")
			client.close()
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
			fmt.Println("send loop stop")
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

	if client.subscriptionManager != nil {
		timeout = time.After(time.Second * 5)
		select {
		case client.subscriptionManager.cancelManagerChannel <- true: //tell the sub manager to stop
		case <-timeout:
		}
	}

}
