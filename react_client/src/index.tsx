import "./css/app.scss";
import React, { useEffect, useState } from "react";
import ReactDOM from "react-dom";
import Header from "./header";
import LoginForm from "./auth/loginForm";
import APIRequest from "./modules/APIRequest";
import PublisherManager from "./publisher/publisherManager";
import Menu from "./menu/menu";
import { appPage } from "./types/types";


const appPages:appPage[] = [
    {
        title:"Publishers",
        key: "publishers"
    },
    {
        title: "Subscriptions",
        key: "subscriptions"
    }
];

const pageMap:Map<string, appPage> = new Map();
appPages.forEach((page)=>{
    pageMap.set(page.key, page);
})

const App = () =>{

    let [authId, setAuthId] = React.useState("");
    let [isAuthed, setIsAuthed] = React.useState(false);

    let [isLoading, setIsLoading] = React.useState(true);

    const [currentPage,setCurrentPage] = useState("publishers");

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
        window.onpopstate = (e)=>{
            console.log(e);
            let url = window.location.search.replace("?","");
            changePage(url);
        }
    }, [lock]);

    
    const onLoggedIn = (id:string) => {
        setAuthId(id);
        setIsAuthed(true);
    }

    const changePage = (page:string) => {
        if(!pageMap.has(page)) return false;
        setCurrentPage(page);
        return true;
    }

    const onPageChange = (page:string) => {
        if(changePage(page)) {
            window.history.pushState({},"",`/?${page}`);
        }
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
                <Menu
                    onPageChange={onPageChange}
                    currentPage={currentPage}
                    pages={appPages}
                ></Menu>
                {currentPage == "publishers" &&
                    <PublisherManager
                        authId={authId}
                    ></PublisherManager>
                }
                
            </div>
        }
    </div>
}

ReactDOM.render(<App/>,document.getElementById('root'));