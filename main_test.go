package main

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type TestConfig struct {
	clientCount int
	wg          *sync.WaitGroup
}

var (
	host = "ws://localhost"
)

func DailServer(wg *sync.WaitGroup) {
	time.Sleep(1 * time.Second)
	dailer := websocket.DefaultDialer
	conn, _, err := dailer.Dial(fmt.Sprintf("%s%s", host, wsPort), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		conn.Close()
		wg.Done()
	}()
	fmt.Println("connected to server:", conn.LocalAddr().String())
	time.Sleep(1 * time.Second)
}

func TestConnection(t *testing.T) {
	go createServer()
	time.Sleep(5 * time.Second)
	tc := TestConfig{clientCount: 7, wg: new(sync.WaitGroup)}
	tc.wg.Add(tc.clientCount)
	for range tc.clientCount {
		go DailServer(tc.wg)

	}
	tc.wg.Wait()
	fmt.Println("Exiting the test")
}
