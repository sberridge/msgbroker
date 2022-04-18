import React, { MouseEvent, useState } from "react";
import APIRequest from "../modules/APIRequest";
import { subscriptionDetails } from "../types/types";

type subscriptionProps = {
    selectedSubscription: subscriptionDetails
    onSubscriptionCreated: ()=>void
}

const Subscription = (props:subscriptionProps) => {

    const [isLoading, setIsLoading] = useState(false);

    const buttonClasses = () => {
        let classes = [
            "button",
            "is-danger"
        ];
        if(isLoading) {
            classes.push("is-loading");
        }
        return classes.join(" ");
    }
    const unsubscribe = async (e: MouseEvent) => {
        e.preventDefault();
        setIsLoading(true);
        const res = await (new APIRequest()).setRoute(`subscriptions/${props.selectedSubscription.id}`)
            .setMethod("DELETE")
            .setData({})
            .send();
        props.onSubscriptionCreated();
    }
    return (
        <div className="section">
            <div className="container">
                <h3 className="title is-4">{props.selectedSubscription.publisher.name}</h3>
                <button disabled={isLoading} type="submit" onClick={unsubscribe} className={buttonClasses()}>Unsubscribe</button>
            </div>
            
        </div>
    )
}

export default Subscription;