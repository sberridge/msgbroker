import React from "react";
import { publisherDetails } from './../types/types'


type publisherListProps = {
    publishers: publisherDetails[]
}

const PublisherList = (props:publisherListProps) => {

    const renderPublisherList = () => {
        return props.publishers.map((publisher)=>{
            return <li key={publisher.id}>{publisher.name}</li>
        })
    }

    return <ul>
        {renderPublisherList()}
    </ul>
}

export default PublisherList