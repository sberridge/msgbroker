package main

import (
	"net/http"

	"github.com/sberridge/bezmongo"
)

func publicationRoutes(server *http.Server, mongo *bezmongo.MongoService) {
	http.HandleFunc("/publications", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("content-type", "application/json")
		session := getSession(r)

		id, authed := checkAuth(session)
		if !authed {
			rw.Write(authResponse())
			return
		}

		responseChan := make(chan []byte)

		switch r.Method {
		case "GET":
			go handleGetPublications(mongo, id, r.URL.Query(), responseChan)
			response := <-responseChan
			rw.Write(response)
		case "POST":
			go handleCreatePublication(r.Body, id, mongo, responseChan)
			response := <-responseChan
			rw.Write(response)
		}

	})
}
