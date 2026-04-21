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
// [*] Upgrade it to WS once client connects
// [*] Add newly connected ws to server
// [*] Add ws client
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
	clients       map[string]*Client
	mu            *sync.RWMutex
	joinServerCH  chan *Client
	leaveServerCH chan *Client
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		ID:   uuid.NewString(),
		mu:   new(sync.RWMutex),
		conn: conn,
	}
}

func (c *Client)readMsgLoop(leaveServerCH chan<- *Client){
	defer func(){
		c.conn.Close()
		leaveServerCH<- c;
	}()
	for{
		_,b,err:=c.conn.ReadMessage()
		if err!=nil{
			return;
		}
		
	}
}

func NewServer() *Server {
	return &Server{
		clients:       map[string]*Client{},
		mu:            new(sync.RWMutex),
		joinServerCH:  make(chan *Client, 64),
		leaveServerCH: make(chan *Client, 64),
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error on HTTP conn upgrade %v\n", err)
		return
	}

	// add conn to server
	client := NewClient(conn)

	// connecting to the server
	s.joinServerCH <- client


}

func (s *Server) AcceptLoop() {
	for {
		select {
		case c := <-s.joinServerCH:
			// handle join logic
			s.joinServer(c)
		case c := <-s.leaveServerCH:
			// handle leave logic
			s.leaveServer(c)
		}
	}
}

func (s *Server) joinServer(c *Client) {
	s.clients[c.ID] = c
	fmt.Printf("Client join the server CID:%v\n", c.ID)
}

func (s *Server) leaveServer(c *Client) {
	delete(s.clients, c.ID)
	fmt.Printf("Client left the server CID:%v\n", c.ID)
}

func createServer() {
	s := NewServer()
	go s.AcceptLoop()
	http.HandleFunc("/", s.handleWS)
	fmt.Printf("Server Started at port :%v\n", wsPort)
	log.Fatal(http.ListenAndServe(wsPort, nil))
}

func main() {
	createServer()
}
