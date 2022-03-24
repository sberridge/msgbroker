import React from "react";
import APIRequest from "./../modules/APIRequest";

type loginFormProps = {
    onLoggedIn: (id:string)=>void
}

const LoginForm = (props:loginFormProps) => {

    
    const [loginId, setLoginId] = React.useState("");

    const attemptLogin = async () => {
        const id = loginId;
        const res = await (new APIRequest)
                        .setRoute("/auth")
                        .setMethod("POST")
                        .setData({
                            id: id
                        })
                        .send();
        return res;
    }

    const handleLoginSubmit = (event:React.FormEvent) => {
        event.preventDefault();
        let success = false;
        attemptLogin().then((res)=>{
            console.log(res);
            if(res.success) {
                props.onLoggedIn(res.data.id)
            }
        }).catch(err=>{
            console.log(err);
        });
    }

    return <form onSubmit={handleLoginSubmit}>
        <label htmlFor="login_id_input">ID</label>
        <input id="login_id_input" type="text" onChange={(e)=>{
            setLoginId(e.target.value);
        }}></input>
        <button type="submit">Login</button>
    </form>
}

export default LoginForm