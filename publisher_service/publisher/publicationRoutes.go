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
		case "POST":
			go handleCreatePublication(r.Body, id, mongo, responseChan)
		default:
			http.NotFound(rw, r)
			return
		}
		response := <-responseChan
		rw.Write(response)
	})

	http.HandleFunc("/publications/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("content-type", "application/json")
		session := getSession(r)
		pubId := r.URL.Path[len("/publications/"):]
		if !validGuid(pubId) {
			http.NotFound(rw, r)
			return
		}
		id, authed := checkAuth(session)
		if !authed {
			rw.Write(authResponse())
			return
		}

		responseChan := make(chan []byte)

		switch r.Method {
		case "DELETE":
			go handleDeletePublication(pubId, id, mongo, responseChan)
		default:
			http.NotFound(rw, r)
			return
		}

		response := <-responseChan
		rw.Write(response)
	})
}
