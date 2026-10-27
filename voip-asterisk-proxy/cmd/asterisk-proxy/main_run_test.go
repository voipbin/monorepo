package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
)

const runTestTimeout = 5 * time.Second

// exitPanic is what the replaced logrus exit function panics with, so a Fatalf
// in main() ends the run without ending the test process.
type exitPanic struct{ code int }

// runEndpoints are the loopback AMI and ARI servers the main() tests share.
// main() never closes its AMI client and the ARI loop cannot be stopped, so
// both servers live until the test binary exits: closing them would make those
// clients log reconnect errors for the rest of the run.
var (
	runEndpointsOnce sync.Once
	runAMI           *fakeAMI
	runARIAddr       string
	runARIAccepted   = make(chan string, 64)
)

func startRunEndpoints(t *testing.T) (amiHostPort [2]string, ariAddr string) {
	t.Helper()

	runEndpointsOnce.Do(func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("could not listen. err: %v", err)
		}
		runAMI = &fakeAMI{ln: ln, actions: make(chan map[string]string, 32)}
		go runAMI.serve()

		upgrader := websocket.Upgrader{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, errUp := upgrader.Upgrade(w, r, nil)
			if errUp != nil {
				return
			}

			select {
			case runARIAccepted <- r.URL.Query().Get("app"):
			default:
			}

			// hold the connection and send nothing.
			for {
				if _, _, errRead := c.ReadMessage(); errRead != nil {
					return
				}
			}
		}))
		runARIAddr = srv.Listener.Addr().String()
	})

	host, port := runAMI.hostPort(t)
	return [2]string{host, port}, runARIAddr
}

// pickRunInterface returns the first interface with a hardware address and a
// non-loopback IPv4 address, the same kind main() needs for the asterisk id.
func pickRunInterface(t *testing.T) (name, mac, ip string) {
	t.Helper()

	ifaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("interfaces are not available: %v", err)
	}

	for _, iface := range ifaces {
		if len(iface.HardwareAddr) == 0 {
			continue
		}

		addrs, errAddrs := iface.Addrs()
		if errAddrs != nil {
			continue
		}

		for _, addr := range addrs {
			n, ok := addr.(*net.IPNet)
			if !ok || n.IP.IsLoopback() || n.IP.To4() == nil {
				continue
			}

			return iface.Name, iface.HardwareAddr.String(), n.IP.To4().String()
		}
	}

	t.Skip("no interface with a hardware address and a non-loopback IPv4 address on this host")
	return "", "", ""
}

// mainRunner runs main() against the mocks. It does not use t.Parallel: the
// sock factory, the metrics registry, the exit function and the package
// variables are process wide.
type mainRunner struct {
	mock  *sockhandler.MockSockHandler
	redis *miniredis.Miniredis
	app   string
}

func newMainRunner(t *testing.T) *mainRunner {
	t.Helper()

	amiHostPort, ariAddr := startRunEndpoints(t)

	mc := gomock.NewController(t)
	mock := sockhandler.NewMockSockHandler(mc)
	mr := miniredis.RunT(t)

	// the request handler registers its metrics with MustRegister, which panics on a second run.
	oldRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = oldRegisterer })

	t.Cleanup(sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mock }))

	// Fatalf returns to its caller once the exit function returns, so it panics instead.
	oldExit := logrus.StandardLogger().ExitFunc
	logrus.StandardLogger().ExitFunc = func(code int) { panic(exitPanic{code: code}) }
	t.Cleanup(func() { logrus.StandardLogger().ExitFunc = oldExit })

	type settings struct {
		ariAddress, ariAccount, ariSubscribeAll, ariApplication       string
		amiHost, amiPort, amiUsername, amiPassword, amiEventFilter    string
		interfaceName, rabbitMQAddress, rabbitMQQueueListen           string
		redisAddress, redisPassword, googleApplicationCredentialsJSON string
		redisDatabase                                                 int
		kubernetesDisabled                                            bool
	}
	saved := settings{
		ariAddress: ariAddress, ariAccount: ariAccount, ariSubscribeAll: ariSubscribeAll, ariApplication: ariApplication,
		amiHost: amiHost, amiPort: amiPort, amiUsername: amiUsername, amiPassword: amiPassword, amiEventFilter: amiEventFilter,
		interfaceName: interfaceName, rabbitMQAddress: rabbitMQAddress, rabbitMQQueueListen: rabbitMQQueueListen,
		redisAddress: redisAddress, redisPassword: redisPassword, googleApplicationCredentialsJSON: googleApplicationCredentialsJSON,
		redisDatabase: redisDatabase, kubernetesDisabled: kubernetesDisabled,
	}

	drainSigs := func() {
		for len(chSigs) > 0 {
			<-chSigs
		}
	}
	drainSigs()
	t.Cleanup(func() {
		drainSigs()

		ariAddress, ariAccount, ariSubscribeAll, ariApplication = saved.ariAddress, saved.ariAccount, saved.ariSubscribeAll, saved.ariApplication
		amiHost, amiPort, amiUsername, amiPassword, amiEventFilter = saved.amiHost, saved.amiPort, saved.amiUsername, saved.amiPassword, saved.amiEventFilter
		interfaceName, rabbitMQAddress, rabbitMQQueueListen = saved.interfaceName, saved.rabbitMQAddress, saved.rabbitMQQueueListen
		redisAddress, redisPassword, redisDatabase = saved.redisAddress, saved.redisPassword, saved.redisDatabase
		googleApplicationCredentialsJSON, kubernetesDisabled = saved.googleApplicationCredentialsJSON, saved.kubernetesDisabled
	})

	// each run has its own application name, so an ARI connection from another run cannot satisfy a wait.
	app := fmt.Sprintf("app-%s-%d", t.Name(), time.Now().UnixNano())

	ariAddress, ariAccount, ariSubscribeAll, ariApplication = ariAddr, "fake-ari-account", "true", app
	amiHost, amiPort, amiUsername = amiHostPort[0], amiHostPort[1], "fake-user"
	amiPassword = "fake-secret"
	amiEventFilter = ""
	rabbitMQAddress = "amqp://fake-broker"
	rabbitMQQueueListen = "asterisk.test.request"
	redisAddress = mr.Addr()
	redisDatabase = 0
	redisPassword = ""
	googleApplicationCredentialsJSON = "{malformed"
	kubernetesDisabled = true

	return &mainRunner{mock: mock, redis: mr, app: app}
}

// run executes main() and returns the exit code of a Fatalf, or -1 when main() returned.
// It fails the test if main() does not finish within runTestTimeout.
func (r *mainRunner) run(t *testing.T) int {
	t.Helper()

	result := make(chan int, 1)
	go func() {
		code := -1
		defer func() {
			if rec := recover(); rec != nil {
				exit, ok := rec.(exitPanic)
				if !ok {
					panic(rec)
				}
				code = exit.code
			}
			result <- code
		}()

		main()
	}()

	select {
	case code := <-result:
		return code
	case <-time.After(runTestTimeout):
		t.Fatalf("main() did not finish in time")
		return 0
	}
}

func waitGroupDone(t *testing.T, wg *sync.WaitGroup, what string) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(runTestTimeout):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func Test_main_runsUntilSignal(t *testing.T) {
	name, mac, ip := pickRunInterface(t)
	r := newMainRunner(t)
	interfaceName = name

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	var consumers sync.WaitGroup
	consumers.Add(2)

	r.mock.EXPECT().Connect().Times(1)
	r.mock.EXPECT().TopicCreate("asterisk.all.event").Return(nil).Times(1)
	r.mock.EXPECT().QueueCreate("asterisk.test.request", "normal").Return(nil).Times(1)
	r.mock.EXPECT().QueueCreate("asterisk."+mac+".request", "volatile").Return(nil).Times(1)
	r.mock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), "asterisk-proxy", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
			consumers.Done()
			<-release
			return nil
		}).Times(2)

	chSigs <- os.Interrupt
	if code := r.run(t); code != -1 {
		t.Fatalf("Wrong match. expect: main() returns, got: exit code %d", code)
	}
	waitGroupDone(t, &consumers, "both consumers to start")

	// the ARI event loop connects with this run's application name.
	deadline := time.After(runTestTimeout)
	for connected := false; !connected; {
		select {
		case got := <-runARIAccepted:
			connected = got == r.app
		case <-deadline:
			t.Fatalf("the ARI connection of this run was not accepted in time")
		}
	}

	// the address is written from a goroutine.
	key := "asterisk." + mac + ".address-internal"
	for end := time.Now().Add(runTestTimeout); !r.redis.Exists(key); {
		if time.Now().After(end) {
			t.Fatalf("the proxy info was not written to redis in time. key: %s", key)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got, _ := r.redis.Get(key); got != ip {
		t.Errorf("Wrong match. expect: %s, got: %s", ip, got)
	}
}

func Test_main_unknownInterface(t *testing.T) {
	r := newMainRunner(t)
	interfaceName = "no-such-iface0"

	r.mock.EXPECT().Connect().Times(1)

	if code := r.run(t); code != 1 {
		t.Errorf("Wrong match. expect: exit code 1, got: %d", code)
	}
}

func Test_main_kubernetesOutsideCluster(t *testing.T) {
	name, _, _ := pickRunInterface(t)
	r := newMainRunner(t)
	interfaceName = name
	kubernetesDisabled = false
	t.Setenv("KUBERNETES_SERVICE_HOST", "")

	// no handler is created, so the mock sees the connect only.
	r.mock.EXPECT().Connect().Times(1)

	if code := r.run(t); code != -1 {
		t.Errorf("Wrong match. expect: main() returns, got: exit code %d", code)
	}
}

func Test_main_emptyListenQueue(t *testing.T) {
	name, _, _ := pickRunInterface(t)
	r := newMainRunner(t)
	interfaceName = name
	rabbitMQQueueListen = ""

	// no QueueCreate or ConsumeRPC expectation: the queue list is rejected before any queue is declared.
	r.mock.EXPECT().Connect().Times(1)
	r.mock.EXPECT().TopicCreate("asterisk.all.event").Return(nil).Times(1)

	if code := r.run(t); code != 1 {
		t.Errorf("Wrong match. expect: exit code 1, got: %d", code)
	}
}

func Test_main_consumerFailure(t *testing.T) {
	name, _, _ := pickRunInterface(t)
	r := newMainRunner(t)
	interfaceName = name

	var consumers sync.WaitGroup
	consumers.Add(2)

	r.mock.EXPECT().Connect().Times(1)
	r.mock.EXPECT().TopicCreate("asterisk.all.event").Return(nil).Times(1)
	r.mock.EXPECT().QueueCreate(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	r.mock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
			consumers.Done()
			return errors.New("consume failed")
		}).Times(2)

	// the failure reaches the final select instead of the signal channel.
	if code := r.run(t); code != 1 {
		t.Errorf("Wrong match. expect: exit code 1, got: %d", code)
	}

	// finish only after the second consumer goroutine has made its call.
	waitGroupDone(t, &consumers, "both consumers to be called")
}
