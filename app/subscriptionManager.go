package main

import (
	"fmt"
	"time"
)

type subscriptionManager struct {
	subscriptions             map[string]*subscription
	newSubscriptionChannel    chan *subscription
	confirmChannel            chan []confirmMessageData
	sendToClientChannel       chan sendRequest
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
					subManager.sendToClientChannel <- sendRequest{
						jSONCommunication{
							Action: "messages",
							Data:   messages,
						},
						errorSuccess{},
					}
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

func (subManager *subscriptionManager) stop() {
	timeout := time.After(2 * time.Second)
	select {
	case subManager.cancelReceiveChannel <- true:
	case <-timeout:
	}

	for _, sub := range subManager.subscriptions {
		timeout := time.After(30 * time.Second)
		select {
		case sub.cancelChannel <- true:
		case <-timeout:
		}
		timeout = time.After(30 * time.Second)
		select {
		case sub.cancelConfirmChannel <- true:
		case <-timeout:
		}
	}
}

func (subManager *subscriptionManager) start(mongoManager *mongoManager) {
	go subManager.receiveLoop()
	for _, sub := range subManager.subscriptions {
		go sub.confirmLoop(mongoManager)
		go sub.loop(mongoManager)
	}
}

func (subManager *subscriptionManager) managerLoop(mongoManager *mongoManager) {
	closed := false
	for {
		select {
		case sub := <-subManager.newSubscriptionChannel:
			subManager.stop()
			subManager.subscriptions[sub.id] = sub
			subManager.start(mongoManager)
		case subId := <-subManager.removeSubscriptionChannel:
			subManager.stop()
			delete(subManager.subscriptions, subId)
			subManager.start(mongoManager)
		case messages := <-subManager.confirmChannel:
			subMessages := make(map[string][]string)
			for _, msg := range messages {
				subMessages[msg.SubscriptionId] = append(subMessages[msg.SubscriptionId], msg.Id)
			}
			for key, v := range subMessages {
				subManager.subscriptions[key].receiveConfirmedChannel <- v
			}

		case <-subManager.cancelManagerChannel:
			fmt.Println("sub manager stop")
			subManager.stop()
			closed = true
		}
		if closed {
			break
		}
	}
}
