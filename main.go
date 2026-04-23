package main

import (
	"encoding/json"
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

type MsgType string

const (
	MsgType_Brodcast MsgType = "broadcast"
)

type ReqMsg struct {
	MsgType MsgType
	Client  *Client
	Data    string
}

type RespMsg struct {
	MsgType  MsgType
	Data     string
	SenderID string
}

func NewRespMsg(msg *ReqMsg) *RespMsg {
	return &RespMsg{
		MsgType:  msg.MsgType,
		Data:     msg.Data,
		SenderID: msg.Client.ID,
	}
}

type Client struct {
	ID    string
	mu    *sync.RWMutex
	conn  *websocket.Conn
	msgCH chan *RespMsg
	done chan struct{}
}

type Server struct {
	clients       map[string]*Client
	mu            *sync.RWMutex
	joinServerCH  chan *Client
	leaveServerCH chan *Client
	broadcastCH   chan *ReqMsg
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		ID:    uuid.NewString(),
		mu:    new(sync.RWMutex),
		conn:  conn,
		msgCH: make(chan *RespMsg,64),
		done: make(chan struct{}),
	}
}

func (c *Client) writeMsgLoop(once *sync.Once){
	defer once.Do(func ()  {
		close(c.done)
	})
	for{
		select{
		case <-c.done:
			return
		case msg:=<-c.msgCH:
			err:=c.conn.WriteJSON(msg)
			if err!=nil{
				fmt.Printf("Error sending msg to clientId= %s\n",err);
				return
			}
		}
	}
}

func (c *Client) readMsgLoop(srv *Server, once *sync.Once) {
	defer func() {
		once.Do(func() {
			close(c.done)
		})
		c.conn.Close()
		srv.leaveServerCH <- c
	}()
	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		msg := new(ReqMsg)
		err = json.Unmarshal(b, msg)
		if err != nil {
			fmt.Printf("unable to unmarshal the msg %v", err)
			continue
		}
		msg.Client = c
		srv.broadcastCH <- msg
	}
}

func NewServer() *Server {
	return &Server{
		clients:       map[string]*Client{},
		mu:            new(sync.RWMutex),
		joinServerCH:  make(chan *Client, 64),
		leaveServerCH: make(chan *Client, 64),
		broadcastCH:   make(chan *ReqMsg, 64),
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

	var once sync.Once;
	go client.writeMsgLoop(&once)
	go client.readMsgLoop(s,&once)

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
		case msg := <-s.broadcastCH:
			go s.brodcast(msg)
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

func (s *Server) brodcast(msg *ReqMsg) {
	cls := []*Client{}

	s.mu.RLock()
	for _, c := range s.clients {
		if c.ID != msg.Client.ID {
			cls = append(cls, c)
		}
	}
	s.mu.RUnlock()
	resp := NewRespMsg(msg)
	for _, c := range cls {
		// err := c.conn.WriteJSON(resp)
		// if err != nil {
		// 	fmt.Printf("Error sending msg to ClientID:%v\n", c.ID)
		// 	continue
		// }
		c.msgCH<-resp
	}
	fmt.Println("Broadcast was sent")
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
