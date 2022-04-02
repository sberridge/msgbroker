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

    return (
        <section className="section">
            <div className="container">
                <h2 className="title is-3">Publisher Management</h2>
                
                <NewPublisherForm
                    onPublisherCreated={onPublisherCreated}
                ></NewPublisherForm>
                
                <PublisherList
                    publishers={publishers}
                ></PublisherList>
        
            </div>
        </section>
    )
}

export default PublisherManager;