package main

func publicationRoutes() []route {
	return []route{
		{
			RoutePattern: "/publishers",
			Method:       "GET",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleGetPublications(rd.MongoService, rd.AuthID, rd.Request.URL.Query())
			},
		},
		{
			RoutePattern: "/publishers",
			Method:       "POST",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleCreatePublisher(rd.Request.Body, rd.AuthID, rd.MongoService)
			},
		},
		{
			RoutePattern: "/publishers/{publisher_id}",
			Method:       "DELETE",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleDeletePublisher(rd.DynamicParams["publisher_id"], rd.AuthID, rd.MongoService)
			},
		},
		{
			RoutePattern: "/publishers/{publisher_id}/subscribers",
			Method:       "GET",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleGetPublisherSubscribers(rd.DynamicParams["publisher_id"], rd.AuthID, rd.MongoService)
			},
		},
	}
}
