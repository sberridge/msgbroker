package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/sberridge/bezmongo"
)

type messageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func readBody(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	bytes, err := io.ReadAll(body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return bytes, nil

}

func createMessageResponse(success bool, message string) []byte {
	res, _ := json.Marshal(messageResponse{
		Success: success,
		Message: message,
	})
	return res
}

var store = sessions.NewCookieStore([]byte("rwerwerwer"))

func startServer(wg *sync.WaitGroup, mongo *bezmongo.MongoService) *http.Server {

	server := &http.Server{Addr: ":8080"}

	http.HandleFunc("/auth", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("content-type", "application/json")
		session, _ := store.Get(r, "session")
		switch r.Method {
		case "POST":
			rw.Write(handleAuth(r.Body, mongo, session))
		}
		session.Save(r, rw)
	})

	http.HandleFunc("/register", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("content-type", "application/json")
		session, _ := store.Get(r, "session")
		switch r.Method {
		case "POST":
			rw.Write(handleRegistration(r.Body, mongo, session))
		}
		session.Save(r, rw)
	})

	go func() {
		defer wg.Done()

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	return server
}

func main() {
	mongo, err := bezmongo.StartMongo()

	if err != nil {
		fmt.Println("Couldn't connect to the Mongo service")

	} else {
		httpServerExitDone := &sync.WaitGroup{}
		httpServerExitDone.Add(1)
		server := startServer(httpServerExitDone, mongo)

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)

		// Waiting for SIGINT (kill -2)
		<-stop

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		fmt.Println("Close")
		if err := server.Shutdown(ctx); err != nil {
			// handle err
			fmt.Println(err)
		}
	}
}
