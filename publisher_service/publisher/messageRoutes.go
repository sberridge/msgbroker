package main

func messageRoutes() []route {
	return []route{
		{
			RoutePattern: "/publishers/{publication_id}/messages",
			Authenticate: true,
			Method:       "POST",
			Func: func(rd routeData, c chan []byte) {
				c <- handlePublishMessage(rd.Request.Body, rd.MongoService, rd.AuthID, rd.DynamicParams["publication_id"])
			},
		},
	}
}
