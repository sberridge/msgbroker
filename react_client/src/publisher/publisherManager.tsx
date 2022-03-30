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
        console.log(result);
    }

    useEffect(()=>{
        loadPublishers();
    },[]);

    const onPublisherCreated = () => {
        loadPublishers();
    }

    return <div>
        <h2>Publisher Management</h2>
        <h3>New Publisher</h3>
        <NewPublisherForm
            onPublisherCreated={onPublisherCreated}
        ></NewPublisherForm>
        <h3>Your Publishers</h3>
        <PublisherList
            publishers={publishers}
        ></PublisherList>
        
        
    </div>
}

export default PublisherManager;