package carriertrackhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/models/mcpserver"
)

const (
	testSecretID    = "SENTINEL-ID-VALUE"
	testSecretValue = "SENTINEL-SECRET-VALUE"
	testSecret      = testSecretID + ":" + testSecretValue
	testNumber      = "FAKE12345678"
)

// fakeAdapter is a carrier adapter whose behaviour a test sets.
type fakeAdapter struct {
	carrier string
	calls   atomic.Int32
	track   func(ctx context.Context, d Deps, cred mcpserver.Credential, number string) (*Result, *CarrierError)

	mu       sync.Mutex
	lastDeps Deps
}

func newFake(carrier string) *fakeAdapter {
	return &fakeAdapter{carrier: carrier}
}

func (a *fakeAdapter) Carrier() string { return a.carrier }
func (a *fakeAdapter) Tool() mcpserver.CarrierToolDef {
	return mcpserver.CarrierToolDef{Name: "track_" + a.carrier, Description: "Look up a fake package.", InputSchema: json.RawMessage(`{"type":"object","properties":{"tracking_number":{"type":"string"}},"required":["tracking_number"]}`)}
}
func (a *fakeAdapter) NormalizeNumber(raw string) (string, bool) {
	n := CleanNumber(raw)
	return n, LooseNumber(n, 8, 40)
}
func (a *fakeAdapter) MaxBody() int64 { return 1234 }
func (a *fakeAdapter) Track(ctx context.Context, d Deps, cred mcpserver.Credential, number string) (*Result, *CarrierError) {
	a.calls.Add(1)
	a.mu.Lock()
	a.lastDeps = d
	a.mu.Unlock()
	if a.track != nil {
		return a.track(ctx, d, cred, number)
	}
	return &Result{Status: StatusInTransit, StatusText: "In transit", Events: []Event{{Time: str("2026-01-01T00:00:00Z"), Description: "Scan"}}}, nil
}

func newTestHandler(t *testing.T, o options, adapters ...Adapter) (*Handler, *fakeClock) {
	t.Helper()
	clk := newFakeClock()
	if o.now == nil {
		o.now = clk.now
	}
	if o.timeout == 0 {
		o.timeout = 10 * time.Second
	}
	if o.reserve == 0 {
		o.reserve = 2 * time.Second
	}
	if o.minBudget == 0 {
		o.minBudget = time.Second
	}
	o.adapters = adapters
	h, err := newHandler(o)
	if err != nil {
		t.Fatal(err)
	}
	return h, clk
}

var customerA = uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
var customerB = uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

type callOpt func(*mcpserver.CarrierCall)

func makeCall(carrier, number, secret string, opts ...callOpt) mcpserver.CarrierCall {
	c := mcpserver.CarrierCall{
		CustomerID:    customerA,
		ServerID:      uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		Carrier:       carrier,
		Credential:    mcpserver.CredentialClientIDSecret,
		ToolName:      "track_" + carrier,
		ArgumentsJSON: fmt.Sprintf(`{"tracking_number":%q}`, number),
		Secret:        func() (string, bool, error) { return secret, secret != "", nil },
	}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func decode(t *testing.T, text string) Result {
	t.Helper()
	var r Result
	if err := json.Unmarshal([]byte(text), &r); err != nil {
		t.Fatalf("not a result: %q: %v", text, err)
	}
	return r
}

func expectError(t *testing.T, text string, isError bool, err error, code ErrCode, carrier string) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected system error: %v", err)
	}
	if !isError {
		t.Fatalf("expected an error result %s, got %q", code, text)
	}
	if want := Message(code, carrier, mcpserver.CredentialClientIDSecret); text != want {
		t.Fatalf("expected the text of %s (%q), got %q", code, want, text)
	}
}

func Test_Tools(t *testing.T) {
	h, _ := newTestHandler(t, options{}, newFake("fake"))

	defs, ok := h.Tools("fake")
	if !ok || len(defs) != 1 || defs[0].Name != "track_fake" {
		t.Errorf("got %+v %v", defs, ok)
	}
	if _, ok := h.Tools("other"); ok {
		t.Errorf("an unregistered carrier has no tools")
	}
}

func Test_New_registersTheOpenCarriersAndFixesTheTimeout(t *testing.T) {
	h, err := New(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.adapters) != 1 || h.adapters["fedex"] == nil {
		t.Errorf("only the fedex adapter is registered, got %d", len(h.adapters))
	}
	if h.timeout != 10*time.Second {
		t.Errorf("a timeout of 0 must become 10 seconds, got %v", h.timeout)
	}
	if h2, _ := New(7); h2.timeout != 7*time.Second {
		t.Errorf("timeout = %v", h2.timeout)
	}
	if tools, ok := h.Tools("fedex"); !ok || len(tools) != 1 || tools[0].Name != "track_fedex" {
		t.Errorf("unexpected tools %v %v", tools, ok)
	}
	if _, ok := h.Tools("ups"); ok {
		t.Errorf("no tools without an adapter")
	}

	// a carrier that has no adapter is closed with the fixed text
	text, isError, err := h.Call(context.Background(), makeCall("ups", testNumber, testSecret))
	if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
		t.Errorf("a call to an absent adapter must close with the fixed text, got %q %v %v", text, isError, err)
	}

	// the production adapter is only called here without a saved key, which
	// ends the call before any request is made
	call := makeCall("fedex", testNumber, "")
	text, isError, err = h.Call(context.Background(), call)
	expectError(t, text, isError, err, CodeCredentialNotConfigured, "fedex")
}

func Test_Call_success(t *testing.T) {
	a := newFake("fake")
	h, clk := newTestHandler(t, options{}, a)

	text, isError, err := h.Call(context.Background(), makeCall("fake", " fake-1234 5678 ", testSecret))
	if err != nil || isError {
		t.Fatalf("got %q %v %v", text, isError, err)
	}
	r := decode(t, text)
	if r.Carrier != "fake" || r.TrackingNumber != testNumber || r.Cached || r.FetchedAt != clk.now().Format(time.RFC3339) || r.Status != StatusInTransit {
		t.Errorf("result wrong: %+v", r)
	}
	if len(r.Display.Fields) == 0 {
		t.Errorf("the card is missing")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lastDeps.MaxBody != 1234 || a.lastDeps.Client == nil || a.lastDeps.TokenKey == "" || a.lastDeps.Carrier != "fake" {
		t.Errorf("deps wrong: %+v", a.lastDeps)
	}
	if strings.Contains(a.lastDeps.TokenKey, testSecretValue) || strings.Contains(a.lastDeps.TokenKey, customerA.String()) {
		t.Errorf("the token key must be a keyed hash")
	}
}

func Test_Call_defaultBodyLimit(t *testing.T) {
	a := &fakeAdapterZeroBody{}
	a.carrier = "fake"
	h, _ := newTestHandler(t, options{}, a)

	if _, _, err := h.Call(context.Background(), makeCall("fake", testNumber, testSecret)); err != nil {
		t.Fatal(err)
	}
	if a.lastDeps.MaxBody != 1<<20 {
		t.Errorf("a zero MaxBody must select the 1 MiB response limit, got %d", a.lastDeps.MaxBody)
	}
}

type fakeAdapterZeroBody struct{ fakeAdapter }

func (a *fakeAdapterZeroBody) MaxBody() int64 { return 0 }
func (a *fakeAdapterZeroBody) Track(ctx context.Context, d Deps, c mcpserver.Credential, n string) (*Result, *CarrierError) {
	a.lastDeps = d
	return &Result{Status: StatusDelivered}, nil
}

func Test_Call_evaluationOrder(t *testing.T) {
	t.Run("an unknown tool name is refused before anything is read", func(t *testing.T) {
		a := newFake("fake")
		h, _ := newTestHandler(t, options{}, a)
		secretRead := false
		call := makeCall("fake", testNumber, testSecret, func(c *mcpserver.CarrierCall) {
			c.ToolName = "other_tool"
			c.Secret = func() (string, bool, error) { secretRead = true; return testSecret, true, nil }
		})
		text, isError, err := h.Call(context.Background(), call)
		expectError(t, text, isError, err, CodeToolUnknown, "fake")
		if secretRead || a.calls.Load() != 0 {
			t.Errorf("nothing may be read or called")
		}
	})

	t.Run("a bad number is reported before the secret is touched", func(t *testing.T) {
		for _, args := range []string{`{"tracking_number":"abc"}`, `{}`, `not json`, `{"tracking_number":123}`, `{"tracking_number":"FAKE1234\u00005678"}`, `{"tracking_number":"` + strings.Repeat("A", 100) + `"}`, ``} {
			a := newFake("fake")
			h, _ := newTestHandler(t, options{}, a)
			secretRead := false
			call := makeCall("fake", "", "", func(c *mcpserver.CarrierCall) {
				c.ArgumentsJSON = args
				c.Secret = func() (string, bool, error) { secretRead = true; return "", false, nil }
			})
			text, isError, err := h.Call(context.Background(), call)
			expectError(t, text, isError, err, CodeInvalidFormat, "fake")
			if secretRead || a.calls.Load() != 0 {
				t.Errorf("args %q: the secret was read or the carrier was called", args)
			}
		}
	})

	t.Run("a missing secret", func(t *testing.T) {
		a := newFake("fake")
		h, _ := newTestHandler(t, options{}, a)
		text, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, ""))
		expectError(t, text, isError, err, CodeCredentialNotConfigured, "fake")
		if a.calls.Load() != 0 {
			t.Errorf("the carrier must not be called without a key")
		}
	})

	t.Run("a malformed secret", func(t *testing.T) {
		for _, secret := range []string{"nocolon", ":x", "x:", "a b:c"} {
			a := newFake("fake")
			h, _ := newTestHandler(t, options{}, a)
			text, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, secret))
			expectError(t, text, isError, err, CodeCredentialMalformed, "fake")
			if a.calls.Load() != 0 {
				t.Errorf("the carrier must not be called with a malformed key")
			}
		}
	})

	t.Run("a secret that cannot be decrypted is a system fault with no detail", func(t *testing.T) {
		a := newFake("fake")
		h, _ := newTestHandler(t, options{}, a)
		call := makeCall("fake", testNumber, testSecret, func(c *mcpserver.CarrierCall) {
			c.Secret = func() (string, bool, error) { return "", false, errors.New("cipher: message authentication failed") }
		})
		text, isError, err := h.Call(context.Background(), call)
		if err == nil || isError || text != "" {
			t.Fatalf("expected a system error, got %q %v %v", text, isError, err)
		}
		if strings.Contains(err.Error(), "cipher") {
			t.Errorf("the underlying error must not be passed on: %v", err)
		}
	})
}

func Test_Call_resultCache(t *testing.T) {
	a := newFake("fake")
	h, clk := newTestHandler(t, options{}, a)

	call := func(opts ...callOpt) Result {
		text, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, testSecret, opts...))
		if err != nil || isError {
			t.Fatalf("%q %v %v", text, isError, err)
		}
		return decode(t, text)
	}

	first := call()
	second := call()
	if a.calls.Load() != 1 || first.Cached || !second.Cached {
		t.Fatalf("calls = %d, first cached = %v, second cached = %v", a.calls.Load(), first.Cached, second.Cached)
	}
	if second.FetchedAt != first.FetchedAt {
		t.Errorf("a cached result keeps the time it was fetched")
	}

	// the same number written differently is the same key
	call(func(c *mcpserver.CarrierCall) { c.ArgumentsJSON = `{"tracking_number":"fake-1234 5678"}` })
	if a.calls.Load() != 1 {
		t.Errorf("a normalized number must hit the cache")
	}

	// another customer with the same key does not share the entry
	call(func(c *mcpserver.CarrierCall) { c.CustomerID = customerB })
	if a.calls.Load() != 2 {
		t.Errorf("a result must not be shared between customers")
	}

	// a replaced key does not hit the old entry
	text, _, _ := h.Call(context.Background(), makeCall("fake", testNumber, "other-id:other-secret"))
	_ = text
	if a.calls.Load() != 3 {
		t.Errorf("a changed key must not hit the old entry")
	}

	// another number
	h.Call(context.Background(), makeCall("fake", "FAKE99999999", testSecret))
	if a.calls.Load() != 4 {
		t.Errorf("another number must call the carrier")
	}

	clk.add(119 * time.Second)
	call()
	if a.calls.Load() != 4 {
		t.Errorf("the entry must live for 120 seconds")
	}
	clk.add(2 * time.Second)
	call()
	if a.calls.Load() != 5 {
		t.Errorf("the entry must expire after 120 seconds, calls = %d", a.calls.Load())
	}
}

func Test_Call_onlyNotFoundAndInvalidFormatAreCached(t *testing.T) {
	// codes with a default cooldown are covered by Test_Call_cooldown
	tests := []struct {
		code        ErrCode
		expectCache bool
	}{
		{CodeNotFound, true},
		{CodeInvalidFormat, true},
		{CodeCarrierUnavailable, false},
		{CodeServiceBusy, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			a := newFake("fake")
			a.track = func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) {
				return nil, &CarrierError{Code: tt.code}
			}
			h, _ := newTestHandler(t, options{}, a)

			for i := 0; i < 2; i++ {
				text, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
				expectError(t, text, isError, err, tt.code, "fake")
			}
			if cached := a.calls.Load() == 1; cached != tt.expectCache {
				t.Errorf("cached = %v (calls %d), expected %v", cached, a.calls.Load(), tt.expectCache)
			}
		})
	}
}

func Test_Call_oversizedResultIsNotCached(t *testing.T) {
	a := newFake("fake")
	h, _ := newTestHandler(t, options{}, a)
	h.resultMaxBytes = 200 // smaller than any result

	logs := logtest.New(t)
	for i := 0; i < 2; i++ {
		if _, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, testSecret)); err != nil || isError {
			t.Fatal(err)
		}
	}
	if a.calls.Load() != 2 {
		t.Errorf("a result over the entry limit must not be cached")
	}
	if !strings.Contains(logs.Dump(), "not cached") {
		t.Errorf("the skip must leave a log line")
	}
	logs.AssertAbsent(t, testSecret, testSecretValue)
}

func Test_Call_cooldown(t *testing.T) {
	tests := []struct {
		name       string
		err        *CarrierError
		cooldown   time.Duration
		expectCode ErrCode
	}{
		{"a rejected key waits ten minutes", &CarrierError{Code: CodeCredentialInvalid}, 10 * time.Minute, CodeCredentialInvalid},
		{"a rate limited key waits a minute", &CarrierError{Code: CodeRateLimited}, time.Minute, CodeRateLimited},
		{"an explicit short block", &CarrierError{Code: CodeCarrierUnavailable, Cooldown: time.Minute}, time.Minute, CodeCarrierUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newFake("fake")
			a.track = func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) { return nil, tt.err }
			h, clk := newTestHandler(t, options{}, a)

			text, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
			expectError(t, text, isError, err, tt.expectCode, "fake")

			// a different number with the same key is blocked without a carrier call
			text, isError, err = h.Call(context.Background(), makeCall("fake", "FAKE99999999", testSecret))
			expectError(t, text, isError, err, tt.expectCode, "fake")
			if a.calls.Load() != 1 {
				t.Fatalf("the carrier must not be called during the cooldown, calls = %d", a.calls.Load())
			}

			// another customer is not blocked
			h.Call(context.Background(), makeCall("fake", testNumber, testSecret, func(c *mcpserver.CarrierCall) { c.CustomerID = customerB }))
			if a.calls.Load() != 2 {
				t.Errorf("the cooldown must not cross customers")
			}

			// a replaced key is tried at once
			h.Call(context.Background(), makeCall("fake", testNumber, "new-id:new-secret"))
			if a.calls.Load() != 3 {
				t.Errorf("a re-entered key must be tried at once")
			}

			clk.add(tt.cooldown - time.Second)
			h.Call(context.Background(), makeCall("fake", "FAKE99999999", testSecret))
			if a.calls.Load() != 3 {
				t.Errorf("still inside the cooldown")
			}
			clk.add(2 * time.Second)
			h.Call(context.Background(), makeCall("fake", "FAKE99999999", testSecret))
			if a.calls.Load() != 4 {
				t.Errorf("the cooldown must end, calls = %d", a.calls.Load())
			}
		})
	}
}

func Test_Call_cooldownIsPerCarrier(t *testing.T) {
	a, b := newFake("fake"), newFake("other")
	a.track = func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) {
		return nil, &CarrierError{Code: CodeCredentialInvalid}
	}
	h, _ := newTestHandler(t, options{}, a, b)

	h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
	text, isError, err := h.Call(context.Background(), makeCall("other", testNumber, testSecret))
	if err != nil || isError {
		t.Fatalf("%q %v %v", text, isError, err)
	}
	if b.calls.Load() != 1 {
		t.Errorf("the other carrier must be called")
	}
}

func Test_Call_cooldownMapIsBounded(t *testing.T) {
	a := newFake("fake")
	a.track = func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) {
		return nil, &CarrierError{Code: CodeCredentialInvalid}
	}
	h, _ := newTestHandler(t, options{}, a)

	for i := 0; i < cooldownLimit+50; i++ {
		h.Call(context.Background(), makeCall("fake", testNumber, fmt.Sprintf("id%d:secret%d", i, i)))
	}
	if n := h.state("fake").cooldowns.len(); n != cooldownLimit {
		t.Errorf("the cooldown map holds %d entries, expected the limit %d", n, cooldownLimit)
	}
}

func Test_Call_dropToken(t *testing.T) {
	a := newFake("fake")
	var key string
	a.track = func(ctx context.Context, d Deps, c mcpserver.Credential, n string) (*Result, *CarrierError) {
		key = d.TokenKey
		d.Tokens.Put(d.TokenKey, "tok", time.Hour)
		return nil, &CarrierError{Code: CodeCarrierUnavailable, DropToken: true}
	}
	h, _ := newTestHandler(t, options{}, a)

	text, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
	expectError(t, text, isError, err, CodeCarrierUnavailable, "fake")
	if _, ok := h.tokens.Get(key); ok {
		t.Errorf("the token must be dropped")
	}
}

func Test_Call_concurrencyLimit(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{}, 10)
	a := newFake("fake")
	a.track = func(ctx context.Context, d Deps, c mcpserver.Credential, n string) (*Result, *CarrierError) {
		started <- struct{}{}
		<-block
		return &Result{Status: StatusDelivered}, nil
	}
	other := newFake("other")
	h, _ := newTestHandler(t, options{}, a, other)

	var wg sync.WaitGroup
	for i := 0; i < maxConcurrentPerCarrier; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h.Call(context.Background(), makeCall("fake", fmt.Sprintf("FAKE0000000%d", i), testSecret))
		}(i)
	}
	for i := 0; i < maxConcurrentPerCarrier; i++ {
		<-started
	}

	// the carrier is full: the next call is refused at once, without waiting
	done := make(chan struct{})
	go func() {
		defer close(done)
		text, isError, err := h.Call(context.Background(), makeCall("fake", "FAKE00000099", testSecret))
		expectError(t, text, isError, err, CodeServiceBusy, "fake")
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a full carrier must refuse at once, not wait")
	}

	// another carrier is not affected
	text, isError, err := h.Call(context.Background(), makeCall("other", testNumber, testSecret))
	if err != nil || isError {
		t.Errorf("another carrier must not be affected: %q %v %v", text, isError, err)
	}

	if got := testutil.ToFloat64(promInflight.WithLabelValues("fake")); got != float64(maxConcurrentPerCarrier) {
		t.Errorf("inflight gauge = %v", got)
	}

	close(block)
	wg.Wait()

	if got := testutil.ToFloat64(promInflight.WithLabelValues("fake")); got != 0 {
		t.Errorf("inflight gauge after the calls = %v", got)
	}
	text, isError, err = h.Call(context.Background(), makeCall("fake", "FAKE00000077", testSecret))
	if err != nil || isError {
		t.Errorf("the slots must be free again: %q %v %v", text, isError, err)
	}
}

func Test_Call_slotsAreReleasedOnEveryPath(t *testing.T) {
	paths := map[string]func(ctx context.Context, d Deps, c mcpserver.Credential, n string) (*Result, *CarrierError){
		"error": func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) {
			return nil, &CarrierError{Code: CodeCarrierUnavailable}
		},
		"nil": func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) { return nil, nil },
		"panic": func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) {
			panic("boom " + testSecretValue)
		},
		"success": func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) {
			return &Result{}, nil
		},
	}
	for name, track := range paths {
		t.Run(name, func(t *testing.T) {
			a := newFake("fake")
			a.track = track
			h, _ := newTestHandler(t, options{}, a)
			for i := 0; i < maxConcurrentPerCarrier*3; i++ {
				h.Call(context.Background(), makeCall("fake", fmt.Sprintf("FAKE0000000%d", i), testSecret))
			}
			if n := len(h.state("fake").slots); n != 0 {
				t.Errorf("%d slots leaked", n)
			}
		})
	}
}

func Test_Call_panicIsContained(t *testing.T) {
	a := newFake("fake")
	a.track = func(context.Context, Deps, mcpserver.Credential, string) (*Result, *CarrierError) {
		panic("boom " + testSecretValue)
	}
	h, _ := newTestHandler(t, options{}, a)
	logs := logtest.New(t)

	text, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
	if err == nil || isError || text != "" {
		t.Fatalf("a panic must come back as a system error, got %q %v %v", text, isError, err)
	}
	if strings.Contains(err.Error(), "boom") || strings.Contains(err.Error(), testSecretValue) {
		t.Errorf("the panic value must not be in the error: %v", err)
	}
	logs.AssertAbsent(t, "boom", testSecretValue, testSecretID, "goroutine", ".go:")
	if !logs.HasField("carrier") {
		t.Errorf("the recovery log must name the carrier")
	}
}

func Test_Call_budget(t *testing.T) {
	t.Run("too little time left calls nothing", func(t *testing.T) {
		a := newFake("fake")
		h, _ := newTestHandler(t, options{}, a)

		// 10 s timeout, 2.5 s left on the caller's context, 2 s reserved: 0.5 s is under the 1 s floor
		ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
		defer cancel()
		text, isError, err := h.Call(ctx, makeCall("fake", testNumber, testSecret))
		expectError(t, text, isError, err, CodeCarrierUnavailable, "fake")
		if a.calls.Load() != 0 {
			t.Errorf("the carrier must not be called")
		}
	})

	t.Run("an expired context calls nothing", func(t *testing.T) {
		a := newFake("fake")
		h, _ := newTestHandler(t, options{}, a)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		text, isError, err := h.Call(ctx, makeCall("fake", testNumber, testSecret))
		expectError(t, text, isError, err, CodeCarrierUnavailable, "fake")
		if a.calls.Load() != 0 {
			t.Errorf("the carrier must not be called")
		}
	})

	t.Run("the budget is the timeout minus the reserve", func(t *testing.T) {
		a := newFake("fake")
		var deadline time.Duration
		a.track = func(ctx context.Context, d Deps, c mcpserver.Credential, n string) (*Result, *CarrierError) {
			dl, ok := ctx.Deadline()
			if !ok {
				t.Error("the call must have a deadline")
			}
			deadline = time.Until(dl)
			return &Result{}, nil
		}
		h, _ := newTestHandler(t, options{timeout: 5 * time.Second, reserve: 2 * time.Second}, a)
		h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
		if deadline > 3*time.Second || deadline < 2*time.Second {
			t.Errorf("budget = %v, expected about 3 s", deadline)
		}
	})

	t.Run("a slow carrier is cut at the budget and nothing is stored", func(t *testing.T) {
		a := newFake("fake")
		a.track = func(ctx context.Context, d Deps, c mcpserver.Credential, n string) (*Result, *CarrierError) {
			<-ctx.Done()
			// the answer arrives after the deadline and must be thrown away
			d.Tokens.Put(d.TokenKey, "late", time.Hour)
			return &Result{Status: StatusDelivered}, nil
		}
		h, _ := newTestHandler(t, options{timeout: 400 * time.Millisecond, reserve: 100 * time.Millisecond, minBudget: 100 * time.Millisecond}, a)

		start := time.Now()
		text, isError, err := h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
		expectError(t, text, isError, err, CodeCarrierUnavailable, "fake")
		if time.Since(start) > 3*time.Second {
			t.Errorf("the call must end at the budget")
		}

		a.track = nil
		h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
		if a.calls.Load() != 2 {
			t.Errorf("a late result must not have been cached, calls = %d", a.calls.Load())
		}
	})
}

func Test_Call_clientIsMadePerCallAndClosed(t *testing.T) {
	var made, closed atomic.Int32
	a := newFake("fake")
	h, _ := newTestHandler(t, options{newClient: func(timeout time.Duration) *http.Client {
		made.Add(1)
		return &http.Client{Transport: &countingTransport{closed: &closed}}
	}}, a)

	h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
	h.Call(context.Background(), makeCall("fake", "FAKE99999999", testSecret))
	if made.Load() != 2 || closed.Load() != 2 {
		t.Errorf("made %d clients and closed %d", made.Load(), closed.Load())
	}
}

type countingTransport struct{ closed *atomic.Int32 }

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("no network in this test")
}
func (c *countingTransport) CloseIdleConnections() { c.closed.Add(1) }

func Test_Call_secretNeverAppears(t *testing.T) {
	codes := []*CarrierError{
		nil,
		{Code: CodeCredentialInvalid}, {Code: CodeRateLimited}, {Code: CodeCarrierUnavailable}, {Code: CodeServiceBusy}, {Code: CodeNotFound}, {Code: CodeInvalidFormat},
	}
	for _, cerr := range codes {
		name := "success"
		if cerr != nil {
			name = string(cerr.Code)
		}
		t.Run(name, func(t *testing.T) {
			a := newFake("fake")
			a.track = func(ctx context.Context, d Deps, c mcpserver.Credential, n string) (*Result, *CarrierError) {
				// the adapter handles the credential; a misbehaving log of it must still be redacted
				logrusLike := fmt.Sprintf("%v %+v %#v %s", c, c, c, c)
				if strings.Contains(logrusLike, testSecretValue) {
					t.Errorf("credential printed: %s", logrusLike)
				}
				if cerr != nil {
					return nil, cerr
				}
				return &Result{Status: StatusDelivered, StatusText: "ok"}, nil
			}
			h, _ := newTestHandler(t, options{}, a)
			logs := logtest.New(t)

			call := makeCall("fake", testNumber, testSecret)
			var outputs []string
			for i := 0; i < 2; i++ {
				text, _, err := h.Call(context.Background(), call)
				if err != nil {
					outputs = append(outputs, err.Error())
				}
				outputs = append(outputs, text)
			}
			for _, o := range outputs {
				if strings.Contains(o, testSecretValue) || strings.Contains(o, testSecretID) {
					t.Errorf("a key leaked into %q", o)
				}
			}
			logs.AssertAbsent(t, testSecret, testSecretValue, testSecretID, customerA.String())
			logs.AssertField(t, "carrier")

			for _, key := range h.debugKeys("fake") {
				if strings.Contains(key, testSecretValue) || strings.Contains(key, customerA.String()) {
					t.Errorf("a cache key contains the key or the customer id: %q", key)
				}
			}
		})
	}
}

func Test_metrics_useAFixedLabelSet(t *testing.T) {
	a := newFake("fake")
	h, _ := newTestHandler(t, options{}, a)
	h.Call(context.Background(), makeCall("fake", testNumber, testSecret))
	h.Call(context.Background(), makeCall("fake", testNumber, testSecret))

	// The label names are a fixed set of two: the carrier name and the outcome.
	// Their values come from the adapter's name and the outcome codes, never
	// from the request.
	for c, want := range map[prometheus.Collector]string{
		promLookupTotal:           "variableLabels: {carrier,outcome}",
		promLookupDurationSeconds: "variableLabels: {carrier}",
		promInflight:              "variableLabels: {carrier}",
		promTokenExchangeTotal:    "variableLabels: {carrier,outcome}",
	} {
		ch := make(chan *prometheus.Desc, 10)
		c.Describe(ch)
		close(ch)
		for d := range ch {
			if !strings.Contains(d.String(), want) {
				t.Errorf("descriptor %s does not have the fixed labels %s", d.String(), want)
			}
		}
	}
	if got := testutil.ToFloat64(promLookupTotal.WithLabelValues("fake", outcomeSuccess)); got < 1 {
		t.Errorf("success not counted")
	}
	if got := testutil.ToFloat64(promLookupTotal.WithLabelValues("fake", outcomeCacheHit)); got < 1 {
		t.Errorf("cache hit not counted")
	}
}

func Test_Call_race(t *testing.T) {
	a := newFake("fake")
	a.track = func(ctx context.Context, d Deps, c mcpserver.Credential, n string) (*Result, *CarrierError) {
		time.Sleep(time.Millisecond)
		if strings.HasSuffix(n, "7") {
			return nil, &CarrierError{Code: CodeCredentialInvalid}
		}
		return &Result{Status: StatusDelivered}, nil
	}
	h, clk := newTestHandler(t, options{}, a)

	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				secret := testSecret
				if g%3 == 0 {
					secret = fmt.Sprintf("id%d:sec%d", g, g)
				}
				call := makeCall("fake", fmt.Sprintf("FAKE%08d", i%10), secret, func(c *mcpserver.CarrierCall) {
					if g%2 == 0 {
						c.CustomerID = customerB
					}
				})
				_, _, _ = h.Call(context.Background(), call)
				if i%10 == 0 {
					clk.add(time.Second) // the shared clock is read by the caches too
				}
			}
		}(g)
	}
	wg.Wait()
	if n := len(h.state("fake").slots); n != 0 {
		t.Errorf("%d slots leaked", n)
	}
}

// keys lists the keys of a cache, for the tests that check what a key holds.
func (c *lruTTL[V]) keys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	res := make([]string, 0, len(c.items))
	for k := range c.items {
		res = append(res, k)
	}
	return res
}

// debugKeys lists every key held in memory for a carrier.
func (h *Handler) debugKeys(carrier string) []string {
	res := append(h.results.keys(), h.tokens.lru.keys()...)
	return append(res, h.state(carrier).cooldowns.keys()...)
}
