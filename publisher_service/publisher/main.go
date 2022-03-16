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

func checkAuth(session *sessions.Session) (string, bool) {
	id := session.Values["auth_id"]
	switch v := id.(type) {
	case string:
		return v, true
	}
	return "", false
}

func authResponse() []byte {
	b, _ := json.Marshal(messageResponse{
		Success: false,
		Message: "Authentication failed",
	})
	return b
}

var store = sessions.NewCookieStore([]byte("rwerwerwer"))

func getSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, "session")
	return session
}

func startServer(wg *sync.WaitGroup, mongo *bezmongo.MongoService) *http.Server {

	server := &http.Server{Addr: ":8080"}

	authRoutes(server, mongo)

	registerRoutes(server, mongo)

	publicationRoutes(server, mongo)

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
