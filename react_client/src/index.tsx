import "./css/app.scss";
import React, { useEffect } from "react";
import ReactDOM from "react-dom";
import Header from "./header";
import LoginForm from "./auth/loginForm";
import APIRequest from "./modules/APIRequest";
import PublisherManager from "./publisher/publisherManager";

const App = () =>{

    let [authId, setAuthId] = React.useState("");
    let [isAuthed, setIsAuthed] = React.useState(false);

    let [isLoading, setIsLoading] = React.useState(true);

    const [lock, setLock] = React.useState(true);
    
    const checkAuth = async () =>{  
        const authRes = await (new APIRequest)
            .setRoute("/auth")
            .setMethod("GET")
            .send();
        return authRes;       
    }

    useEffect(()=>{
        checkAuth().then((r)=>{
            if(r.success) {
                setIsAuthed(true);
                setAuthId(r.data.id);
            } else {
                setIsAuthed(false);
            }
            setIsLoading(false);
        }).catch(err=>{
            console.log(err);
        });
    }, [lock]);

    
    const onLoggedIn = (id:string) => {
        setAuthId(id);
        setIsAuthed(true);
    }
    

    return <div id="app-container">
        <Header
            isAuthed={isAuthed}
            authID={authId}
        ></Header>
        {isLoading &&
            <p>Please wait...</p>
        }
        {!isLoading && !isAuthed &&
            <LoginForm
                onLoggedIn={onLoggedIn}
            ></LoginForm>
        }
        {isAuthed &&
            <div>
                <div className="tabs">
                    <ul>
                        <li className="is-active"><a>Publishers</a></li>
                        <li><a>Subscriptions</a></li>
                    </ul>
                </div>
                <PublisherManager
                    authId={authId}
                ></PublisherManager>
            </div>
        }
    </div>
}

ReactDOM.render(<App/>,document.getElementById('root'));