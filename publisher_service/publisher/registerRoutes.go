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
	/* http.HandleFunc("/register", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("content-type", "application/json")

		session := getSession(r)

		responseChan := make(chan []byte)

		switch r.Method {
		case "POST":
			go handleRegistration(r.Body, mongo, session, responseChan)
		}

		response := <-responseChan
		session.Save(r, rw)
		rw.Write(response)
	}) */
}
