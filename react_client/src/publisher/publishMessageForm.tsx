import React, { useState } from "react";
import { Validator } from "../lib/Validator";
import APIRequest from "../modules/APIRequest";
import { statusMessage } from "../types/types";

type publisherMessageFormProps = {
    publisherId: string
}

const PublishMessageForm = (props:publisherMessageFormProps) => {
    const [formValidation,setFormValidation] = useState({
        publish_message_text: null,
        publish_message_expire_ttl: null,
    } as {
        publish_message_text:string|null
        publish_message_expire_ttl:string|null
    })

    const [statusMessage,setStatusMessage] = useState<statusMessage|null>(null);

    const [isLoading, setIsLoading] = useState(false);

    const form = document.getElementById('publish_message_form') as HTMLFormElement;

    const validate = async (data:object):Promise<[boolean, {[key:string]:string}]> => {
        let validator = new Validator(data);
        validator.validateRequired("publish_message_text");
        validator.validateMinLength("publish_message_text", 1);
        validator.validateRequired("publish_message_expire_ttl");
        validator.validateInteger("publish_message_expire_ttl");
        let result = await validator.validate();
        return [validator.success, result];
    }

    const checkIsValidationField = (field:string): field is keyof typeof formValidation => {
        return field in formValidation;
    }
    const handleValidationResult = (result:{[key:string]:string})=>{
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

    const publishMessage = async (data:{message:string,ttl:number}) => {
        setIsLoading(true);
        let response = await (new APIRequest)
            .setRoute(`publishers/${props.publisherId}/messages`)
            .setMethod("POST")
            .setData({
                "ttl": data.ttl,
                "payload": data.message
            })
            .send().catch(e=>{
                setStatusMessage({
                    type: "is-danger",
                    message: "Something went wrong publishing the message, please try again"
                });
            });
        if(response) {
            if(response.success) {
                setStatusMessage({
                    type: "is-success",
                    message: "Message published!"
                });
            } else {
                setStatusMessage({
                    type: "is-danger",
                    message: `Message not published: ${("message" in response) ? response.message : 'Unknown error'}`
                });
            }
        }
        setIsLoading(false);
        form.reset();
    }

    const handlePublishMessageSubmit = async (e:React.FormEvent) => {
        e.preventDefault();
        const data = new FormData(form);
        const values = {
            publish_message_text: data.get("publish_message_text"),
            publish_message_expire_ttl: data.get("publish_message_expire_ttl"),
        };
        let [valid, result] = await validate(values);
        handleValidationResult(result);
        if(!valid) {
            return;
        }
        publishMessage({
            message: values.publish_message_text as string,
            ttl: parseInt(values.publish_message_expire_ttl as string)
        });

        
    }

    return (
        <div>
            <h4 className="title is-5">Publish Message</h4>
            {statusMessage && 
                <div className={`notification ${statusMessage.type}`}>
                    <button className="delete" onClick={()=>{setStatusMessage(null);}}></button>
                    {statusMessage.message}
                </div>
            }
            <form id="publish_message_form" onSubmit={handlePublishMessageSubmit}>
                <div className="field">
                    <label htmlFor="publish-message-text" className="label">Message</label>
                    <div className="control">
                        <textarea id="publish-message-text" name="publish_message_text" className="textarea"></textarea>
                    </div>
                    <p className="help is-danger">{formValidation.publish_message_text ?? <span dangerouslySetInnerHTML={{__html: "&nbsp;"}}></span>}</p>
                </div>
                <div className="field">
                    <label htmlFor="publish-message-expire-ttl" className="label">Time to Live (seconds)</label>
                    <div className="control">
                        <input id="publish-message-expire-ttl" name="publish_message_expire_ttl" type="number" className="input"></input>
                    </div>
                    <p className="help is-danger">{formValidation.publish_message_expire_ttl ?? <span dangerouslySetInnerHTML={{__html: "&nbsp;"}}></span>}</p>
                </div>
                <div className="field">
                    <button disabled={isLoading} className={`button is-primary ${isLoading ? "is-loading" : ""}`}>Publish</button>
                </div>
            </form>
        </div>
    )
}

export default PublishMessageForm;