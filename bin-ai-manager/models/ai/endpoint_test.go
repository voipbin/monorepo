package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func sameError(got, expect error) bool {
	if expect == nil {
		return got == nil
	}

	return errors.Is(got, expect)
}

func stubResolver(addrs ...string) func(ctx context.Context, host string) ([]netip.Addr, error) {
	return func(ctx context.Context, host string) ([]netip.Addr, error) {
		res := []netip.Addr{}
		for _, a := range addrs {
			res = append(res, netip.MustParseAddr(a))
		}
		return res, nil
	}
}

func Test_RedactedString(t *testing.T) {
	const secret = "sk-very-secret-1234"
	r := RedactedString(secret)

	type holder struct {
		Key RedactedString
	}
	h := holder{Key: r}

	outs := []string{
		r.String(),
		r.GoString(),
		fmt.Sprintf("%v", r),
		fmt.Sprintf("%s", r),
		fmt.Sprintf("%q", r),
		fmt.Sprintf("%+v", r),
		fmt.Sprintf("%#v", r),
		fmt.Sprintf("%v", h),
		fmt.Sprintf("%+v", h),
		fmt.Sprintf("%#v", h),
		fmt.Sprintf("%v", &h),
		fmt.Sprint(r),
	}
	for _, o := range outs {
		if strings.Contains(o, secret) {
			t.Errorf("secret leaked: %q", o)
		}
		if !strings.Contains(o, "<redacted>") {
			t.Errorf("missing marker: %q", o)
		}
	}
}

func Test_AI_FormatDoesNotLeakKey(t *testing.T) {
	const secret = "sk-very-secret-5678"
	a := &AI{
		EngineBaseURL:    "https://llm.example.com/v1",
		EngineBaseKeyEnc: []byte("ciphertext"),
		EngineBaseKey:    RedactedString(secret),
	}

	for _, o := range []string{
		fmt.Sprintf("%v", a), fmt.Sprintf("%+v", a), fmt.Sprintf("%#v", a),
		fmt.Sprintf("%v", *a), fmt.Sprintf("%+v", *a), fmt.Sprintf("%#v", *a),
	} {
		if strings.Contains(o, secret) {
			t.Errorf("secret leaked: %q", o)
		}
	}
}

func Test_ValidateEngineBaseURL_PureRules(t *testing.T) {
	restore := SetEngineResolverForTest(stubResolver("93.184.216.34"))
	defer restore()

	long := "https://" + strings.Repeat("a", 60) + "." + strings.Repeat("b", 60) + ".example.com/" + strings.Repeat("p", 400)

	tests := []struct {
		name   string
		raw    string
		expect error
	}{
		{"ok_simple", "https://llm.example.com", nil},
		{"ok_path", "https://llm.example.com/v1", nil},
		{"ok_port", "https://llm.example.com:8443/v1", nil},
		{"ok_port_max", "https://llm.example.com:65535", nil},
		{"ok_hyphen_label", "https://my-llm.ex-ample.com", nil},
		{"ok_ip_public", "https://93.184.216.34/v1", nil},
		{"ok_ipv6_public", "https://[2606:4700:4700::1111]:8443/v1", nil},

		{"empty", "", ErrInvalidEngineBaseURL},
		{"too_long", long, ErrInvalidEngineBaseURL},
		{"http", "http://llm.example.com", ErrInvalidEngineBaseURL},
		{"upper_scheme", "HTTPS://llm.example.com", ErrInvalidEngineBaseURL},
		{"no_scheme", "llm.example.com", ErrInvalidEngineBaseURL},
		{"userinfo", "https://user:pw@llm.example.com", ErrInvalidEngineBaseURL},
		{"userinfo_user_only", "https://user@llm.example.com", ErrInvalidEngineBaseURL},
		{"query", "https://llm.example.com/v1?a=b", ErrInvalidEngineBaseURL},
		{"query_empty", "https://llm.example.com/v1?", ErrInvalidEngineBaseURL},
		{"fragment", "https://llm.example.com/v1#x", ErrInvalidEngineBaseURL},
		{"fragment_empty", "https://llm.example.com/#", ErrInvalidEngineBaseURL},
		{"space", "https://llm.example.com/a b", ErrInvalidEngineBaseURL},
		{"leading_space", " https://llm.example.com", ErrInvalidEngineBaseURL},
		{"trailing_space", "https://llm.example.com ", ErrInvalidEngineBaseURL},
		{"newline", "https://llm.example.com/\n", ErrInvalidEngineBaseURL},
		{"non_ascii", "https://llm.example.com/é", ErrInvalidEngineBaseURL},
		{"non_ascii_host", "https://llm.exämple.com", ErrInvalidEngineBaseURL},
		{"opaque", "https:llm.example.com", ErrInvalidEngineBaseURL},
		{"empty_host", "https:///v1", ErrInvalidEngineBaseURL},
		{"port_zero", "https://llm.example.com:0", ErrInvalidEngineBaseURL},
		{"port_too_big", "https://llm.example.com:65536", ErrInvalidEngineBaseURL},
		{"port_not_numeric", "https://llm.example.com:abc", ErrInvalidEngineBaseURL},
		{"port_empty", "https://llm.example.com:/v1", ErrInvalidEngineBaseURL},
		{"single_label", "https://localhost", ErrInvalidEngineBaseURL},
		{"single_label_port", "https://intranet:8443", ErrInvalidEngineBaseURL},
		{"trailing_dot", "https://llm.example.com./v1", ErrInvalidEngineBaseURL},
		{"empty_label", "https://llm..example.com", ErrInvalidEngineBaseURL},
		{"leading_dot", "https://.example.com", ErrInvalidEngineBaseURL},
		{"label_leading_hyphen", "https://-a.example.com", ErrInvalidEngineBaseURL},
		{"label_trailing_hyphen", "https://a-.example.com", ErrInvalidEngineBaseURL},
		{"underscore", "https://a_b.example.com", ErrInvalidEngineBaseURL},
		{"percent_host", "https://a%41.example.com", ErrInvalidEngineBaseURL},
		{"ipv6_zone", "https://[fe80::1%25eth0]", ErrInvalidEngineBaseURL},
		{"backslash_host", "https://llm.example.com\\@evil.com", ErrInvalidEngineBaseURL},
		{"backslash_in_path", "https://llm.example.com/v1\\x", ErrInvalidEngineBaseURL},
		{"label_too_long", "https://" + strings.Repeat("a", 64) + ".example.com", ErrInvalidEngineBaseURL},

		// literal addresses are checked against the table immediately
		{"ip_loopback", "https://127.0.0.1", ErrEngineBaseURLRejected},
		{"ip_private", "https://10.0.0.5:8443", ErrEngineBaseURLRejected},
		{"ip_metadata", "https://169.254.169.254/latest", ErrEngineBaseURLRejected},
		{"ipv6_loopback", "https://[::1]", ErrEngineBaseURLRejected},
		{"ipv6_mapped_private", "https://[::ffff:10.0.0.1]", ErrEngineBaseURLRejected},
		{"ipv6_unique_local", "https://[fd00::1]", ErrEngineBaseURLRejected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEngineBaseURL(context.Background(), tt.raw)
			if !sameError(err, tt.expect) {
				t.Fatalf("expect: %v, got: %v", tt.expect, err)
			}
			if err != nil && strings.Contains(err.Error(), "example.com") {
				t.Errorf("the error echoes the url: %v", err)
			}
		})
	}
}

func Test_ValidateEngineBaseURL_Resolution(t *testing.T) {
	tests := []struct {
		name   string
		answer func(ctx context.Context, host string) ([]netip.Addr, error)
		expect error
	}{
		{"public", stubResolver("93.184.216.34"), nil},
		{"public_v6", stubResolver("2606:4700:4700::1111"), nil},
		{"two_public", stubResolver("93.184.216.34", "8.8.8.8"), nil},
		{"private", stubResolver("10.0.0.1"), ErrEngineBaseURLRejected},
		{"loopback", stubResolver("127.0.0.1"), ErrEngineBaseURLRejected},
		{"metadata", stubResolver("169.254.169.254"), ErrEngineBaseURLRejected},
		{"public_then_private", stubResolver("93.184.216.34", "10.0.0.1"), ErrEngineBaseURLRejected},
		{"private_then_public", stubResolver("10.0.0.1", "93.184.216.34"), ErrEngineBaseURLRejected},
		{"public_then_mapped_private", stubResolver("93.184.216.34", "::ffff:192.168.0.1"), ErrEngineBaseURLRejected},
		{"public_then_zone_loopback", stubResolver("93.184.216.34", "::1%lo"), ErrEngineBaseURLRejected},
		{"no_answer", stubResolver(), ErrEngineBaseURLRejected},
		{"error", func(ctx context.Context, host string) ([]netip.Addr, error) {
			return nil, errors.New("lookup failed for llm.example.com")
		}, ErrEngineBaseURLRejected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := SetEngineResolverForTest(tt.answer)
			defer restore()

			err := ValidateEngineBaseURL(context.Background(), "https://llm.example.com/v1")
			if !sameError(err, tt.expect) {
				t.Fatalf("expect: %v, got: %v", tt.expect, err)
			}
			if err != nil && (strings.Contains(err.Error(), "example.com") || strings.Contains(err.Error(), "lookup failed")) {
				t.Errorf("the error carries the url or the resolver text: %v", err)
			}
		})
	}
}

func Test_ValidateEngineBaseURL_RejectionsShareOneMessage(t *testing.T) {
	restore := SetEngineResolverForTest(stubResolver("10.0.0.1"))
	defer restore()
	errPrivate := ValidateEngineBaseURL(context.Background(), "https://llm.example.com")

	restore2 := SetEngineResolverForTest(stubResolver())
	defer restore2()
	errNone := ValidateEngineBaseURL(context.Background(), "https://llm.example.com")

	errLiteral := ValidateEngineBaseURL(context.Background(), "https://10.0.0.1")

	if errPrivate == nil || errNone == nil || errLiteral == nil {
		t.Fatalf("all three must fail. %v %v %v", errPrivate, errNone, errLiteral)
	}
	if errPrivate.Error() != errNone.Error() || errPrivate.Error() != errLiteral.Error() {
		t.Errorf("the rejection text must not reveal the cause. %q %q %q", errPrivate, errNone, errLiteral)
	}
}

func Test_ValidateEngineBaseURL_AbsoluteNameLookup(t *testing.T) {
	var got []string
	restore := SetEngineResolverForTest(func(ctx context.Context, host string) ([]netip.Addr, error) {
		got = append(got, host)
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	})
	defer restore()

	if err := ValidateEngineBaseURL(context.Background(), "https://LLM.Example.com:8443/v1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "LLM.Example.com." {
		t.Errorf("expect one lookup of the absolute name. got: %v", got)
	}

	got = nil
	if err := ValidateEngineBaseURL(context.Background(), "https://93.184.216.34/v1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("an ip literal must never be looked up. got: %v", got)
	}

	// legacy ipv4 spellings are names here and are looked up as absolute names
	got = nil
	restore2 := SetEngineResolverForTest(func(ctx context.Context, host string) ([]netip.Addr, error) {
		got = append(got, host)
		return nil, errors.New("no such host")
	})
	defer restore2()
	if err := ValidateEngineBaseURL(context.Background(), "https://127.1"); !errors.Is(err, ErrEngineBaseURLRejected) {
		t.Errorf("expect rejected. got: %v", err)
	}
	if len(got) != 1 || got[0] != "127.1." {
		t.Errorf("expect the absolute name. got: %v", got)
	}
}

func Test_ValidateEngineBaseURL_Deadline(t *testing.T) {
	restore := SetEngineResolverForTest(func(ctx context.Context, host string) ([]netip.Addr, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
			// the validator did not apply its own deadline
			return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
		}
	})
	defer restore()

	start := time.Now()
	err := ValidateEngineBaseURL(context.Background(), "https://slow.example.com")
	elapsed := time.Since(start)

	if !errors.Is(err, ErrEngineBaseURLRejected) {
		t.Fatalf("expect rejected. got: %v", err)
	}
	if elapsed < 900*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("expect a deadline of about one second. got: %v", elapsed)
	}
}

func Test_ValidateEngineBaseURL_ConcurrencyLimit(t *testing.T) {
	release := make(chan struct{})
	var started atomic.Int32
	restore := SetEngineResolverForTest(func(ctx context.Context, host string) ([]netip.Addr, error) {
		started.Add(1)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	})
	defer restore()

	var wg sync.WaitGroup
	results := make(chan error, 10)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- ValidateEngineBaseURL(context.Background(), "https://slow.example.com")
		}()
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for started.Load() < 4 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if started.Load() != 4 {
		t.Fatalf("expect four lookups to run. got: %d", started.Load())
	}

	// the fifth one is refused at once, without waiting, with the same error
	begin := time.Now()
	err := ValidateEngineBaseURL(context.Background(), "https://slow.example.com")
	if !errors.Is(err, ErrEngineBaseURLRejected) {
		t.Errorf("expect rejected over the limit. got: %v", err)
	}
	if time.Since(begin) > 200*time.Millisecond {
		t.Errorf("the fifth lookup must not wait. took: %v", time.Since(begin))
	}

	close(release)
	wg.Wait()
	close(results)

	// the permits are given back
	restore2 := SetEngineResolverForTest(stubResolver("93.184.216.34"))
	defer restore2()
	if err := ValidateEngineBaseURL(context.Background(), "https://slow.example.com"); err != nil {
		t.Errorf("permits must be given back. got: %v", err)
	}
}

func Test_ValidateEngineEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		model   EngineModel
		baseURL string
		expect  error
	}{
		{"custom_with_url", "customendpoint.my-model", "https://llm.example.com", nil},
		{"custom_without_url", "customendpoint.my-model", "", ErrEngineBaseURLRequired},
		{"openai_without_url", "openai.gpt-5", "", nil},
		{"openai_with_url", "openai.gpt-5", "https://llm.example.com", ErrEngineBaseURLNotAllowed},
		{"openrouter_custom_with_url", "openrouter.vendor/model", "https://llm.example.com", ErrEngineBaseURLNotAllowed},
		{"empty_model_without_url", "", "", nil},
		{"empty_model_with_url", "", "https://llm.example.com", ErrEngineBaseURLNotAllowed},
		{"upper_prefix_is_not_custom", "CustomEndpoint.x", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEngineEndpoint(tt.model, tt.baseURL)
			if !sameError(err, tt.expect) {
				t.Fatalf("expect: %v, got: %v", tt.expect, err)
			}
		})
	}
}

func Test_ValidateEngineBaseKey(t *testing.T) {
	str := func(s string) *string { return &s }

	tests := []struct {
		name   string
		model  EngineModel
		key    *string
		expect error
	}{
		{"custom_nil", "customendpoint.m", nil, nil},
		{"custom_empty", "customendpoint.m", str(""), nil},
		{"custom_value", "customendpoint.m", str("sk-abc_123.~+/="), nil},
		{"custom_max", "customendpoint.m", str(strings.Repeat("k", 512)), nil},
		{"custom_too_long", "customendpoint.m", str(strings.Repeat("k", 513)), ErrInvalidEngineBaseKey},
		{"custom_space", "customendpoint.m", str("sk abc"), ErrInvalidEngineBaseKey},
		{"custom_newline", "customendpoint.m", str("sk\nabc"), ErrInvalidEngineBaseKey},
		{"custom_crlf_injection", "customendpoint.m", str("sk\r\nX-Evil: 1"), ErrInvalidEngineBaseKey},
		{"custom_tab", "customendpoint.m", str("sk\tabc"), ErrInvalidEngineBaseKey},
		{"custom_non_ascii", "customendpoint.m", str("skê"), ErrInvalidEngineBaseKey},
		{"custom_del", "customendpoint.m", str("sk\x7f"), ErrInvalidEngineBaseKey},
		{"other_nil", "openai.gpt-5", nil, nil},
		{"other_empty_clears", "openai.gpt-5", str(""), nil},
		{"other_value", "openai.gpt-5", str("sk-abc"), ErrEngineBaseKeyNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEngineBaseKey(tt.model, tt.key)
			if !sameError(err, tt.expect) {
				t.Fatalf("expect: %v, got: %v", tt.expect, err)
			}
			if err != nil && tt.key != nil && *tt.key != "" && strings.Contains(err.Error(), *tt.key) {
				t.Errorf("the error echoes the key")
			}
		})
	}
}

func Test_ResolveEngine_CustomEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		model   EngineModel
		outcome Outcome
	}{
		{"ok", "customendpoint.my-model", OutcomeCustomEndpoint},
		{"ok_slash_colon", "customendpoint.org/model:7b-q4", OutcomeCustomEndpoint},
		{"ok_dots", "customendpoint.llama3.1", OutcomeCustomEndpoint},
		{"empty_id", "customendpoint.", OutcomeRejected},
		{"id_starts_with_dot", "customendpoint..x", OutcomeRejected},
		{"id_starts_with_hyphen", "customendpoint.-x", OutcomeRejected},
		{"space", "customendpoint.my model", OutcomeRejected},
		{"upper_prefix", "CustomEndpoint.x", OutcomeRejected},
		{"no_dot_after_prefix", "customendpoint", OutcomeRejected},
		{"too_long", EngineModel(CustomEndpointPrefix + strings.Repeat("a", 255)), OutcomeRejected},
		{"exact_limit", EngineModel(CustomEndpointPrefix + strings.Repeat("a", 255-len(CustomEndpointPrefix))), OutcomeCustomEndpoint},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, o := ResolveEngine(tt.model)
			if o != tt.outcome {
				t.Fatalf("outcome. expect: %v, got: %v", tt.outcome, o)
			}
			if o != OutcomeCustomEndpoint {
				return
			}
			if r.RunnerType != string(tt.model) || !r.BlankKey || r.Entry != nil || r.RequireKey {
				t.Errorf("unexpected resolved value: %+v", r)
			}
			if !IsValidEngineModel(tt.model) {
				t.Errorf("a custom endpoint model must be valid")
			}
		})
	}
}

func Test_OutcomeCustomEndpoint_IsLast(t *testing.T) {
	if OutcomeCustomEndpoint != OutcomeCustomOpenRouter+1 {
		t.Errorf("OutcomeCustomEndpoint must be appended last")
	}
	if OutcomeCustomOpenRouter != 3 {
		t.Errorf("iota values must stay stable")
	}
}

func Test_ValidateEngine_CustomEndpoint(t *testing.T) {
	if err := ValidateEngine("customendpoint.m", "", true); err != nil {
		t.Errorf("a custom endpoint model needs no legacy engine key. err: %v", err)
	}
	if err := ValidateEngine("customendpoint.", "", true); !errors.Is(err, ErrInvalidEngineModel) {
		t.Errorf("a malformed id must be rejected. err: %v", err)
	}
}

func Test_ConvertWebhookMessage_EndpointFields(t *testing.T) {
	a := &AI{
		EngineBaseURL:    "https://llm.example.com/v1",
		EngineBaseKeyEnc: []byte("cipher"),
		EngineBaseKey:    RedactedString("plain-secret"),
	}

	w := a.ConvertWebhookMessage()
	if w.EngineBaseURL != a.EngineBaseURL {
		t.Errorf("the url must be copied. got: %q", w.EngineBaseURL)
	}

	m, err := a.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(m), "plain-secret") || strings.Contains(string(m), "cipher") || strings.Contains(string(m), "Y2lwaGVy") {
		t.Errorf("the key material reached the webhook: %s", m)
	}
}

func Test_WebhookMessage_HasNoKeyField(t *testing.T) {
	typ := reflect.TypeOf(WebhookMessage{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.ToLower(typ.Field(i).Name)
		tag := strings.ToLower(typ.Field(i).Tag.Get("json"))
		if strings.Contains(name, "basekey") || strings.Contains(tag, "base_key") {
			t.Errorf("WebhookMessage must not expose the endpoint key: %s", typ.Field(i).Name)
		}
	}
}

func Test_AI_JSONRoundTripKeepsCiphertext(t *testing.T) {
	in := AI{EngineBaseURL: "https://llm.example.com", EngineBaseKeyEnc: []byte{0, 1, 2, 255}}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out AI
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(out.EngineBaseKeyEnc, in.EngineBaseKeyEnc) || out.EngineBaseURL != in.EngineBaseURL {
		t.Errorf("the cache form must keep the url and the ciphertext")
	}

	// the plaintext crosses the internal RPC as the plain string (documented limitation)
	in.EngineBaseKey = "plain"
	raw, _ = json.Marshal(in)
	if !strings.Contains(string(raw), `"engine_base_key":"plain"`) {
		t.Errorf("the plaintext must be carried by the get-by-id JSON: %s", raw)
	}
}

// Test_ValidateEngineBaseURLForSession_noInProcessLimit pins that a session start is not subject to
// the limit of concurrent lookups that protects the ai-manager RPC workers (a team with several
// custom endpoint members is checked at the same time), while ValidateEngineBaseURL keeps it.
func Test_ValidateEngineBaseURLForSession_noInProcessLimit(t *testing.T) {
	const parallel = maxEngineLookups + 4

	started := make(chan struct{}, parallel)
	release := make(chan struct{})
	restore := SetEngineResolverForTest(func(_ context.Context, _ string) ([]netip.Addr, error) {
		started <- struct{}{}
		<-release
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	})
	defer restore()

	run := func(f func(context.Context, string) error) []error {
		results := make(chan error, parallel)
		for i := 0; i < parallel; i++ {
			go func() { results <- f(context.Background(), "https://llm.example-endpoint.test/v1") }()
		}
		// every call that is allowed to look up is now waiting in the resolver
		time.Sleep(100 * time.Millisecond)
		close(release)
		out := []error{}
		for i := 0; i < parallel; i++ {
			out = append(out, <-results)
		}
		return out
	}

	countErr := func(errs []error) int {
		n := 0
		for _, e := range errs {
			if e != nil {
				n++
			}
		}
		return n
	}

	if n := countErr(run(ValidateEngineBaseURLForSession)); n != 0 {
		t.Errorf("session validation must not reject concurrent lookups, got %d errors", n)
	}

	// the limited variant rejects the calls over the limit (fresh release channel)
	release = make(chan struct{})
	for len(started) > 0 {
		<-started
	}
	if n := countErr(run(ValidateEngineBaseURL)); n != parallel-maxEngineLookups {
		t.Errorf("limited validation must reject the calls over the limit, got %d errors", n)
	}
}
