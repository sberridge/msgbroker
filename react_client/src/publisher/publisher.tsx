import React, { useEffect, useState } from "react";
import { Validator } from "../lib/Validator";
import APIRequest from "../modules/APIRequest";
import { publisherDetails } from "../types/types";
import PublishMessageForm from "./publishMessageForm";

type publisherProps = {
    publisher: publisherDetails
}

type subscriberDetails = {
    id: string
    name: string
}

const Publisher = (props:publisherProps) => {

    

    const [subscribers,setSubscribers] = useState<subscriberDetails[]>([])


    const getSubscribers = async () => {
        const result = await (new APIRequest())
            .setRoute(`publishers/${props.publisher.id}/subscribers`)
            .setMethod("GET")
            .send();

        return result;
    }
    useEffect(()=>{
        getSubscribers().then((result)=>{
            if(result.success) {
                setSubscribers(result.subscribers);
            }
        });
    },[props.publisher.id]);

    const renderSubscribers = () => {
        return subscribers.map((subscriber)=>{
            return <li key={subscriber.id}>{subscriber.name}</li>
        })
    }

    return (
        <div className="section">
            <div className="container">
                <h3 className="title is-4">{props.publisher.name}</h3>
                <div className="columns">
                    <div className="column">
                        <PublishMessageForm
                            publisherId={props.publisher.id}
                        ></PublishMessageForm>
                    </div>
                    <div className="column is-narrow">
                        <h4 className="title is-5">Subscribers</h4>
                        <ul>
                            {renderSubscribers()}
                        </ul>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Publisher;