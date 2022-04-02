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

    return (
        <div className="section">
            <div className="container">
                <h3 className="title is-4">Your Publishers</h3>
                <div className="content">
                    <ul>
                        {renderPublisherList()}
                    </ul>
                </div>
            </div>
        </div>
        
    )
}

export default PublisherList