import React from "react";
import renderer, { ReactTestRendererJSON } from "react-test-renderer";
import Menu from "../menu";
import {appPage} from "../../types/types"

type props = {
    onPageChange:(page:string)=>void
    currentPage:string
    pages:appPage[]
}
const createInstance = (props:props)=>{
    return <Menu
        onPageChange={props.onPageChange}
        currentPage={props.currentPage}
        pages={props.pages}
    ></Menu>
}

it("should render",()=>{
    let currentPage = "subscriptions";
    const onPageChange = (page:string)=>{
       comp.update(createInstance({
           onPageChange:onPageChange,
           currentPage:page,
           pages:appPages
       }))
    }
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
    const comp = renderer.create(
        createInstance({
            onPageChange:onPageChange,
            currentPage:currentPage,
            pages:appPages
        })
    )

    const getNestedChild = (el:ReactTestRendererJSON, levels:number) : ReactTestRendererJSON => {
        if(levels > 0) {
            if(el.children) {
                return getNestedChild(el.children[0] as ReactTestRendererJSON, --levels);
            }
        }
        return el;
    }

    let tree = comp.toJSON() as ReactTestRendererJSON;
    expect(tree).toMatchSnapshot();
    renderer.act(()=>{
        let a = getNestedChild(tree, 3);
        a.props.onClick();
    })

    tree = comp.toJSON() as ReactTestRendererJSON;
    expect(tree).toMatchSnapshot();

})
