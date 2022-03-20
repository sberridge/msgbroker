package main

func publicationRoutes() []route {
	return []route{
		{
			RoutePattern: "/publications",
			Method:       "GET",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleGetPublications(rd.MongoService, rd.AuthID, rd.Request.URL.Query())
			},
		},
		{
			RoutePattern: "/publications",
			Method:       "POST",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleCreatePublication(rd.Request.Body, rd.AuthID, rd.MongoService)
			},
		},
		{
			RoutePattern: "/publications/{publication_id}",
			Method:       "DELETE",
			Authenticate: true,
			Func: func(rd routeData, c chan []byte) {
				c <- handleDeletePublication(rd.DynamicParams["publication_id"], rd.AuthID, rd.MongoService)
			},
		},
	}
}
