package main

import (
	"fmt"
	"time"
)

type subscriptionManager struct {
	subscriptions             map[string]*subscription
	newSubscriptionChannel    chan *subscription
	confirmChannel            chan *subscriptionManagerConfirmation
	sendToClientChannel       chan sendRequest
	removeSubscriptionChannel chan string
	cancelReceiveChannel      chan bool
	cancelManagerChannel      chan bool
}

type subscriptionManagerConfirmation struct {
	messages               []confirmMessageData
	numberConfirmedChannel chan int
}

func (subManager *subscriptionManager) receiveLoop() {
	closed := false
	for {
		for _, sub := range subManager.subscriptions {
			select {
			case messages := <-sub.messagesChannel:
				if len(messages) > 0 {
					subManager.sendToClientChannel <- sendRequest{
						jsonCommunication{
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
	}
}

func (subManager *subscriptionManager) start(mongoManager *mongoManager) {
	go subManager.receiveLoop()
	for _, sub := range subManager.subscriptions {
		go sub.loop(mongoManager)
	}
}

func waitForSubToConfirm(messages []string, sub *subscription, confirmedChannel chan int) {
	sub.receiveConfirmedChannel <- &subscriptionMessagesConfirmation{
		messages:         messages,
		confirmedChannel: confirmedChannel,
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
		case confirmation := <-subManager.confirmChannel:
			subMessages := make(map[string][]string)
			for _, msg := range confirmation.messages {
				subMessages[msg.SubscriptionID] = append(subMessages[msg.SubscriptionID], msg.Id)
			}
			subConfirmedChannels := []chan int{}
			for key, v := range subMessages {
				confirmedChannel := make(chan int)
				subConfirmedChannels = append(subConfirmedChannels, confirmedChannel)
				go waitForSubToConfirm(v, subManager.subscriptions[key], confirmedChannel)
			}
			totalConfirmed := 0
			for _, confirmChannel := range subConfirmedChannels {
				totalConfirmed += <-confirmChannel
			}
			confirmation.numberConfirmedChannel <- totalConfirmed
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
