package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// initVarsSnapshot holds every package variable that initVariable assigns, one
// per flag.
type initVarsSnapshot struct {
	ariAddress, ariAccount, ariSubscribeAll, ariApplication string

	amiHost, amiPort, amiUsername, amiPassword, amiEventFilter string

	interfaceName string

	rabbitMQAddress, rabbitMQQueueListen string

	redisAddress  string
	redisDatabase int
	redisPassword string

	prometheusEndpoint, prometheusListenAddress string

	recordingBucketName, recordingAsteriskDirectory, recordingBucketDirectory string

	googleApplicationCredentialsJSON string

	kubernetesDisabled bool
}

func snapshotInitVars() initVarsSnapshot {
	return initVarsSnapshot{
		ariAddress: ariAddress, ariAccount: ariAccount, ariSubscribeAll: ariSubscribeAll, ariApplication: ariApplication,

		amiHost: amiHost, amiPort: amiPort, amiUsername: amiUsername, amiPassword: amiPassword, amiEventFilter: amiEventFilter,

		interfaceName: interfaceName,

		rabbitMQAddress: rabbitMQAddress, rabbitMQQueueListen: rabbitMQQueueListen,

		redisAddress: redisAddress, redisDatabase: redisDatabase, redisPassword: redisPassword,

		prometheusEndpoint: prometheusEndpoint, prometheusListenAddress: prometheusListenAddress,

		recordingBucketName: recordingBucketName, recordingAsteriskDirectory: recordingAsteriskDirectory, recordingBucketDirectory: recordingBucketDirectory,

		googleApplicationCredentialsJSON: googleApplicationCredentialsJSON,

		kubernetesDisabled: kubernetesDisabled,
	}
}

func (s initVarsSnapshot) restore() {
	ariAddress, ariAccount, ariSubscribeAll, ariApplication = s.ariAddress, s.ariAccount, s.ariSubscribeAll, s.ariApplication
	amiHost, amiPort, amiUsername, amiPassword, amiEventFilter = s.amiHost, s.amiPort, s.amiUsername, s.amiPassword, s.amiEventFilter
	interfaceName = s.interfaceName
	rabbitMQAddress, rabbitMQQueueListen = s.rabbitMQAddress, s.rabbitMQQueueListen
	redisAddress, redisDatabase, redisPassword = s.redisAddress, s.redisDatabase, s.redisPassword
	prometheusEndpoint, prometheusListenAddress = s.prometheusEndpoint, s.prometheusListenAddress
	recordingBucketName, recordingAsteriskDirectory, recordingBucketDirectory = s.recordingBucketName, s.recordingAsteriskDirectory, s.recordingBucketDirectory
	googleApplicationCredentialsJSON = s.googleApplicationCredentialsJSON
	kubernetesDisabled = s.kubernetesDisabled
}

// isolateInitState saves the process wide state initVariable touches and
// restores it when the test ends. viper has no snapshot, so it is reset; no other
// test in this package uses viper.
func isolateInitState(t *testing.T) {
	t.Helper()

	origCommandLine := pflag.CommandLine
	origArgs := os.Args
	snap := snapshotInitVars()

	t.Cleanup(func() {
		pflag.CommandLine = origCommandLine
		os.Args = origArgs
		snap.restore()
		viper.Reset()
	})

	// only the program name, so the test binary's own flags are not parsed.
	os.Args = []string{"asterisk-proxy"}
}

// runInitVariable runs initVariable on a fresh flag set. When breakFlag is not
// empty, that flag's name stops resolving once parsing is done, so the flag
// lookup that follows returns nil. It returns the flag set and the recovered
// panic value, if any.
func runInitVariable(breakFlag string) (fs *pflag.FlagSet, recovered any) {
	fs = pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		if breakFlag != "" && name == breakFlag && f.Parsed() {
			return pflag.NormalizedName("unresolved_" + name)
		}

		return pflag.NormalizedName(name)
	})
	pflag.CommandLine = fs
	viper.Reset()

	defer func() {
		recovered = recover()
	}()
	initVariable()

	return fs, nil
}

func Test_initVariable_envOverridesDefaults(t *testing.T) {
	isolateInitState(t)
	t.Setenv("ARI_ADDRESS", "203.0.113.7:8088")
	t.Setenv("REDIS_DATABASE", "5")

	_, recovered := runInitVariable("")
	if recovered != nil {
		t.Fatalf("Wrong match. expect: ok, got a panic: %v", recovered)
	}

	if ariAddress != "203.0.113.7:8088" {
		t.Errorf("Wrong ari address. expect: 203.0.113.7:8088, got: %s", ariAddress)
	}
	if redisDatabase != 5 {
		t.Errorf("Wrong redis database. expect: 5, got: %d", redisDatabase)
	}
}

func Test_initVariable_flagBindingError(t *testing.T) {
	isolateInitState(t)

	fs, recovered := runInitVariable("")
	if recovered != nil {
		t.Fatalf("Wrong match. expect: ok, got a panic: %v", recovered)
	}

	names := []string{}
	fs.VisitAll(func(f *pflag.Flag) {
		names = append(names, f.Name)
	})
	sort.Strings(names)

	// one flag per variable in the snapshot; a smaller table would hide
	// flags that are not checked.
	if len(names) != 22 {
		t.Fatalf("Wrong flag count. expect: 22, got: %d (%v)", len(names), names)
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			_, recovered := runInitVariable(name)

			err, ok := recovered.(error)
			if !ok {
				t.Fatalf("Wrong panic value. expect: an error, got: %v", recovered)
			}

			expect := fmt.Sprintf("flag for %q is nil", name)
			if err.Error() != expect {
				t.Errorf("Wrong error. expect: %s, got: %s", expect, err.Error())
			}
		})
	}
}

// promTestSeq makes the endpoint unique per run: the default mux cannot
// unregister a handler, and init() already registered the default endpoint.
var promTestSeq atomic.Int32

func Test_initProm_retriesUntilListenAddressIsFree(t *testing.T) {
	occupier, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	t.Cleanup(func() {
		_ = occupier.Close()
	})
	addr := occupier.Addr().String()
	endpoint := fmt.Sprintf("/test-metrics-%d", promTestSeq.Add(1))

	logs := captureLogs(t)
	initProm(endpoint, addr)

	// the listen fails while the address is taken.
	logs.waitFor(t, "listener failure log", func(e capturedEntry) bool {
		return strings.HasPrefix(e.message, "Could not start prometheus listener")
	})

	// the log carries no address, so it cannot prove this call's goroutine
	// retried. Free the address and expect the endpoint to be served by it.
	_ = occupier.Close()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, errGet := client.Get("http://" + addr + endpoint)
		if errGet == nil {
			status := resp.StatusCode
			_ = resp.Body.Close()
			if status != http.StatusOK {
				t.Fatalf("Wrong status. expect: %d, got: %d", http.StatusOK, status)
			}

			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("the metrics endpoint was not served after the address was freed. err: %v", errGet)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
