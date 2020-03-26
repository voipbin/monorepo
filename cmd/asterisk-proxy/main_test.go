package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

func mockAsterisk(addr string) {
	http.HandleFunc("/ari/events", echo)

	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Printf("Could not listen the addr. err: %v\n", err)
		}
	}()

	// sleep for 100ms for initiating the http
	time.Sleep(time.Microsecond * 100)
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func TestConnectARI(t *testing.T) {

	// create mock asterisk
	addr := "127.0.0.1:8080"
	mockAsterisk(addr)

	_, err := connectARI(addr, "asterisk:asterisk", "True", "voipbin")
	if err != nil {
		t.Errorf("Expected ok, but got an error. err: %v", err)
	}
}
