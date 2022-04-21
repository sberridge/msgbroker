import React, { FormEvent, useState } from "react";
import { Validator } from "../lib/Validator";
import APIRequest from "../modules/APIRequest";
import { formValidation, formValidationResponse, statusMessage } from "../types/types";

type newSubscriptionFormProps = {
    onSubscriptionCreated: ()=>void
}

const NewSubscriptionForm = (props:newSubscriptionFormProps) => {

    const form = document.getElementById('new_subscription_form') as HTMLFormElement;
    const [statusMessage, setStatusMessage] = useState<statusMessage|null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [formValidation, setFormValidation] = useState({
        "publisher_id": null
    } as {
        "publisher_id": string | null
    });

    const checkIsValidationField = (field:string): field is keyof typeof formValidation => {
        return field in formValidation;
    }
    const handleValidationResult = (result:formValidation)=>{
        let newState = {...formValidation};
        for(let key in newState) {
            if(checkIsValidationField(key)) {
                if(key in result) {
                    newState[key] = result[key];
                } else {
                    newState[key] = null;
                }                
            }
        }
        setFormValidation(newState);
    }

    const validate = async (data:object):Promise<formValidationResponse> => {
        let validator = new Validator(data);
        validator.validateRequired("publisher_id");
        validator.validateMinLength("publisher_id", 1);
        let result = await validator.validate();
        return [validator.success, result];
    }

    const createSubscription = async (data:{publisher_id:string}) => {
        setIsLoading(true);
        const request = new APIRequest();
        const createResult = await request.setRoute("subscriptions")
            .setMethod("POST")
            .setData({
                "publisher_id": data.publisher_id
            })
            .send().catch((e)=>{
                setStatusMessage({
                    type: "is-danger",
                    message: "Something went wrong subscribing, please try again"
                })
            });
        if(createResult) {
            if(createResult.success) {
                setStatusMessage({
                    type: "is-success",
                    message: "Subscribed"
                });
                props.onSubscriptionCreated();
                form.reset();
            } else {
                setStatusMessage({
                    type: "is-danger",
                    message: `Not subscribed: ${("message" in createResult) ? createResult.message : 'Unknown error'}`
                });
            }
        }
        setIsLoading(false);
    }
    
    const handleSubmit = async (e:FormEvent)=>{

        e.preventDefault();
        const data = new FormData(form);
        const values = {
            publisher_id: data.get("publisher_id")
        };
        let [valid, result] = await validate(values);
        handleValidationResult(result);
        if(!valid) {
            return;
        }
        createSubscription({
            publisher_id: values.publisher_id as string
        });
        
    }

    const buttonClasses = () => {
        let classes = [
            "button",
            "is-primary"
        ];
        if(isLoading) {
            classes.push("is-loading");
        }
        return classes.join(" ");
    }

    return (
        <div className="section">
            <div className="container">
                <h3 className="title is-4">New Subscription</h3>
                {statusMessage && 
                    <div className={`notification ${statusMessage.type}`}>
                        <button className="delete" onClick={()=>{setStatusMessage(null);}}></button>
                        {statusMessage.message}
                    </div>
                }
                <form id="new_subscription_form" onSubmit={handleSubmit}>
                    <div className="field">
                        <label className="label" htmlFor="publisher_id">Publisher ID</label>
                        <div className="control">
                            <input className="input" type="text" name="publisher_id" id="publisher_id"></input>
                        </div>
                        <p className="help is-danger">{formValidation.publisher_id ?? <span dangerouslySetInnerHTML={{__html: "&nbsp;"}}></span>}</p>
                        
                    </div>
                    <button disabled={isLoading} type="submit" className={buttonClasses()}>Create Publisher</button>
                </form>
            </div>
        </div>
    )
}

export default NewSubscriptionForm;