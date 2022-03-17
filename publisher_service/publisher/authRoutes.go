package main

import (
	"net/http"

	"github.com/sberridge/bezmongo"
)

func authRoutes(server *http.Server, mongo *bezmongo.MongoService) {
	http.HandleFunc("/auth", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("content-type", "application/json")
		session := getSession(r)
		responseChan := make(chan []byte)
		switch r.Method {
		case "POST":
			go handleAuth(r.Body, mongo, session, responseChan)

		}

		response := <-responseChan
		session.Save(r, rw)
		rw.Write(response)

	})
}
