import React, { useEffect, useState } from "react";
import APIRequest from "../modules/APIRequest";
import { publisherDetails } from './../types/types'
import NewPublisherForm from "./newPublisherForm";
import PublisherList from './publisherList'

type publisherManagerProps = {
    authId: string
};


const PublisherManager = (props:publisherManagerProps) => {

    const [publishers, setPublishers] = useState<publisherDetails[]>([])
    const [selectedPublisher, setSelectedPublisher] = useState<null|string>(null);

    const loadPublishers = async () =>{
        let request = new APIRequest();
        request.setRoute("/publishers")
        let result = await request.send();
        if(result.success) {
            setPublishers(result.publishers);
        }
    }

    useEffect(()=>{
        loadPublishers();
    },[]);

    const onPublisherCreated = () => {
        loadPublishers();
    }

    const onSelectPublisher = (id:string|null)=>{
        setSelectedPublisher(id);
    }

    return (
        <section className="section">
            <h2 className="title is-3">Publisher Management</h2>
            <div className="columns">
                <div className="column is-narrow">
                    <PublisherList
                        publishers={publishers}
                        selectedPublisher={selectedPublisher}
                        onSelectPublisher={onSelectPublisher}
                    ></PublisherList>
                </div>
                <div className="column">
                    <NewPublisherForm
                        onPublisherCreated={onPublisherCreated}
                    ></NewPublisherForm>
                </div>
                
                
                
                
        
            </div>
        </section>
    )
}

export default PublisherManager;