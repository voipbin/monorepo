package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-hook-manager/internal/config"
)

const testRabbitAddress = "amqp://rabbit-send-test"

// prepareSend makes the run independent of test order: viper is reset, the config singleton is
// loaded (the first load in the process wins) and then pointed to a fixed address that is restored
// afterwards, and a fresh Prometheus registry is installed because every request handler creation
// registers collectors on the default registerer.
func prepareSend(t *testing.T) {
	t.Helper()

	resetViper(t)
	config.LoadGlobalConfig()

	cfg := config.Get()
	orig := *cfg
	t.Cleanup(func() { *cfg = orig })
	cfg.RabbitMQAddress = testRabbitAddress

	prevReg := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = prevReg })
}

// installSock replaces the rabbit factory with a mock whose Connect is expected once. The returned
// slice records the addresses the factory was asked to connect to.
func installSock(t *testing.T, mc *gomock.Controller) (*sockhandler.MockSockHandler, *[]string) {
	t.Helper()

	mockSock := sockhandler.NewMockSockHandler(mc)
	uris := []string{}
	restore := sockhandler.SetRabbitFactory(func(serverURI string) sockhandler.SockHandler {
		uris = append(uris, serverURI)
		return mockSock
	})
	t.Cleanup(restore)

	return mockSock, &uris
}

// runCommand executes the root command with the args. Cobra usage and error text go to the
// returned buffer, the printed JSON is captured from stdout.
func runCommand(t *testing.T, args ...string) (stdout string, cobraOut string, err error) {
	t.Helper()

	orig := os.Stdout
	r, w, errPipe := os.Pipe()
	if errPipe != nil {
		t.Fatalf("could not create a pipe: %v", errPipe)
	}
	outCh := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		outCh <- string(out)
	}()
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	buf := &bytes.Buffer{}
	root := initCommand()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err = root.Execute()

	os.Stdout = orig
	_ = w.Close()

	return <-outCh, buf.String(), err
}

func Test_runSend(t *testing.T) {
	// {"k":"v"} as base64
	const payloadBase64 = "eyJrIjoidiJ9"

	tests := []struct {
		name string

		command string
		viaFile bool

		expectQueue string
		expectData  string
		expectType  string
	}{
		{
			name:        "email",
			command:     "send-email",
			expectQueue: "bin-manager.email-manager.request",
			expectData:  `{"received_uri":"hook.example.test","received_data":"` + payloadBase64 + `","received_method":"","received_signature":""}`,
			expectType:  "email",
		},
		{
			name:        "message",
			command:     "send-message",
			viaFile:     true,
			expectQueue: "bin-manager.message-manager.request",
			expectData:  `{"received_uri":"hook.example.test","received_data":"` + payloadBase64 + `","received_method":"","received_signature":""}`,
			expectType:  "message",
		},
		{
			name:        "conversation",
			command:     "send-conversation",
			expectQueue: "bin-manager.conversation-manager.request",
			expectData:  `{"received_uri":"hook.example.test/","received_data":"` + payloadBase64 + `","received_method":"POST","received_signature":""}`,
			expectType:  "conversation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareSend(t)

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockSock, uris := installSock(t, mc)

			args := []string{tt.command, "--uri", "hook.example.test"}
			if tt.viaFile {
				filePath := filepath.Join(t.TempDir(), "payload.json")
				if err := os.WriteFile(filePath, []byte(`{"k":"v"}`), 0o600); err != nil {
					t.Fatalf("could not prepare the payload file: %v", err)
				}
				args = append(args, "--file", filePath)
			} else {
				args = append(args, "--data", `{"k":"v"}`)
			}

			mockSock.EXPECT().Connect().Times(1)
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, &sock.Request{
				URI:       "/v1/hooks",
				Method:    sock.RequestMethodPost,
				Publisher: "hook-manager",
				DataType:  "application/json",
				Data:      json.RawMessage(tt.expectData),
			}).Return(&sock.Response{StatusCode: 200}, nil).Times(1)

			stdout, _, err := runCommand(t, args...)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if len(*uris) != 1 || (*uris)[0] != testRabbitAddress {
				t.Errorf("Wrong match. expect the factory to receive %s once, got: %v", testRabbitAddress, *uris)
			}

			expectOut := fmt.Sprintf("{\n  \"payload\": \"{\\\"k\\\":\\\"v\\\"}\",\n  \"status\": \"sent\",\n  \"type\": \"%s\",\n  \"uri\": \"hook.example.test\"\n}\n", tt.expectType)
			if stdout != expectOut {
				t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expectOut, stdout)
			}
		})
	}
}

func Test_runSend_publishFailure(t *testing.T) {
	tests := []struct {
		name string

		command string
		queue   string

		response    *sock.Response
		publishErr  error
		expectError string
	}{
		{
			name:        "email, publish error",
			command:     "send-email",
			queue:       "bin-manager.email-manager.request",
			publishErr:  fmt.Errorf("broker unavailable"),
			expectError: "failed to send email webhook: ",
		},
		{
			name:        "email, error status",
			command:     "send-email",
			queue:       "bin-manager.email-manager.request",
			response:    &sock.Response{StatusCode: 500},
			expectError: "failed to send email webhook: ",
		},
		{
			name:        "message, publish error",
			command:     "send-message",
			queue:       "bin-manager.message-manager.request",
			publishErr:  fmt.Errorf("broker unavailable"),
			expectError: "failed to send message webhook: ",
		},
		{
			name:        "message, error status",
			command:     "send-message",
			queue:       "bin-manager.message-manager.request",
			response:    &sock.Response{StatusCode: 404},
			expectError: "failed to send message webhook: ",
		},
		{
			name:        "conversation, publish error",
			command:     "send-conversation",
			queue:       "bin-manager.conversation-manager.request",
			publishErr:  fmt.Errorf("broker unavailable"),
			expectError: "failed to send conversation webhook: ",
		},
		{
			name:        "conversation, error status",
			command:     "send-conversation",
			queue:       "bin-manager.conversation-manager.request",
			response:    &sock.Response{StatusCode: 400},
			expectError: "failed to send conversation webhook: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareSend(t)

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockSock, _ := installSock(t, mc)

			mockSock.EXPECT().Connect().Times(1)
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.queue, gomock.Any()).Return(tt.response, tt.publishErr).Times(1)

			stdout, _, err := runCommand(t, tt.command, "--uri", "hook.example.test", "--data", `{"k":"v"}`)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.HasPrefix(err.Error(), tt.expectError) {
				t.Errorf("Wrong match. expect prefix: %q, got: %v", tt.expectError, err)
			}
			if stdout != "" {
				t.Errorf("nothing must be printed on failure. got: %q", stdout)
			}
		})
	}
}

func Test_initHandler(t *testing.T) {
	prepareSend(t)

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock, uris := installSock(t, mc)

	mockSock.EXPECT().Connect().Times(1)

	handler, err := initHandler()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if handler == nil {
		t.Fatalf("Wrong match. expect: a handler, got: nil")
	}
	if len(*uris) != 1 || (*uris)[0] != testRabbitAddress {
		t.Errorf("Wrong match. expect the factory to receive %s once, got: %v", testRabbitAddress, *uris)
	}
}
