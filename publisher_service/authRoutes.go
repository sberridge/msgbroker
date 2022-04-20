package main

func authRoutes() []route {
	return []route{
		{
			Authenticate: false,
			RoutePattern: "/auth",
			Method:       "POST",
			Func: func(data routeData, responseChan chan []byte) {
				responseChan <- handleAuth(data.Request.Body, data.MongoService, data.Session)
			},
		},
		{
			Authenticate: false,
			RoutePattern: "/auth",
			Method:       "GET",
			Func: func(data routeData, responseChan chan []byte) {
				responseChan <- handleCheckAuth(data.Session, data.MongoService)
			},
		},
	}
}
