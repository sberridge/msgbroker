import React from "react";

const NewPublisherForm = () => {
    const form = document.getElementById('new_publisher_form') as HTMLFormElement;
    console.log(form);
    return <form id="new_publisher_form" onSubmit={(e:React.FormEvent)=>{
        e.preventDefault();
        
    }}>
        <label htmlFor="new_publisher_name">Publisher Name</label>
        <input type="text" name="publisher_name" id="new_publisher_name"></input>
        <button type="submit">Create Publisher</button>
    </form>

}

export default NewPublisherForm;