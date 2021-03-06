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
import RegisterForm from "./auth/registerForm";
import SubscriptionManager from "./subscription/subscriptionManager";


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
        if(!(document as any).createDocumentTransition) {
            setCurrentPage(page);
            return true;
        }
        const transition = (document as any).createDocumentTransition();
        transition.start(()=>setCurrentPage(page));
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
            <div>
                <LoginForm
                    onLoggedIn={onLoggedIn}
                ></LoginForm>
                <RegisterForm
                    onRegistered={onLoggedIn}
                ></RegisterForm>
            </div>
        }
        {authedUser &&
            <div>
                <Menu
                    onPageChange={onPageChange}
                    currentPage={currentPage}
                    pages={appPages}
                ></Menu>
                
                <div className="columns">
                    <div className="column is-two-thirds">
                        <div className="app-page">
                            {currentPage == "publishers" &&
                                <PublisherManager
                                    authId={authedUser.id}
                                ></PublisherManager>
                            }
                            {currentPage == "subscriptions" &&
                                <SubscriptionManager
                                    authId={authedUser.id}
                                ></SubscriptionManager>
                            }
                        </div>
                    </div>
                    <div className="column">
                        <MessageFeed
                            authedUser={authedUser}
                        ></MessageFeed>
                    </div>
                </div>
            </div>
        }
    </div>
}

ReactDOM.render(<App/>,document.getElementById('root'));