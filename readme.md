# Message Broker Service

Service to facilitate communicating messages between other services.

## Installation

Includes a docker-compose.yml file which can be used to setup a test instance of the message broker service along with its MongoDB database and a test client application.

Create the project in Docker using the docker compose command, e.g.

```
docker compose -p "test_message_broker" up -d
```

You can then access the test client by accessing "http://localhost:3000" in the browser.

## Connect to the Service

Connect by creating a new websocket connection to the service running on port 8001.

```javascript
let ws = new WebSocket("ws:localhost:8001/ws");

ws.addEventListener('message',(message)=>{
    console.log(message);
});
```

## Identify Connected Client

The first message you receive on the websocket connection will ask you to authenticate in order to identify the connecting client. 

### Register New Client

Since this is the first time connecting to the service you will need to register as a new client.

```javascript
ws.send(JSON.stringify({
    "register": true,
    "name": "My Client"
}));
```

After sending this to the service, a message will be sent back containing the ID of the client.

```json
{
    "action": "authentication_successful",
    "data": {
        "id": "dddfd3e7-1624-41b3-a71d-5c049a4775f2",
        "name": "My Client"
    }
}
```

### Identify as Registered Client

After registering a client, future connections to the service can then use the ID to identify the client.

```javascript
ws.send(JSON.stringify({
    "register": false,
    "id": "dddfd3e7-1624-41b3-a71d-5c049a4775f2"
}));
```

## Publishers

### Create Publisher

The message broker functions with publishers and subscribers.

Once created, a publisher can be used to send out messages which will then be picked up and consumed by subscribers.

A client can own multiple publishers if necessary in order to send out messages to different services or to help organise messages.

Publishers are created by sending the following message to the service:

```javascript
ws.send(JSON.stringify({
    "action": "register_publisher",
    "name": "My First Feed"
}));
```

After registering a publisher, the service will respond with a message confirming that the publisher was created along with the details of the publisher.

```json
{
    "action": "publisher_registered",
    "data": {
        "id": "a9379f06-2d4e-43fd-9161-449c3d18058c",
        "name": "My First Feed"
    }
}
```

### Get Publishers

The following message can be sent to the service in order to return a list of publishers belonging to the identified client:

```javascript
{
    "action": "get_publishers"
}
```

The service will then return a message with a list of publishers.

```json
{
    "action": "your_publishers",
    "data": [
        {
            "id": "a9379f06-2d4e-43fd-9161-449c3d18058c",
            "name": "My First Feed"
        }
    ]
}
```

### Publishing Messages

The "publish_message" action is used to publish a message to the service.

This message must include the following values:

* publisher_id - the ID of the publisher
* ttl - the time to live value in seconds
* payload - string containing the data of the message

```javascript
ws.send(JSON.stringify({
    "action": "publish_message",
    "data": {
        "publisher_id": "a9379f06-2d4e-43fd-9161-449c3d18058c",
        "ttl": 1000,
        "payload": "message encoded as a string"
    }
}));
```

## Subscribers

Subscribing to a publisher allows a client to receive published messages.

A client can subscribe to multiple publishers if necessary to receive messages from different services.

### Subscribe

The following message can be sent to subscribe to a publisher:

```javascript
ws.send(JSON.stringify({
    "action": "subscribe",
    "data": {
        "publisher_id": "a9379f06-2d4e-43fd-9161-449c3d18058c"
    }
}));
```

### Receiving Messages

Once subscribed to a publisher a client will start to receive published messages in the following format.

```json
{
    "action": "messages",
    "data": [
        {
            "id": "6e572c17-514b-44ab-b404-736ebfe6e42e",
            "subscription_id": "100622fa-78b0-43df-b1b8-bb11ad03e47c",
            "payload": "message encoded as a string"
        }
    ]
}
```

### Confirming Received Messages

After receiving messages from the service a response should be sent back to confirm that the messages have been received, this will prevent the same messages from being sent again.

No more messages from the same subscription will be received during the current connection to the service until the confirmation response is sent.

```javascript
ws.send(JSON.stringify({
    "action": "confirm_messages",
    "data": {
        "messages": [
            {
                "id": "6e572c17-514b-44ab-b404-736ebfe6e42e",
                "subscription_id": "100622fa-78b0-43df-b1b8-bb11ad03e47c"
            }
        ]
    }
}));
```