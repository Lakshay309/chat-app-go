package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// TODO
// [*] HTTP server
// [] Upgrade it to WS once client connects
// [] Add newly connected ws to server
// [] Add ws client
// [] Remove client on disconnect
// [] send brodcast msg -> no race condition

var (
	wsPort = ":3223"
)

type Client struct {
	ID   string
	mu   *sync.RWMutex
	conn *websocket.Conn
}

type Server struct {
	clients []*Client
	mu      *sync.RWMutex
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		ID:   uuid.NewString(),
		mu:   new(sync.RWMutex),
		conn: conn,
	}
}

func NewServer() *Server {
	return &Server{
		clients: []*Client{},
		mu:      new(sync.RWMutex),
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	upgrader :=websocket.Upgrader{
		ReadBufferSize: 512,
		WriteBufferSize: 512,
		CheckOrigin: func(r *http.Request) bool {return true},
	}
	_,err:=upgrader.Upgrade(w,r,nil)
	if err!=nil{
		fmt.Println("error on http conn upgrade %v\n",err)
		return;
	}
	
}

func main() {
	http.HandleFunc("/", handleWS)

	log.Fatal(http.ListenAndServe(wsPort, nil))
}
