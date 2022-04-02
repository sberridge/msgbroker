import React from "react";
import {appPage} from "./../types/types";
type menuProps = {
    onPageChange: (page:string)=>void
    currentPage: string
    pages: appPage[]
}

const Menu = (props:menuProps) => {
    const renderPageLinks = () => {
        return props.pages.map((page)=>{
            return <li key={page.key} className={props.currentPage == page.key ? "is-active": ""}><a onClick={()=>{props.onPageChange(page.key)}}>{page.title}</a></li>
        })
    }
    return (
        <div className="tabs">
            <ul>
                {renderPageLinks()}
            </ul>
        </div>
    )
}

export default Menu;