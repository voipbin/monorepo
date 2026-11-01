package listenhandler

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/ivahaev/amigo"

	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// contractSock is a SockHandler stand-in that delivers every published request
// to the proxy listenHandler of the running test, the way the bus would.
// Only RequestPublish is implemented.
type contractSock struct {
	sockhandler.SockHandler

	mu     sync.Mutex
	target *listenHandler
}

func (s *contractSock) setTarget(h *listenHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.target = h
}

func (s *contractSock) RequestPublish(_ context.Context, _ string, req *sock.Request) (*sock.Response, error) {
	s.mu.Lock()
	h := s.target
	s.mu.Unlock()

	return h.processRequest(req)
}

var (
	contractSockHandler = &contractSock{}

	// NewRequestHandler registers Prometheus metrics, so it can run only once per process.
	contractRequestHandler = sync.OnceValue(func() requesthandler.RequestHandler {
		return requesthandler.NewRequestHandler(contractSockHandler, outline.ServiceNameCallManager)
	})
)

// Test_AstAMIRedirect_reaches_proxy sends the real AstAMIRedirect request through the
// real proxy routing and a fake AMI server, so the sender URI and the proxy route
// cannot drift apart unnoticed.
func Test_AstAMIRedirect_reaches_proxy(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectErr       bool
		expectErrSubstr string
	}{
		{
			name:       "success",
			asteriskID: "00:11:22:33:44:01",
			channelID:  "6b79ae28-e3f1-11ea-bf62-7f539c6300fc",
		},
		{
			name:            "ami error response",
			asteriskID:      "00:11:22:33:44:02",
			channelID:       fakeAMIMissingChannel,
			expectErr:       true,
			expectErrSubstr: "response: Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				amiSock.Close()
			})
			waitConnected(t, amiSock.Connected)
			fake.waitAction(t) // login

			contractSockHandler.setTarget(&listenHandler{amiSock: amiSock})

			err := contractRequestHandler().AstAMIRedirect(context.Background(), tt.asteriskID, tt.channelID, "svc-stasis", "s", "1")
			if tt.expectErr {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrSubstr) {
					t.Fatalf("Wrong match. expect: %s, got: %v", tt.expectErrSubstr, err)
				}
			} else if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			sent := fake.waitAction(t)
			if sent["Action"] != "Redirect" || sent["Channel"] != tt.channelID || sent["Context"] != "svc-stasis" || sent["Exten"] != "s" || sent["Priority"] != "1" {
				t.Errorf("Wrong action frame. got: %v", sent)
			}
		})
	}
}
