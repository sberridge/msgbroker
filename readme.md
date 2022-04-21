# Message Broker Service

Service to facilitate communicating messages between other services.

## Overview

The service works by having clients registered to the service which can send out messages via "publishers" and receive messages via "subscriptions" to those publishers.

Clients can act as either a publisher, a subscriber, or both.

* Client registers to the service (publisher)
    * Register publisher(s)
        * Publish messages
* Second client registers to the service (subscriber)
    * Subscribe to publisher(s)
        * Start receiving messages from subscribed publishers
            * Client confirms received messages

## Installation

Includes a docker-compose.yml file which can be used to setup a test instance of the message broker service along with its MongoDB database and a test client application.

Create the project in Docker using the docker compose command, e.g.

```
docker compose -p "test_message_broker" up -d
```

You can then access the test client by accessing "http://localhost:8080" in the browser.