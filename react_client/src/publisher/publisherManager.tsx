import React, { useEffect, useState } from "react";
import APIRequest from "../modules/APIRequest";
import { publisherDetails } from './../types/types'
import NewPublisherForm from "./newPublisherForm";
import Publisher from "./publisher";
import PublisherList from './publisherList'

type publisherManagerProps = {
    authId: string
};


const PublisherManager = (props:publisherManagerProps) => {

    const [publishers, setPublishers] = useState<publisherDetails[]>([])
    const [selectedPublisher, setSelectedPublisher] = useState<null|publisherDetails>(null);

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

    const onSelectPublisher = (publisher:publisherDetails|null)=>{
        setSelectedPublisher(publisher);
    }

    const onPublisherDeleted = ()=>{
        setSelectedPublisher(null);
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
                    {!selectedPublisher && 
                        <NewPublisherForm
                            onPublisherCreated={onPublisherCreated}
                        ></NewPublisherForm>
                    }
                    {selectedPublisher &&
                        <Publisher
                            publisher={selectedPublisher}
                            onPublisherDeleted={onPublisherDeleted}
                        ></Publisher>
                    }                 
                </div>
                
                
                
                
        
            </div>
        </section>
    )
}

export default PublisherManager;