import React from "react";
import { subscriptionDetails } from './../types/types'


type subscriptionListProps = {
    subscriptions: subscriptionDetails[]
    selectedSubscription: null | subscriptionDetails
    onSelectSubscription: (subscriptionId:subscriptionDetails|null)=>void
}

const SubscriptionList = (props:subscriptionListProps) => {

    const renderSubscriptionList = () => {
        return props.subscriptions.map((subscription)=>{
            return <li key={subscription.id}><a className={props.selectedSubscription?.id == subscription.id ? "is-active" : ""} onClick={()=>{props.onSelectSubscription(subscription)}}>{subscription.publisher.name}</a></li>
        })
    }

    return (
        <div className="menu">
            <p className="menu-label">New Subscription</p>
            <ul className="menu-list">
                <li><a onClick={()=>{props.onSelectSubscription(null)}} className={!props.selectedSubscription ? "is-active" : ""}>New Subscription</a></li>
            </ul>
            <p className="menu-label">Your Subscriptions</p>
            <ul className="menu-list">
                {renderSubscriptionList()}
            </ul>
        </div>        
    )
}

export default SubscriptionList