// Package carriertrackhandler looks up the status of a package at a carrier
// for an AI's tool call, in the same process, with the customer's own API key
// (VOIP-1677).
//
// The adapters of the carriers are added one by one, each together with its
// line in the carrier table of models/mcpserver: a carrier is open while both
// exist, and removing both closes it.
//
// The rules every adapter works under:
//   - the key is read only after the tracking number has been checked, is held
//     in a mcpserver.Credential that prints as "[redacted]", and goes to the
//     carrier in a header or a body, never in a url;
//   - what the model, the stored message and the customer's webhook receive is
//     the allow-list Result or one of the fixed texts of Message, nothing the
//     carrier wrote;
//   - memory is bounded: a result cache, a token cache and a cooldown map with
//     a fixed number of entries and a time to live;
//   - at most two lookups per carrier run at once in a pod, and a call that
//     cannot start is refused at once.
package carriertrackhandler

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

const (
	// defaultTimeoutSeconds is used when the configured tool call timeout is
	// not positive, as mcptoolhandler does.
	defaultTimeoutSeconds = 10

	// budgetReserve is kept out of the call's time for the work around the
	// lookup (the message writes after the tool returns).
	budgetReserve = 2 * time.Second

	// minBudget is the least time worth starting a lookup with.
	minBudget = time.Second

	// maxConcurrentPerCarrier bounds the lookups of one carrier running at
	// once in this process. The tool calls share ten RPC workers.
	maxConcurrentPerCarrier = 2

	resultTTL       = 120 * time.Second
	resultLimit     = 256
	resultMaxBytes  = 16 << 10
	tokenLimit      = 128
	cooldownLimit   = 256
	maxNumberLen    = 64
	outcomeInternal = "internal_error"
)

var errPanic = errors.New("carrier adapter failed")
var errSecret = errors.New("could not read the saved key")

// Handler runs the tracking tool calls of the registered carriers.
type Handler struct {
	adapters map[string]Adapter

	timeout        time.Duration
	reserve        time.Duration
	minBudget      time.Duration
	resultMaxBytes int
	now            func() time.Time
	newClient      func(timeout time.Duration) *http.Client

	hmacKey []byte
	results *lruTTL[cachedEntry]
	tokens  *TokenCache

	mu     sync.Mutex
	states map[string]*carrierState
}

// carrierState is what is kept per carrier, so that one carrier's limits and
// cooldowns never use another's.
type carrierState struct {
	slots     chan struct{}
	cooldowns *lruTTL[ErrCode]
}

// cachedEntry is a stored outcome: a result, or the code of a not_found or
// invalid_format answer.
type cachedEntry struct {
	result *Result
	code   ErrCode
}

// options are the settings of newHandler; the tests set what they need to.
type options struct {
	adapters  []Adapter
	timeout   time.Duration
	reserve   time.Duration
	minBudget time.Duration
	now       func() time.Time
	newClient func(timeout time.Duration) *http.Client
}

// New returns the production handler, with the adapters of the open carriers.
// timeoutSeconds is mcp_tool_call_timeout_seconds; a value of zero or less
// selects 10, as in mcptoolhandler.
func New(timeoutSeconds int) (*Handler, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeoutSeconds
	}
	return newHandler(options{
		timeout:  time.Duration(timeoutSeconds) * time.Second,
		adapters: []Adapter{newFedExAdapter(), NewUPS()},
	})
}

func newHandler(o options) (*Handler, error) {
	if o.now == nil {
		o.now = time.Now
	}
	if o.newClient == nil {
		o.newClient = mcpserverhandler.NewSSRFGuardedClient
	}
	if o.reserve == 0 {
		o.reserve = budgetReserve
	}
	if o.minBudget == 0 {
		o.minBudget = minBudget
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, errors.New("carriertrackhandler: could not create the cache key")
	}

	h := &Handler{
		adapters:       map[string]Adapter{},
		timeout:        o.timeout,
		reserve:        o.reserve,
		minBudget:      o.minBudget,
		resultMaxBytes: resultMaxBytes,
		now:            o.now,
		newClient:      o.newClient,
		hmacKey:        key,
		results:        newLRU[cachedEntry](resultLimit, o.now),
		tokens:         newTokenCache(tokenLimit, o.now),
		states:         map[string]*carrierState{},
	}
	for _, a := range o.adapters {
		h.adapters[a.Carrier()] = a
	}
	return h, nil
}

// state returns the per-carrier state, created on first use.
func (h *Handler) state(carrier string) *carrierState {
	h.mu.Lock()
	defer h.mu.Unlock()

	st, ok := h.states[carrier]
	if !ok {
		st = &carrierState{
			slots:     make(chan struct{}, maxConcurrentPerCarrier),
			cooldowns: newLRU[ErrCode](cooldownLimit, h.now),
		}
		h.states[carrier] = st
	}
	return st
}

// key derives an opaque cache key. The keyed hash keeps the customer id, the
// carrier key and the tracking number out of memory dumps and metric labels,
// and the process-local random key makes the hashes useless elsewhere.
func (h *Handler) key(kind string, parts ...string) string {
	m := hmac.New(sha256.New, h.hmacKey)
	m.Write([]byte(kind))
	for _, p := range parts {
		m.Write([]byte{0})
		m.Write([]byte(p))
	}
	return hex.EncodeToString(m.Sum(nil))
}

// Tools returns the tool definitions of carrier, and false when no adapter is
// registered for it.
func (h *Handler) Tools(carrier string) ([]mcpserver.CarrierToolDef, bool) {
	a, ok := h.adapters[carrier]
	if !ok {
		return nil, false
	}
	return []mcpserver.CarrierToolDef{a.Tool()}, true
}

// outcome is the end of one call.
type outcome struct {
	text    string
	isError bool
	label   string
	err     error
}

func (h *Handler) fail(code ErrCode, carrier string, format mcpserver.CredentialFormat) outcome {
	return outcome{text: Message(code, carrier, format), isError: true, label: string(code)}
}

// Call runs one tool call. A refusal the model can act on is returned as the
// text of an error result with a nil error; err is set only for a fault of the
// service itself (the key could not be decrypted, a panic).
func (h *Handler) Call(ctx context.Context, call mcpserver.CarrierCall) (text string, isError bool, err error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Call",
		"carrier":       call.Carrier,
		"mcp_server_id": call.ServerID,
	})

	a, ok := h.adapters[call.Carrier]
	if !ok {
		log.Warnf("No adapter is registered for the carrier.")
		return mcpserver.IntegrationUnavailableText, true, nil
	}

	start := h.now()
	label := outcomeInternal
	defer func() {
		// The recovered value and the stack are not logged: the call's
		// arguments, key included, may be on the stack.
		if r := recover(); r != nil {
			log.Errorf("The carrier adapter panicked.")
			text, isError, err = "", false, errPanic
			label = outcomeInternal
		}
		promLookupTotal.WithLabelValues(call.Carrier, label).Inc()
		log.WithFields(logrus.Fields{"outcome": label, "duration_ms": h.now().Sub(start).Milliseconds()}).Debugf("Finished the carrier lookup.")
	}()

	out := h.call(ctx, a, call, log)
	label = out.label
	return out.text, out.isError, out.err
}

func (h *Handler) call(ctx context.Context, a Adapter, call mcpserver.CarrierCall, log *logrus.Entry) outcome {
	carrier := call.Carrier
	fail := func(code ErrCode) outcome { return h.fail(code, carrier, call.Credential) }

	// 1. The tool name.
	if call.ToolName != a.Tool().Name {
		return fail(CodeToolUnknown)
	}

	// 2. The tracking number, before the key is touched.
	number, ok := parseNumber(a, call.ArgumentsJSON)
	if !ok {
		return fail(CodeInvalidFormat)
	}

	// 3. The saved key.
	secret, present, err := call.Secret()
	if err != nil {
		log.Errorf("Could not read the saved key.")
		return outcome{label: outcomeInternal, err: errSecret}
	}
	if !present || strings.TrimSpace(secret) == "" {
		return fail(CodeCredentialNotConfigured)
	}
	cred, err := mcpserver.ParseCredential(call.Credential, secret)
	if err != nil {
		return fail(CodeCredentialMalformed)
	}

	customer := call.CustomerID.String()
	resultKey := h.key("result", customer, carrier, secret, number)
	cooldownKey := h.key("cooldown", customer, carrier, secret)
	tokenKey := h.key("token", customer, carrier, secret)
	st := h.state(carrier)

	// 4. The result cache.
	if entry, hit := h.results.get(resultKey); hit {
		if entry.result == nil {
			return fail(entry.code)
		}
		r := *entry.result
		r.Cached = true
		text, err := r.marshal()
		if err != nil {
			return outcome{label: outcomeInternal, err: errPanic}
		}
		return outcome{text: text, label: outcomeCacheHit}
	}

	// 5. The cooldown of a rejected or limited key.
	if code, blocked := st.cooldowns.get(cooldownKey); blocked {
		return fail(code)
	}

	// 6. The time budget: what is left of the call's time minus a reserve.
	budget, ok := h.budget(ctx)
	if !ok {
		return fail(CodeCarrierUnavailable)
	}

	// 7. A free place for this carrier, or an answer at once.
	select {
	case st.slots <- struct{}{}:
	default:
		return fail(CodeServiceBusy)
	}
	promInflight.WithLabelValues(carrier).Inc()
	defer func() {
		promInflight.WithLabelValues(carrier).Dec()
		<-st.slots
	}()

	// 8. The lookup, on a client made for this call.
	budgetCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	client := h.newClient(budget)
	defer client.CloseIdleConnections()

	maxBody := a.MaxBody()
	if maxBody <= 0 {
		maxBody = mcpserverhandler.McpHTTPResponseSizeCapBytes
	}
	deps := Deps{
		Carrier:  carrier,
		Client:   client,
		Now:      h.now,
		Tokens:   h.tokens,
		TokenKey: tokenKey,
		MaxBody:  maxBody,
	}

	started := time.Now()
	res, cerr := a.Track(budgetCtx, deps, cred, number)
	promLookupDurationSeconds.WithLabelValues(carrier).Observe(time.Since(started).Seconds())

	// An answer that came after the deadline is thrown away, and nothing is
	// stored for it.
	if budgetCtx.Err() != nil {
		return fail(CodeCarrierUnavailable)
	}

	return h.finish(res, cerr, lookupKeys{result: resultKey, cooldown: cooldownKey, token: tokenKey}, st, call, number, log)
}

type lookupKeys struct{ result, cooldown, token string }

// budget returns how long one lookup may take: what is left of the caller's
// time (or the configured timeout) minus a reserve. It is false when too little
// is left to try.
func (h *Handler) budget(ctx context.Context) (time.Duration, bool) {
	remaining := h.timeout
	if dl, has := ctx.Deadline(); has {
		if r := time.Until(dl); r < remaining {
			remaining = r
		}
	}
	budget := remaining - h.reserve
	if ctx.Err() != nil || budget < h.minBudget {
		return 0, false
	}
	return budget, true
}

// finish turns what the adapter returned into an outcome and stores what may
// be stored.
func (h *Handler) finish(res *Result, cerr *CarrierError, keys lookupKeys, st *carrierState, call mcpserver.CarrierCall, number string, log *logrus.Entry) outcome {
	carrier := call.Carrier
	fail := func(code ErrCode) outcome { return h.fail(code, carrier, call.Credential) }

	if cerr != nil {
		if cerr.DropToken {
			h.tokens.Delete(keys.token)
		}
		code := cerr.Code
		if !knownCode(code) {
			code = CodeCarrierUnavailable
		}
		if cd := cerr.cooldown(); cd > 0 {
			st.cooldowns.put(keys.cooldown, code, cd)
		}
		if code == CodeNotFound || code == CodeInvalidFormat {
			h.results.put(keys.result, cachedEntry{code: code}, resultTTL)
		}
		return fail(code)
	}
	if res == nil {
		return fail(CodeCarrierUnavailable)
	}

	res.finalize(carrier, number, h.now())
	text, err := res.marshal()
	if err != nil {
		return outcome{label: outcomeInternal, err: errPanic}
	}
	if len(text) > h.resultMaxBytes {
		log.Warnf("The result is over the cache entry limit and is not cached.")
	} else {
		h.results.put(keys.result, cachedEntry{result: res}, resultTTL)
	}
	return outcome{text: text, label: outcomeSuccess}
}

// parseNumber reads the tracking number of the arguments and asks the adapter
// to normalize it.
func parseNumber(a Adapter, argumentsJSON string) (string, bool) {
	var in struct {
		TrackingNumber *string `json:"tracking_number"`
	}
	if err := json.Unmarshal([]byte(argumentsJSON), &in); err != nil || in.TrackingNumber == nil {
		return "", false
	}
	raw := strings.TrimSpace(*in.TrackingNumber)
	if raw == "" || len(raw) > maxNumberLen || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return "", false
	}
	return a.NormalizeNumber(raw)
}

func knownCode(code ErrCode) bool {
	switch code {
	case CodeCredentialNotConfigured, CodeCredentialMalformed, CodeCredentialInvalid, CodeRateLimited,
		CodeCarrierUnavailable, CodeServiceBusy, CodeInvalidFormat, CodeNotFound, CodeToolUnknown:
		return true
	}
	return false
}
