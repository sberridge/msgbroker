import React from "react";
import { authedUser } from "./types/types";
type headerProps = {
    isAuthed: boolean
    authedUser: authedUser | null
}
const Header = (props:headerProps) => {
    return <header className="navbar main-nav has-background-dark">
        <div className="navbar-brand">
            <div className="navbar-item">
                <h1 className="title is-2 has-text-light">Message Broker</h1>
            </div>
        </div>
        {props.authedUser &&
            <div className="navbar-end">
                <div className="navbar-item auth-user-details">
                    <h4 className="title is-4"><strong className="has-text-light">{props.authedUser.name}</strong></h4>
                    <h5 className="subtitle"><strong className="has-text-light">{props.authedUser.id}</strong></h5>
                </div>
            </div>
        }
        
    </header>
}

export default Header