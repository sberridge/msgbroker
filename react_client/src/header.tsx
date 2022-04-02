import React from "react";
type headerProps = {
    isAuthed: boolean
    authID: string
}
const Header = (props:headerProps) => {
    return <header className="navbar main-nav has-background-dark">
        <div className="navbar-brand">
            <div className="navbar-item">
                <h1 className="title is-1 has-text-light">Message Broker</h1>
            </div>
        </div>
        {props.isAuthed &&
            <div className="navbar-end">
                <div className="navbar-item">
                    <p><strong className="has-text-light">{props.authID}</strong></p>
                </div>
            </div>
        }
        
    </header>
}

export default Header