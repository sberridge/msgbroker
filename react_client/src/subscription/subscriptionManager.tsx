import React, { useEffect, useState } from "react";
import APIRequest from "../modules/APIRequest";
import { subscriptionDetails } from './../types/types'
import NewSubscriptionForm from "./newSubscriptionForm";
import Subscription from "./subscription";
import SubscriptionList from './subscriptionList'

type subscriptionManagerProps = {
    authId: string
};


const SubscriptionManager = (props:subscriptionManagerProps) => {

    const [subscriptions, setSubscriptions] = useState<subscriptionDetails[]>([])
    const [selectedSubscription, setSelectedSubscription] = useState<null|subscriptionDetails>(null);

    const loadSubscriptions = async () =>{
        let request = new APIRequest();
        request.setRoute("/subscriptions")
        let result = await request.send();
        if(result.success) {
            setSubscriptions(result.subscriptions);
        }
    }

    useEffect(()=>{
        loadSubscriptions();
    },[]);

    const onSubscriptionCreated = () => {
        loadSubscriptions();
    }
    
    const onSubscriptionDeleted = () => {
        setSelectedSubscription(null);
        loadSubscriptions();
    }

    const onSelectSubscription = (subscription:subscriptionDetails|null)=>{
        setSelectedSubscription(subscription);
    }

    return (
        <section className="section">
            <h2 className="title is-3">Subscription Management</h2>
            <div className="columns">
                <div className="column is-narrow">
                    <SubscriptionList
                        subscriptions={subscriptions}
                        selectedSubscription={selectedSubscription}
                        onSelectSubscription={onSelectSubscription}
                    ></SubscriptionList>
                </div>
                <div className="column">
                    {!selectedSubscription && 
                        <NewSubscriptionForm
                            onSubscriptionCreated={onSubscriptionCreated}
                        ></NewSubscriptionForm>
                    }
                    {selectedSubscription &&
                        <Subscription
                            selectedSubscription={selectedSubscription}
                            onSubscriptionCreated={onSubscriptionDeleted}
                        ></Subscription>
                    }                 
                </div>
                
                
                
                
        
            </div>
        </section>
    )
}

export default SubscriptionManager;