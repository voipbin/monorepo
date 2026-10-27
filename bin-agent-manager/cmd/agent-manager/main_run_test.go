package main

import (
	"bufio"
	"bytes"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/internal/config"
	"monorepo/bin-agent-manager/pkg/agenthandler"
)

// unparsableDSN is rejected by the database driver while it is parsed, so a
// connection attempt fails at once without any network access.
const unparsableDSN = "not-a-dsn"

// waitFor is the upper bound for events that happen on other goroutines.
const waitFor = 5 * time.Second

// setGlobalConfig overwrites the process-wide config for one test and restores
// the previous value when the test ends.
func setGlobalConfig(t *testing.T, mutate func(c *config.Config)) {
	t.Helper()

	orig := *config.Get()
	mutate(config.Get())
	t.Cleanup(func() { *config.Get() = orig })
}

// restoreLogrus restores the global logrus formatter and level at the end of the test.
func restoreLogrus(t *testing.T) {
	t.Helper()

	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})
}

// isolateMux gives the test its own default http mux, so the endpoints that
// initProm registers do not collide with other tests.
func isolateMux(t *testing.T) {
	t.Helper()

	orig := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = orig })
}

// respStub is a minimal in-process RESP server on a loopback port chosen by the
// operating system. It answers PING and records the commands it receives.
type respStub struct {
	ln net.Listener

	mu       sync.Mutex
	commands []string

	// failWith, when set, is sent as an error reply to every command.
	failWith string
}

func newRespStub(t *testing.T, failWith string) *respStub {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen on loopback: %v", err)
	}

	s := &respStub{ln: ln, failWith: failWith}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })

	return s
}

func (s *respStub) addr() string { return s.ln.Addr().String() }

func (s *respStub) received() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string{}, s.commands...)
}

func (s *respStub) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *respStub) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	r := bufio.NewReader(conn)
	for {
		args, err := readRespCommand(r)
		if err != nil {
			return
		}

		s.mu.Lock()
		s.commands = append(s.commands, strings.ToUpper(args[0]))
		s.mu.Unlock()

		switch {
		case s.failWith != "":
			_, _ = fmt.Fprintf(conn, "-ERR %s\r\n", s.failWith)
		case strings.EqualFold(args[0], "PING"):
			_, _ = conn.Write([]byte("+PONG\r\n"))
		default:
			_, _ = fmt.Fprintf(conn, "-ERR unknown command %q\r\n", args[0])
		}
	}
}

func readRespCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected line: %q", line)
	}
	n, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimSpace(header[1:]))
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}

	return args, nil
}

// captureStdout runs fn and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = orig
	out := <-done
	_ = r.Close()

	return out
}

func Test_runDaemon_databaseFailure(t *testing.T) {
	isolateMux(t)
	t.Cleanup(func() { signal.Stop(chSigs) })
	setGlobalConfig(t, func(c *config.Config) {
		c.DatabaseDSN = unparsableDSN
		// the metrics server must fail to start at once: a unique endpoint and an
		// address that cannot be listened on.
		c.PrometheusEndpoint = "/metrics-rundaemon-test"
		c.PrometheusListenAddress = "bad::addr::rundaemon"
	})

	err := runDaemon()
	if err == nil {
		t.Fatalf("Wrong match. expect an error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "could not connect to the database: database open error: invalid DSN") {
		t.Errorf("Wrong match. got: %q", err.Error())
	}

	// the metrics endpoint is registered before the database is opened.
	req, errReq := http.NewRequest(http.MethodGet, "/metrics-rundaemon-test", nil)
	if errReq != nil {
		t.Fatalf("could not create the request: %v", errReq)
	}
	if _, pattern := http.DefaultServeMux.Handler(req); pattern != "/metrics-rundaemon-test" {
		t.Errorf("Wrong match. expect the metrics handler to be registered, got pattern: %q", pattern)
	}
}

func Test_initCache(t *testing.T) {
	t.Run("a pong reply returns the connected handler", func(t *testing.T) {
		stub := newRespStub(t, "")
		setGlobalConfig(t, func(c *config.Config) {
			c.RedisAddress = stub.addr()
			c.RedisDatabase = 0
		})

		res, err := initCache()
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if res == nil {
			t.Fatalf("Wrong match. expect a handler, got nil")
		}
		if got := stub.received(); len(got) != 1 || got[0] != "PING" {
			t.Errorf("Wrong match. expect exactly one PING, got: %v", got)
		}
	})

	t.Run("an error reply is a connect failure", func(t *testing.T) {
		stub := newRespStub(t, "down for test")
		setGlobalConfig(t, func(c *config.Config) { c.RedisAddress = stub.addr() })

		res, err := initCache()
		if err == nil {
			t.Fatalf("Wrong match. expect an error, got a handler: %v", res)
		}
		if res != nil {
			t.Errorf("Wrong match. expect no handler, got: %v", res)
		}
		if !strings.HasPrefix(err.Error(), "cache connect error: ") || !strings.Contains(err.Error(), "down for test") {
			t.Errorf("Wrong match. got: %q", err.Error())
		}
	})
}

// Test_initSignal injects the signal into the channel that initSignal reads, so
// no real signal is sent. Each round calls initSignal again, because its
// goroutine ends after one signal.
func Test_initSignal(t *testing.T) {
	t.Cleanup(func() { signal.Stop(chSigs) })

	signals := []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP}

	for _, sig := range signals {
		t.Run(sig.String(), func(t *testing.T) {
			select {
			case <-chDone:
			default:
			}

			initSignal()

			select {
			case chSigs <- sig:
			case <-time.After(waitFor):
				t.Fatalf("Wrong match. the signal channel did not accept the signal")
			}

			select {
			case got := <-chDone:
				if !got {
					t.Errorf("Wrong match. expect true on the done channel")
				}
			case <-time.After(waitFor):
				t.Errorf("Wrong match. expect the done channel to receive after the signal")
			}
		})
	}
}

func Test_initProm(t *testing.T) {
	isolateMux(t)

	// the error log of the failed server is the only way to see that its goroutine ended.
	origHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	hook := logrustest.NewGlobal()
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(origHooks) })

	const endpoint = "/metrics-initprom-test"
	const listen = "bad::addr::initprom"

	initProm(endpoint, listen)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("could not create the request: %v", err)
	}
	if _, pattern := http.DefaultServeMux.Handler(req); pattern != endpoint {
		t.Errorf("Wrong match. expect the metrics handler on %q, got pattern: %q", endpoint, pattern)
	}

	deadline := time.Now().Add(waitFor)
	for {
		found := false
		for _, e := range hook.AllEntries() {
			if e.Level == logrus.ErrorLevel && strings.Contains(e.Message, "Prometheus server error") && strings.Contains(e.Message, listen) {
				found = true
				break
			}
		}
		if found {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Wrong match. expect the listen error to be logged, got: %v", hook.AllEntries())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func Test_runServiceListen(t *testing.T) {
	queue := string(commonoutline.QueueNameAgentRequest)
	errBoom := stderrors.New("boom")

	t.Run("a queue creation failure is returned and nothing else is called", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockAgent := agenthandler.NewMockAgentHandler(mc)

		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errBoom)

		err := runServiceListen(mockSock, mockAgent)
		if !stderrors.Is(err, errBoom) {
			t.Errorf("Wrong match. expect the wrapped boom, got: %v", err)
		}
	})

	t.Run("success starts the request consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockAgent := agenthandler.NewMockAgentHandler(mc)
		consumed := make(chan struct{})

		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, string(commonoutline.ServiceNameAgentManager), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
				close(consumed)
				return nil
			}).Times(1)

		if err := runServiceListen(mockSock, mockAgent); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}

		select {
		case <-consumed:
		case <-time.After(waitFor):
			t.Errorf("Wrong match. ConsumeRPC was not called")
		}
	})
}

func Test_runServiceSubscribe(t *testing.T) {
	queue := string(commonoutline.QueueNameAgentSubscribe)
	webhookTopic := string(commonoutline.QueueNameWebhookEventTopic)
	webhookFanout := string(commonoutline.QueueNameWebhookEvent)
	eventExchange := string(commonoutline.QueueNameEvent)
	errBoom := stderrors.New("boom")

	tests := []struct {
		name string

		// expect registers the broker calls of the test; the consume call closes
		// the channel it is given.
		expect func(mockSock *sockhandler.MockSockHandler, consumed chan struct{})

		expectErr     string
		expectWrap    error
		expectConsume bool
	}{
		{
			name: "queue creation failure",
			expect: func(m *sockhandler.MockSockHandler, _ chan struct{}) {
				m.EXPECT().QueueCreate(queue, "normal").Return(errBoom)
			},
			expectErr:  "could not declare the queue for listenHandler",
			expectWrap: errBoom,
		},
		{
			name: "topic exchange declaration failure",
			expect: func(m *sockhandler.MockSockHandler, _ chan struct{}) {
				m.EXPECT().QueueCreate(queue, "normal").Return(nil)
				m.EXPECT().QueueBind(queue, "#", webhookTopic, false, nil).Return(nil)
				m.EXPECT().QueueUnbind(queue, "", webhookFanout, nil).Return(nil)
				m.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(errBoom)
			},
			expectErr:  "could not declare the global topic exchange",
			expectWrap: errBoom,
		},
		{
			name: "topic pattern binding failure",
			expect: func(m *sockhandler.MockSockHandler, _ chan struct{}) {
				m.EXPECT().QueueCreate(queue, "normal").Return(nil)
				m.EXPECT().QueueBind(queue, "#", webhookTopic, false, nil).Return(nil)
				m.EXPECT().QueueUnbind(queue, "", webhookFanout, nil).Return(nil)
				m.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(nil)
				m.EXPECT().QueueBind(queue, gomock.Any(), eventExchange, false, nil).Return(errBoom).Times(1)
			},
			expectErr:  "could not bind the topic pattern",
			expectWrap: errBoom,
		},
		{
			name: "success starts the event consumer",
			expect: func(m *sockhandler.MockSockHandler, consumed chan struct{}) {
				m.EXPECT().QueueCreate(queue, "normal").Return(nil)
				m.EXPECT().QueueBind(queue, "#", webhookTopic, false, nil).Return(nil)
				m.EXPECT().QueueUnbind(queue, "", webhookFanout, nil).Return(nil)
				m.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(nil)
				m.EXPECT().QueueBind(queue, gomock.Any(), eventExchange, false, nil).Return(nil).Times(3)
				m.EXPECT().ConsumeMessage(gomock.Any(), queue, string(commonoutline.ServiceNameAgentManager), false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
						close(consumed)
						return nil
					}).Times(1)
			},
			expectConsume: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)
			consumed := make(chan struct{})
			tt.expect(mockSock, consumed)

			err := runServiceSubscribe(mockSock, mockAgent)
			if tt.expectWrap == nil {
				if err != nil {
					t.Fatalf("Wrong match. expect ok, got: %v", err)
				}
			} else {
				if !stderrors.Is(err, tt.expectWrap) {
					t.Fatalf("Wrong match. expect the wrapped boom, got: %v", err)
				}
				if !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect %q in the error, got: %q", tt.expectErr, err.Error())
				}
			}

			if tt.expectConsume {
				select {
				case <-consumed:
				case <-time.After(waitFor):
					t.Errorf("Wrong match. ConsumeMessage was not called")
				}
			}
		})
	}
}

func Test_main_help(t *testing.T) {
	restoreLogrus(t)
	viper.Reset()
	t.Cleanup(viper.Reset)

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"agent-manager", "--help"}

	out := captureStdout(t, main)
	if !strings.Contains(out, "Voipbin Agent Manager Daemon") {
		t.Errorf("Wrong match. expect the help text, got: %q", out)
	}
}
