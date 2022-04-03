import React, { useState } from "react";
import { Validator } from "../lib/Validator";
import APIRequest from "../modules/APIRequest";

import { statusMessage } from "./../types/types"

type newPublisherProps = {
    onPublisherCreated: ()=>void
}



const NewPublisherForm = (props:newPublisherProps) => {

    const [formValidation,setFormValidation] = useState({
        publisher_name: null
    } as {
        publisher_name:string|null
    })

    const [statusMessage,setStatusMessage] = useState<statusMessage|null>(null);

    const [isLoading, setIsLoading] = useState(false);

    const form = document.getElementById('new_publisher_form') as HTMLFormElement;

    const validate = async (data:object):Promise<[boolean, {[key:string]:string}]> => {
        let validator = new Validator(data);
        validator.validateRequired("publisher_name");
        validator.validateMinLength("publisher_name", 1);
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

    const createPublisher = async (data:{publisher_name:string}) => {
        setIsLoading(true);
        const request = new APIRequest();
        const createResult = await request.setRoute("publishers")
            .setMethod("POST")
            .setData({
                "name": data.publisher_name
            })
            .send().catch((e)=>{
                setStatusMessage({
                    type: "is-danger",
                    message: "Something went wrong creating the publisher, please try again"
                })
            });
        if(createResult) {
            if(createResult.success) {
                setStatusMessage({
                    type: "is-success",
                    message: "Publisher Created"
                });
                props.onPublisherCreated();
                form.reset();
            } else {
                setStatusMessage({
                    type: "is-danger",
                    message: `Publisher not created: ${("message" in createResult) ? createResult.message : 'Unknown error'}`
                });
            }
        }
        setIsLoading(false);
    }
    
    const handleSubmit = async (e:React.FormEvent) => {
        e.preventDefault();
        const data = new FormData(form);
        const values = {
            publisher_name: data.get("publisher_name")
        };
        let [valid, result] = await validate(values);
        handleValidationResult(result);
        if(!valid) {
            return;
        }
        createPublisher({
            publisher_name: values.publisher_name as string
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
                <h3 className="title is-4">New Publisher</h3>
                {statusMessage && 
                    <div className={`notification ${statusMessage.type}`}>
                        <button className="delete" onClick={()=>{setStatusMessage(null);}}></button>
                        {statusMessage.message}
                    </div>
                }
                <form id="new_publisher_form" onSubmit={handleSubmit}>
                    <div className="field">
                        <label className="label" htmlFor="new_publisher_name">Publisher Name</label>
                        <div className="control">
                            <input className="input" type="text" name="publisher_name" id="new_publisher_name"></input>
                        </div>
                        <p className="help is-danger">{formValidation.publisher_name ?? <span dangerouslySetInnerHTML={{__html: "&nbsp;"}}></span>}</p>
                        
                    </div>
                    <button disabled={isLoading} type="submit" className={buttonClasses()}>Create Publisher</button>
                </form>
            </div>
        </div>
    
    )

}

export default NewPublisherForm;