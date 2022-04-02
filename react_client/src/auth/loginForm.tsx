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
        attemptLogin().then((res)=>{
            if(res.success) {
                props.onLoggedIn(res.data.id)
            }
        }).catch(err=>{
            console.log(err);
        });
    }

    return (
        <section className="section">
            <div className="container">
                <h2 className="title is-3">Login</h2>
                <form onSubmit={handleLoginSubmit}>
                    <div className="field">
                        <label className="label is-small" htmlFor="login_id_input">ID</label>
                        <div className="control">
                            <input className="input is-small" id="login_id_input" type="text" onChange={(e)=>{
                                setLoginId(e.target.value);
                            }}></input>
                        </div>
                    </div>
                    <div className="control">
                        <button className="button is-primary" type="submit">Login</button>
                    </div>
                </form>
            </div>
        </section>
    )
}

export default LoginForm