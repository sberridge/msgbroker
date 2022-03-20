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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/sberridge/bezmongo"
)

const messageBrokerDb = "message-broker"

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

var store = sessions.NewCookieStore([]byte("rwerwerwer"))

func getSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, "session")
	return session
}

func separateRoute(url string) []string {
	path := strings.Split(url, "/")
	if path[0] == "" {
		path = path[1:]
	}
	if path[len(path)-1] == "" {
		path = path[:len(path)-1]
	}
	return path
}

type routeData struct {
	Request       *http.Request
	MongoService  *bezmongo.MongoService
	Session       *sessions.Session
	AuthID        string
	DynamicParams map[string]string
}

type route struct {
	RoutePattern string
	routeParts   []string
	Method       string
	Authenticate bool
	Func         func(routeData, chan []byte)
}

func (route *route) GetDynamicParams(url string) map[string]string {
	urlParts := separateRoute(url)
	dynamicParams := make(map[string]string)
	if len(urlParts) != len(route.routeParts) {
		return dynamicParams
	}
	for i, part := range urlParts {
		correPart := route.routeParts[i]
		if string(correPart[0]) == "{" && string(correPart[len(correPart)-1]) == "}" {
			dynamicUrlParamName := correPart[1 : len(correPart)-1]
			dynamicParams[dynamicUrlParamName] = part
		}
	}
	return dynamicParams
}

func (route *route) Match(url string, method string) bool {
	if route.Method != method {
		return false
	}
	urlParts := separateRoute(url)
	if route.routeParts == nil {
		route.routeParts = separateRoute(route.RoutePattern)
	}
	if len(route.routeParts) != len(urlParts) {
		return false
	}
	for i, part := range urlParts {
		correPart := route.routeParts[i]
		if string(correPart[0]) == "{" && string(correPart[len(correPart)-1]) == "}" {
			continue
		} else if correPart != part {
			return false
		}
	}
	return true

}

var routes []route

func createResponseChannel() chan []byte {
	return make(chan []byte)
}

func matchRoute(url string, method string) (route, bool) {
	for _, route := range routes {
		if route.Match(url, method) {
			return route, true
		}
	}
	return route{}, false
}

func startServer(wg *sync.WaitGroup, mongo *bezmongo.MongoService) *http.Server {

	server := &http.Server{Addr: ":8080"}

	routes = append(routes, authRoutes()...)

	routes = append(routes, registerRoutes()...)

	routes = append(routes, publicationRoutes()...)

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {

		route, found := matchRoute(r.URL.Path, r.Method)

		if !found {
			http.NotFound(rw, r)
			return
		}

		session := getSession(r)

		rd := routeData{
			Request:       r,
			MongoService:  mongo,
			Session:       session,
			DynamicParams: route.GetDynamicParams(r.URL.Path),
		}

		if route.Authenticate {
			id, authed := checkAuth(session)
			if !authed {
				rw.WriteHeader(http.StatusForbidden)
				rw.Write(createMessageResponse(false, "Forbidden >:("))
				return
			}
			rd.AuthID = id
		}

		channel := createResponseChannel()

		go route.Func(rd, channel)

		response := <-channel
		session.Save(r, rw)
		rw.Write(response)
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
