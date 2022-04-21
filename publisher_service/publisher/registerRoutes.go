package main

func registerRoutes() []route {
	return []route{
		{
			RoutePattern: "/register",
			Method:       "POST",
			Authenticate: false,
			Func: func(rd routeData, c chan []byte) {
				c <- handleRegistration(rd.Request.Body, rd.MongoService, rd.Session)
			},
		},
	}
}
