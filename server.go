package main

import (
	"context"
	"flag"
	"fmt"
	gohandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket" // go get github.com/gorilla/websocket
	"github.com/nicholasjackson/env"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

// TODO: The access to the clients map is not concurrency-safe.
// Try using sync.Map
var bindAddress = env.String("BIND_ADDRESS", false, ":9090", "Bind address for the server")
var clients = make(map[*websocket.Conn]bool)  // global : currently connected clients
var broadcast = make(chan Message)            // global: broadcast message queue/ channel
var connectionUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // normal HTTP connections to websocket

// Message will contains details of user
// Email can be used to fetch unique gravatar
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	// simple static file server binded to root
	l := log.New(os.Stdout, "products-api ", log.LstdFlags)
	sm := mux.NewRouter()
	getR := sm.Methods(http.MethodGet).Subrouter()
	getR.HandleFunc("/ws", handleWSConnections)
	getR.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("work")
	})
	// configure websocket route

	// takes messages from the broadcast channel from before and pass them to clients
	go handleMessages()
	// starts the web server on localhost and log any errors
	flag.Parse()


	ch := gohandlers.CORS(gohandlers.AllowedOrigins([]string{"localhost:3001"}))
	s := http.Server{
		Addr:         "localhost:9090",      // configure the bind address
		Handler:      ch(sm),            // set the default handler
		ErrorLog:     l,                 // set the logger for the server
		ReadTimeout:  5 * time.Second,   // max time to read request from the client
		WriteTimeout: 10 * time.Second,  // max time to write response to the client
		IdleTimeout:  120 * time.Second, // max time for connections using TCP Keep-Alive
	}

	// start the server
	go func() {
		l.Println("Starting server on port 9090")

		err := s.ListenAndServe()
		if err != nil {
			l.Printf("Error starting server: %s\n", err)
			os.Exit(1)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// Block until a signal is received.
	sig := <-c
	log.Println("Got signal:", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(ctx)

}

func handleWSConnections(res http.ResponseWriter, req *http.Request) {
	// will run for each web request and run indefinitely until there is some error
	// from client (we assume that the connection is closed)

	// upgrade initial GET request to a websocket
	ws, err := connectionUpgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Fatalln("handleWSConnections:upgrader,Upgrade", err)
		// close the connection before function returns
		defer ws.Close()

	}

	// registering the client in global map
	clients[ws] = true

	// infinite loop
	for {
		var msg Message
		// read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Println("handleWSConnections:ReadJSON:", err)
			delete(clients, ws)
			break
		}
		// send the newly received message to the broadcast message queue
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// grab the next message from the message queue
		msg := <-broadcast
		//newMsg := "echo " + msg.Email
		// send ot out to every client that is currently connected
		out, err := exec.Command("node","test.js").Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("The date is: \n%s\n%s\n", out,msg)
		//for client := range clients {
		//	err := client.WriteJSON(msg)
		//	if err != nil {
		//		log.Println("handleMessages:WriteJSON", err)
		//		client.Close()
		//		delete(clients, client)
		//	}
		//}
	}
}