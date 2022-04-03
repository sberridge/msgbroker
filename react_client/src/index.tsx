import "./css/app.scss";
import React, { useEffect, useState } from "react";
import ReactDOM from "react-dom";
import Header from "./header";
import LoginForm from "./auth/loginForm";
import APIRequest from "./modules/APIRequest";
import PublisherManager from "./publisher/publisherManager";
import Menu from "./menu/menu";
import { appPage, authedUser } from "./types/types";
import MessageFeed from "./message-feed/messageFeed";


const appPages:appPage[] = [
    {
        title:"Publishers",
        key: "publishers"
    },
    {
        title: "Subscriptions",
        key: "subscriptions"
    },
    {
        title: "Message Feed",
        key: "feed"
    }
];

const pageMap:Map<string, appPage> = new Map();
appPages.forEach((page)=>{
    pageMap.set(page.key, page);
})

const App = () =>{

    let [authedUser, setAuthedUser] = useState<authedUser|null>(null);
    let [isAuthed, setIsAuthed] = React.useState(false);

    let [isLoading, setIsLoading] = React.useState(true);

    const [currentPage,setCurrentPage] = useState("publishers");
    
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
                console.log(r);
                setIsAuthed(true);
                setAuthedUser({
                    id: r.data.id,
                    name: r.data.name
                });
            } else {
                setIsAuthed(false);
            }
            setIsLoading(false);
        }).catch(err=>{
            console.log(err);
        });
        window.onpopstate = ()=>{
            let url = window.location.search.replace("?","");
            changePage(url);
        }
    }, []);

    
    const onLoggedIn = (authedUser:authedUser) => {
        setAuthedUser(authedUser);
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
            authedUser={authedUser}
        ></Header>
        {isLoading &&
            <p>Please wait...</p>
        }
        {!isLoading && !isAuthed &&
            <LoginForm
                onLoggedIn={onLoggedIn}
            ></LoginForm>
        }
        {authedUser &&
            <div>
                <Menu
                    onPageChange={onPageChange}
                    currentPage={currentPage}
                    pages={appPages}
                ></Menu>
                {currentPage == "publishers" &&
                    <PublisherManager
                        authId={authedUser.id}
                    ></PublisherManager>
                }
                {currentPage == "feed" &&
                    <MessageFeed
                        authedUser={authedUser}
                    ></MessageFeed>
                }
                
            </div>
        }
    </div>
}

ReactDOM.render(<App/>,document.getElementById('root'));