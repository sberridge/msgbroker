import React, { useEffect } from "react";
import ReactDOM from "react-dom";
import Header from "./header";
import LoginForm from "./auth/loginForm";
import APIRequest from "./modules/APIRequest";
import ProviderManager from "./provider/providerManager";

const App = () =>{

    let [authId, setAuthId] = React.useState("");
    let [isAuthed, setIsAuthed] = React.useState(false);

    let [isLoading, setIsLoading] = React.useState(true);

    const [lock, setLock] = React.useState(true);

    console.log(isAuthed);
    
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
        console.log(id);
        setAuthId(id);
        setIsAuthed(true);
    }
    

    return <div id="app-container">
        <Header></Header>
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
                <p>Authenticated as: {authId}</p>
                <ProviderManager
                    authId={authId}
                ></ProviderManager>
            </div>
        }
    </div>
}

ReactDOM.render(<App/>,document.getElementById('root'));