import React, { MouseEvent, useEffect, useState } from "react";
import APIRequest from "../modules/APIRequest";
import { publisherDetails, statusMessage } from "../types/types";
import PublishMessageForm from "./publishMessageForm";

type publisherProps = {
    publisher: publisherDetails
    onPublisherDeleted: ()=>void
}

type subscriberDetails = {
    id: string
    name: string
}

const Publisher = (props:publisherProps) => {

    
    const [statusMessage,setStatusMessage] = useState<statusMessage|null>(null);
    const [subscribers,setSubscribers] = useState<subscriberDetails[]>([])
    const [loading,setLoading] = useState(false);
    const [deleteConfirmationVisible, setDeleteConfirmationVisible] = useState(false);


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

    const handleDelete = async (e:MouseEvent) => {
        e.preventDefault();
        setLoading(true);
        const result = await (new APIRequest())
            .setRoute(`/publishers/${props.publisher.id}`)
            .setMethod("DELETE")
            .send()
            .catch((e)=>{
                setStatusMessage({
                    "message": "Failed to delete publisher",
                    "type": "is-danger"
                })
            });
        if(!result) return;
        if(!result.success) {
            setStatusMessage({
                "message": result.message ?? "Failed to delete publisher",
                "type": "is-danger"
            })
        }
        props.onPublisherDeleted();

    }


    return (
        <div className="section">
            <div className="container">
                <h3 className="title is-4">{props.publisher.name}</h3>
                <h4 className="title is-5">{props.publisher.id}</h4>
                <div className={`modal ${deleteConfirmationVisible ? 'is-active' : ''}`}>
                <   div className="modal-background"></div>
                    <div className="modal-card">
                        <header className="modal-card-head">
                            <p className="modal-card-title">Confirm</p>
                            <button className="delete" aria-label="close"></button>
                        </header>
                        <section className="modal-card-body">
                            {statusMessage && 
                                <div className={`notification ${statusMessage.type}`}>
                                    <button className="delete" onClick={()=>{setStatusMessage(null);}}></button>
                                    {statusMessage.message}
                                </div>
                            }
                            <p>Are you sure you want to delete this publisher?</p>
                        </section>
                        <footer className="modal-card-foot">
                            <button onClick={handleDelete} className={`button is-success ${loading ? "is-loading" : ""}`}>Confirm</button>
                            <button onClick={()=>{setDeleteConfirmationVisible(false)}} className={`button is-danger ${loading ? "is-loading" : ""}`}>Cancel</button>
                        </footer>

                    </div>
                    
                </div>
                <div className="columns">
                    <div className="column">
                        <PublishMessageForm
                            publisherId={props.publisher.id}
                        ></PublishMessageForm>
                        <div className="container mt-6">
                            <button onClick={()=>{setDeleteConfirmationVisible(true)}} className={`button is-danger`}>Delete</button>
                        </div>
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