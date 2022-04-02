import React from "react";
import { publisherDetails } from './../types/types'


type publisherListProps = {
    publishers: publisherDetails[]
    selectedPublisher: null | string
    onSelectPublisher: (publisherId:string|null)=>void
}

const PublisherList = (props:publisherListProps) => {

    const renderPublisherList = () => {
        return props.publishers.map((publisher)=>{
            return <li key={publisher.id}><a className={props.selectedPublisher == publisher.id ? "is-active" : ""} onClick={()=>{props.onSelectPublisher(publisher.id)}}>{publisher.name}</a></li>
        })
    }

    return (
        <div className="menu">
            <p className="menu-label">New Publisher</p>
            <ul className="menu-list">
                <li><a onClick={()=>{props.onSelectPublisher(null)}} className={!props.selectedPublisher ? "is-active" : ""}>New Publisher</a></li>
            </ul>
            <p className="menu-label">Your Publishers</p>
            <ul className="menu-list">
                {renderPublisherList()}
            </ul>
        </div>        
    )
}

export default PublisherList