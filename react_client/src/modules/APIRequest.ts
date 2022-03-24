type requestMethod = "GET" | "POST" | "DELETE" | "PUT";
export default class APIRequest {
    private static endpoint = "http://localhost:8081";
    private route: string = "";
    private method: requestMethod = "GET";
    private data: any;

    public setRoute(route:string) {
        if(route.substring(0,1) == "/") {
            route = route.substring(1, route.length);
        }
        this.route = route;
        return this;
    }
    
    public setMethod(method:requestMethod) {
        this.method = method;
        return this;
    }

    public setData(data:any) {
        this.data = data;
        return this;
    }

    public async send() {
        let options:RequestInit = {
            method: this.method,
            credentials: "include"
        }
        if(["POST", "PUT"].includes(this.method)) {
            options.headers = {
                "Content-Type": "application/json"
            };            
            if(this.data) {
                options.body = JSON.stringify(this.data);
            } else {
                options.body = "{}";
            }
        }
        return await fetch(`${APIRequest.endpoint}/${this.route}`, options)
            .then(res=>res.json());
    }

}