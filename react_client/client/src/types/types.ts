export type publisherDetails = {
    id: string
    name: string
}

export type appPage = {
    title: string
    key: string
}

export type authedUser = {
    id: string
    name: string
}

export type statusMessage = {
    message: string
    type: string
}

export type subscriptionDetails = {
    id: string
    publisher: {
        id: string
        name: string
        owner_id: string
    }
}

export type formValidation = {[key:string]:string}

export type formValidationResponse =  [boolean, formValidation]