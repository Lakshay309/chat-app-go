package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type TestConfig struct {
	clientCount    int
	wg             *sync.WaitGroup
	brMsgCount     *atomic.Int64
	targetMsgCount int
}

var (
	host = "ws://localhost"
)

func DailServer(tc *TestConfig) *websocket.Conn {
	exit := make(chan struct{})
	var once sync.Once
	// time.Sleep(1 * time.Second)
	dailer := websocket.DefaultDialer

	conn, _, err := dailer.Dial(fmt.Sprintf("%s%s", host, wsPort), nil)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			time.Sleep(2 * time.Second)
			if tc.targetMsgCount == int(tc.brMsgCount.Load()) {
				once.Do(func() {
					close(exit)
				})
				return
			}
		}
	}()

	go func() {
		<-exit
		conn.Close()
		tc.wg.Done()
	}()
	fmt.Println("connected to server:", conn.LocalAddr().String())
	time.Sleep(1 * time.Second)

	go func() {
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				once.Do(func() {
					close(exit)
				})
				return
			}
			if len(b) > 0 {
				tc.brMsgCount.Add(1)
			}

		}
	}()
	return conn
}

type TestClient struct {
	conn  *websocket.Conn
	msgCH chan *ReqMsg
	// done  chan struct{}
	ctx   context.Context
}

func NewTestClient(conn *websocket.Conn, ctx context.Context) *TestClient {
	return &TestClient{
		conn:  conn,
		msgCH: make(chan *ReqMsg, 64),
		ctx:   ctx,
	}
}

func (c *TestClient) writeLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.msgCH:
			err := c.conn.WriteJSON(msg)
			if err != nil {
				fmt.Printf("Error sending msg %v\n", err)
				return
			}
		}
	}
}

func TestConnection(t *testing.T) {
	go createServer()
	ctx, cancel := context.WithCancel(context.Background())

	time.Sleep(1 * time.Second)
	clientCount := 5
	brCount := 3

	tc := TestConfig{
		clientCount:    clientCount,
		wg:             new(sync.WaitGroup),
		brMsgCount:     new(atomic.Int64),
		targetMsgCount: clientCount * brCount,
	}
	tc.wg.Add(tc.clientCount + 1)

	brConn := DailServer(&tc)
	brClient := NewTestClient(brConn, ctx)
	go brClient.writeLoop()

	for range tc.clientCount {
		go DailServer(&tc)
	}
	time.Sleep(800 * time.Millisecond)
	for range brCount {
		msg := ReqMsg{
			MsgType: MsgType_Brodcast,
			Data:    "Hello from test",
		}
		// time.Sleep(100 * time.Millisecond)
		// go func() {
		// err := brClient.WriteJSON(&msg)
		// if err != nil {
		// 	fmt.Printf("Error sending msg %v\n", err)
		// 	return
		// }
		// }()
		brClient.msgCH <- &msg
	}

	tc.wg.Wait()
	cancel()
	time.Sleep(3 * time.Second)
	fmt.Println("Exiting the test")

}
