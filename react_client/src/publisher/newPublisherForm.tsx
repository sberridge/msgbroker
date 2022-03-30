import React from "react";
import { Validator } from "../lib/Validator";

type newPublisherProps = {
    onPublisherCreated: ()=>void
}



const NewPublisherForm = (props:newPublisherProps) => {
    const form = document.getElementById('new_publisher_form') as HTMLFormElement;
    const validate = async (data:object) => {
        let validator = new Validator(data);
        validator.validateRequired("name");
        validator.validateMinLength("name", 1);
        let result = await validator.validate();

        console.log(result);
    }
    const handleSubmit = async (e:React.FormEvent) => {
        e.preventDefault();
        const data = new FormData(form);
        const values = {
            name: data.get("publisher_name")
        };
        let valid = await validate(values);
    }

    return <form id="new_publisher_form" onSubmit={handleSubmit}>
        <label htmlFor="new_publisher_name">Publisher Name</label>
        <input type="text" name="publisher_name" id="new_publisher_name"></input>
        <button type="submit">Create Publisher</button>
    </form>

}

export default NewPublisherForm;