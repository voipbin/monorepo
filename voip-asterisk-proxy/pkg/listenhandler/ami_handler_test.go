package listenhandler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ivahaev/amigo"

	"monorepo/bin-common-handler/models/sock"
)

func Test_sendRequestToAsteriskAMI_invalid_json(t *testing.T) {
	h := &listenHandler{
		amiSock: amigo.New(&amigo.Settings{}),
	}

	status, res, err := h.sendRequestToAsteriskAMI(&sock.Request{
		URI:    "/ami/action",
		Method: sock.RequestMethodPost,
		Data:   []byte("not-json"),
	})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if status != 0 || res != nil {
		t.Errorf("Wrong match. expect: 0 and nil, got: %d, %v", status, res)
	}
}

func Test_listenHandlerAMI_not_connected(t *testing.T) {
	// the client is never connected: Action reports a not-connected error and
	// the handler must return it instead of building a response.
	h := &listenHandler{
		amiSock: amigo.New(&amigo.Settings{}),
	}

	res, err := h.listenHandlerAMI(&sock.Request{
		URI:    "/ami/action",
		Method: sock.RequestMethodPost,
		Data:   []byte(`{"Action":"Ping"}`),
	})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if !strings.Contains(err.Error(), "Not connected") {
		t.Errorf("Wrong match. expect: not connected error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
}

func Test_listenHandlerAMI_connected(t *testing.T) {
	fake := newFakeAMI(t)
	host, port := fake.hostPort(t)

	settings := &amigo.Settings{
		Host:     host,
		Port:     port,
		Username: "fake-user",
	}
	settings.Password = "fake-secret"
	amiSock := amigo.New(settings)
	amiSock.Connect()
	t.Cleanup(func() {
		// registered after the fake server cleanup, so it runs first. the
		// client is stopped only after it has connected.
		amiSock.Close()
	})
	waitConnected(t, amiSock.Connected)

	login := fake.waitAction(t)
	if login["Action"] != "Login" || login["Username"] != "fake-user" || login["Secret"] != "fake-secret" {
		t.Errorf("Wrong login frame. got: Action=%s Username=%s", login["Action"], login["Username"])
	}

	h := &listenHandler{amiSock: amiSock}

	// requests go through processRequest, the same way a bus message does.
	res, err := h.processRequest(&sock.Request{
		URI:      "/ami/action",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"Action":"Ping","ActionID":"test-action-1"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 200 || res.DataType != "application/json" {
		t.Errorf("Wrong match. expect: 200 json, got: %d %s", res.StatusCode, res.DataType)
	}

	var got map[string]string
	if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil {
		t.Fatalf("Could not unmarshal the response. err: %v", errUnmarshal)
	}
	if got["Response"] != "Success" || got["ActionID"] != "test-action-1" || got["Message"] != "handled Ping" {
		t.Errorf("Wrong response. Response=%s ActionID=%s Message=%s", got["Response"], got["ActionID"], got["Message"])
	}

	// the action reached the server unchanged.
	sent := fake.waitAction(t)
	if sent["Action"] != "Ping" || sent["ActionID"] != "test-action-1" {
		t.Errorf("Wrong action frame. got: Action=%s ActionID=%s", sent["Action"], sent["ActionID"])
	}
}
