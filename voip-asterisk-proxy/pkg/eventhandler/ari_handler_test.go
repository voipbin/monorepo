package eventhandler

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
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
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func Test_eventARIRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// create mock asterisk
	addr := "127.0.0.1:8080"
	mockAsterisk(addr)

	eventHandler := eventHandler{
		// rabbitSock: mockRabbit,
		ariAddr: "127.0.0.1:8080",
	}

	if err := eventHandler.eventARIConnect(); err != nil {
		t.Errorf("Expected ok, but got an error. err: %v", err)
	}
}
