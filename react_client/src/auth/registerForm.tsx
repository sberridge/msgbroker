import React, { useEffect, useState } from "react";
import { authedUser, formValidation, formValidationResponse } from "../types/types";
import { statusMessage } from "./../types/types"
import APIRequest from "./../modules/APIRequest";
import { Validator } from "../lib/Validator";

type registerFormProps = {
    onRegistered: (authedUser:authedUser)=>void
}

const RegisterForm = (props:registerFormProps) => {

    const [formValidation,setFormValidation] = useState({
        name: null
    } as {
        name:string|null
    });

    const [statusMessage,setStatusMessage] = useState<statusMessage|null>(null)
    const [isLoading,setIsLoading] = useState(false);
    let form = document.getElementById("register_form") as HTMLFormElement;
    useEffect(()=>{
        form = document.getElementById("register_form") as HTMLFormElement
    },[]);

    const attemptRegistration = async (data:{name:string}) => {
        setIsLoading(true);
        const registerResult = await (new APIRequest())
            .setRoute("register")
            .setMethod("POST")
            .setData({
                "name": data.name
            })
            .send().catch((e)=>{
                setStatusMessage({
                    type: "is-danger",
                    message: "Something went wrong registering, please try again"
                });
            })
        if(!registerResult) return;

        if(registerResult.success) {
            props.onRegistered({
                id: registerResult.row.id,
                name: data.name
            });
        } else {
            setStatusMessage({
                type: "is-danger",
                message: `Registration failed: ${("message" in registerResult) ? registerResult.message : 'Unknown error'}`
            });
            setIsLoading(false);
        }
        
    }

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
        validator.validateRequired("name");
        validator.validateMinLength("name", 1);
        let result = await validator.validate();
        return [validator.success, result];
    }

    const handleRegisterSubmit = async (event:React.FormEvent) => {
        event.preventDefault();
        const data = new FormData(form);
        const values = {
            name: data.get("name")
        }
        let [valid, result] = await validate(values);
        handleValidationResult(result);
        if(!valid) {
            return;
        }
        attemptRegistration({
            name: values.name as string
        })
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
        <section className="section">
            <div className="container">
                <h2 className="title is-3">Register</h2>
                {statusMessage && 
                    <div className={`notification ${statusMessage.type}`}>
                        <button className="delete" onClick={()=>{setStatusMessage(null);}}></button>
                        {statusMessage.message}
                    </div>
                }
                <form id="register_form" onSubmit={handleRegisterSubmit}>
                    <div className="field">
                        <label className="label is-small" htmlFor="register_name_input">Name</label>
                        <div className="control">
                            <input className="input is-small" name="name" id="register_name_input" type="text"></input>
                        </div>
                        <p className="help is-danger">{formValidation.name ?? <span dangerouslySetInnerHTML={{__html: "&nbsp;"}}></span>}</p>
                    </div>
                    <div className="control">
                        <button disabled={isLoading} className={buttonClasses()} type="submit">Register</button>
                    </div>
                </form>
            </div>
        </section>
    )
}

export default RegisterForm