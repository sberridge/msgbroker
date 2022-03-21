package main

func subscriberRoutes() []route {
	return []route{
		{
			RoutePattern: "/subscriptions",
			Method:       "GET",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleGetSubscriptions(rd.AuthID, rd.MongoService)
			},
		},
		{
			RoutePattern: "/subscriptions",
			Method:       "POST",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleSubscribe(rd.Request.Body, rd.AuthID, rd.MongoService)
			},
		},
	}
}
