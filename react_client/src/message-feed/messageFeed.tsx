import React, { useEffect, useReducer, useState } from "react";
import { authedUser } from "../types/types";

type messageFeedProps = {
    authedUser: authedUser
}

type message = {
    id: string
    publisherId: string
    payload: string
    confirmed: boolean
}

interface MessagesStateInterface {
    messages: message[]
}

interface MessagesActionInterface {
    type: string
    value: message[]
}

const initialMessagesState:MessagesStateInterface = {
    messages: []
};

const messagesReducer = (state:MessagesStateInterface, action: MessagesActionInterface) => {
    
    if(action.type === "add") {
        return {
            messages: [...state.messages,...action.value]
        }
    }
    if(action.type === "confirm") {
        return {
            messages: state.messages.map((message)=>{
                message.confirmed = true;
                return message;
            })
        }
    }
    return {
        messages: [...state.messages]
    }
}

const MessageFeed = (props:messageFeedProps) => {

    const [connectionStatus,setConnectionStatus] = useState("is-warning");
    const [connectionMessage,setConnectionMessage] = useState("connecting");

    const [state, dispatch] = useReducer(messagesReducer, initialMessagesState);


    let ws:WebSocket;

    const send = (data:any) => {
        ws.send(JSON.stringify(data));
    }

    const handleWSMessage = (message:MessageEvent) => {
        let content = JSON.parse(message.data);
        let newMessageList:message[];
        switch(content.action) {
            case "authenticate":
                send({
                    id: props.authedUser.id
                });
                break;
            case "authentication_successful":
                setConnectionStatus("is-success");
                setConnectionMessage("connected");
                break;
            case "authentication_failed":
                setConnectionStatus("is-danger");
                setConnectionMessage("failed");
                break;
            case "messages":

                newMessageList = [];
                let confirmMessageIds:{id:string,subscription_id:string}[] = [];
                content.data.forEach((message:any)=>{
                    newMessageList.push({
                        id: message.id,
                        payload: message.payload,
                        publisherId: message.publisher_id,
                        confirmed: false
                    });
                    confirmMessageIds.push({
                        id: message.id,
                        subscription_id: message.subscription_id
                    });
                })
                dispatch({type: "add", value: newMessageList});
                send({
                    action: "confirm_messages",
                    data: {
                        messages: confirmMessageIds
                    }
                })
                console.log('sent confirm'); 
                break;
            case "messages_confirmed":
                dispatch({
                    type: "confirm",
                    value:[]
                });
                break;
        }

    }

    useEffect(()=>{
        ws = new WebSocket("ws://localhost:8001/ws");
        ws.onmessage = handleWSMessage
        return ()=>{
            console.log('test');
            ws.close();
        }
    },[])

    const renderMessages = () => {
        return [...state.messages].reverse().map((message)=>{
            return <article key={message.id} className={`message ${message.confirmed ? "is-success" : "is-warning"}`}>
                <div className="message-header">
                    <p>{message.publisherId}</p>
                </div>
                <div className="message-body">
                    {message.payload}
                </div>
            </article>
        })
    }

    return (
        <div className="section">
            <div className="tags has-addons">
                <span className="tag is-dark">connection</span>
                <span className={`tag ${connectionStatus}`}>{connectionMessage}</span>
            </div>
            <div>
                {renderMessages()}
            </div>
        </div>
    )
}

export default MessageFeed;