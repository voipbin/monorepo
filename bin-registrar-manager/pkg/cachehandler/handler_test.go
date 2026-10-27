package cachehandler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astcontact"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/trunk"
)

var errInjected = errors.New("injected command failure")

// failNthHook makes the n-th (1-based) command with the given name fail.
type failNthHook struct {
	name  string
	n     int
	count int
}

func (k *failNthHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	if cmd.Name() != k.name {
		return ctx, nil
	}
	k.count++
	if k.count == k.n {
		return ctx, errInjected
	}
	return ctx, nil
}

func (k *failNthHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	return nil
}

func (k *failNthHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (k *failNthHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	return nil
}

func setupHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h := &handler{
		Addr:  mr.Addr(),
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1}),
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	return h, mr
}

func strPtr(s string) *string {
	return &s
}

var (
	testExtensionID = uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	testCustomerID  = uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	testTrunkID     = uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
)

func newTestExtension() *extension.Extension {
	return &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         testExtensionID,
			CustomerID: testCustomerID,
		},
		Name:       "test extension",
		EndpointID: "endpoint-test@example.invalid",
		Extension:  "1001",
		Realm:      "example.invalid",
	}
}

func newTestTrunk() *trunk.Trunk {
	return &trunk.Trunk{
		Identity: commonidentity.Identity{
			ID:         testTrunkID,
			CustomerID: testCustomerID,
		},
		Name:       "test trunk",
		DomainName: "test-trunk",
		Username:   "trunk-user",
	}
}

func assertTTL(t *testing.T, mr *miniredis.Miniredis, key string, want time.Duration) {
	t.Helper()

	if !mr.Exists(key) {
		t.Errorf("key %q was not written", key)
		return
	}
	if got := mr.TTL(key); got != want {
		t.Errorf("TTL of %q = %v, want %v", key, got, want)
	}
}

func Test_NewHandler(t *testing.T) {
	res := NewHandler("127.0.0.1:6379", "secret", 3)

	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("NewHandler() returned %T, want *handler", res)
	}
	if h.Addr != "127.0.0.1:6379" || h.Password != "secret" || h.DB != 3 {
		t.Errorf("NewHandler() fields = %q/%q/%d", h.Addr, h.Password, h.DB)
	}
	if h.Cache == nil {
		t.Fatalf("NewHandler() did not create a redis client")
	}
	opts := h.Cache.Options()
	if opts.Addr != "127.0.0.1:6379" || opts.Password != "secret" || opts.DB != 3 {
		t.Errorf("redis options = %q/%q/%d", opts.Addr, opts.Password, opts.DB)
	}
	_ = h.Cache.Close()
}

func Test_Connect(t *testing.T) {
	h, mr := setupHandler(t)

	if err := h.Connect(); err != nil {
		t.Errorf("Connect() error = %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Connect() on a stopped server returned nil")
	}
}

func Test_AstEndpoint(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	e := &astendpoint.AstEndpoint{ID: strPtr("ep-1"), Transport: strPtr("transport-udp"), AORs: strPtr("ep-1")}
	if err := h.AstEndpointSet(ctx, e); err != nil {
		t.Fatalf("AstEndpointSet() error = %v", err)
	}
	assertTTL(t, mr, "ast_endpoint:ep-1", 3*time.Minute)

	got, err := h.AstEndpointGet(ctx, "ep-1")
	if err != nil {
		t.Fatalf("AstEndpointGet() error = %v", err)
	}
	if !reflect.DeepEqual(got, e) {
		t.Errorf("AstEndpointGet() = %+v, want %+v", got, e)
	}

	if err := h.AstEndpointDel(ctx, "ep-1"); err != nil {
		t.Errorf("AstEndpointDel() error = %v", err)
	}
	if mr.Exists("ast_endpoint:ep-1") {
		t.Errorf("AstEndpointDel() left the key")
	}
	if _, err := h.AstEndpointGet(ctx, "ep-1"); !errors.Is(err, redis.Nil) {
		t.Errorf("AstEndpointGet() after delete error = %v, want redis.Nil", err)
	}
	if err := h.AstEndpointDel(ctx, "ep-missing"); err != nil {
		t.Errorf("AstEndpointDel() of a missing key error = %v", err)
	}
}

func Test_AstAuth(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	e := &astauth.AstAuth{ID: strPtr("auth-1"), Realm: strPtr("example.invalid")}
	if err := h.AstAuthSet(ctx, e); err != nil {
		t.Fatalf("AstAuthSet() error = %v", err)
	}
	assertTTL(t, mr, "ast_auth:auth-1", 3*time.Minute)

	got, err := h.AstAuthGet(ctx, "auth-1")
	if err != nil {
		t.Fatalf("AstAuthGet() error = %v", err)
	}
	if !reflect.DeepEqual(got, e) {
		t.Errorf("AstAuthGet() = %+v, want %+v", got, e)
	}

	if err := h.AstAuthDel(ctx, "auth-1"); err != nil {
		t.Errorf("AstAuthDel() error = %v", err)
	}
	if _, err := h.AstAuthGet(ctx, "auth-1"); !errors.Is(err, redis.Nil) {
		t.Errorf("AstAuthGet() after delete error = %v, want redis.Nil", err)
	}
}

func Test_AstAOR(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	e := &astaor.AstAOR{ID: strPtr("aor-1"), VoicemailExtension: strPtr("1001")}
	if err := h.AstAORSet(ctx, e); err != nil {
		t.Fatalf("AstAORSet() error = %v", err)
	}
	assertTTL(t, mr, "ast_aor:aor-1", 3*time.Minute)

	got, err := h.AstAORGet(ctx, "aor-1")
	if err != nil {
		t.Fatalf("AstAORGet() error = %v", err)
	}
	if !reflect.DeepEqual(got, e) {
		t.Errorf("AstAORGet() = %+v, want %+v", got, e)
	}

	if err := h.AstAORDel(ctx, "aor-1"); err != nil {
		t.Errorf("AstAORDel() error = %v", err)
	}
	if _, err := h.AstAORGet(ctx, "aor-1"); !errors.Is(err, redis.Nil) {
		t.Errorf("AstAORGet() after delete error = %v, want redis.Nil", err)
	}
}

func Test_AstContacts(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	contacts := []*astcontact.AstContact{
		{ID: "contact-a", URI: "sip:a@192.0.2.1:5060", Endpoint: "ep-1"},
		{ID: "contact-b", URI: "sip:b@192.0.2.2:5060", Endpoint: "ep-1"},
		{ID: "contact-c", URI: "sip:c@192.0.2.3:5060", Endpoint: "ep-1"},
	}
	if err := h.AstContactsSet(ctx, "ep-1", contacts); err != nil {
		t.Fatalf("AstContactsSet() error = %v", err)
	}
	assertTTL(t, mr, "ast_contacts:ep-1", 3*time.Minute)

	got, err := h.AstContactsGet(ctx, "ep-1")
	if err != nil {
		t.Fatalf("AstContactsGet() error = %v", err)
	}
	// every element must be a distinct restored contact, not the last one repeated
	if !reflect.DeepEqual(got, contacts) {
		t.Errorf("AstContactsGet() = %+v, want %+v", got, contacts)
	}

	if err := h.AstContactsDel(ctx, "ep-1"); err != nil {
		t.Errorf("AstContactsDel() error = %v", err)
	}
	if _, err := h.AstContactsGet(ctx, "ep-1"); !errors.Is(err, redis.Nil) {
		t.Errorf("AstContactsGet() after delete error = %v, want redis.Nil", err)
	}
}

func Test_AstContacts_nilSlice(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	if err := h.AstContactsSet(ctx, "ep-nil", nil); err != nil {
		t.Fatalf("AstContactsSet() error = %v", err)
	}
	if v, _ := mr.Get("ast_contacts:ep-nil"); v != "null" {
		t.Errorf("stored value = %q, want null", v)
	}

	got, err := h.AstContactsGet(ctx, "ep-nil")
	if err != nil {
		t.Fatalf("AstContactsGet() error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("AstContactsGet() = %#v, want a non-nil empty slice", got)
	}
}

func Test_Extension(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	e := newTestExtension()
	if err := h.ExtensionSet(ctx, e); err != nil {
		t.Fatalf("ExtensionSet() error = %v", err)
	}

	keyID := "registrar:extension:11111111-1111-1111-1111-111111111111"
	keyEndpoint := "registrar:extension_endpoint_id:endpoint-test@example.invalid"
	keyExt := "registrar:customer_id_extension:22222222-2222-2222-2222-222222222222:1001"
	for _, key := range []string{keyID, keyEndpoint, keyExt} {
		assertTTL(t, mr, key, 24*time.Hour)
	}

	byID, err := h.ExtensionGet(ctx, testExtensionID)
	if err != nil {
		t.Fatalf("ExtensionGet() error = %v", err)
	}
	if !reflect.DeepEqual(byID, e) {
		t.Errorf("ExtensionGet() = %+v, want %+v", byID, e)
	}

	byEndpoint, err := h.ExtensionGetByEndpointID(ctx, e.EndpointID)
	if err != nil {
		t.Fatalf("ExtensionGetByEndpointID() error = %v", err)
	}
	if !reflect.DeepEqual(byEndpoint, e) {
		t.Errorf("ExtensionGetByEndpointID() = %+v, want %+v", byEndpoint, e)
	}

	byExt, err := h.ExtensionGetByCustomerIDANDExtension(ctx, testCustomerID, "1001")
	if err != nil {
		t.Fatalf("ExtensionGetByCustomerIDANDExtension() error = %v", err)
	}
	if !reflect.DeepEqual(byExt, e) {
		t.Errorf("ExtensionGetByCustomerIDANDExtension() = %+v, want %+v", byExt, e)
	}

	// a different customer must not see the extension
	other := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	if _, err := h.ExtensionGetByCustomerIDANDExtension(ctx, other, "1001"); !errors.Is(err, redis.Nil) {
		t.Errorf("lookup with another customer error = %v, want redis.Nil", err)
	}
}

func Test_Extension_getErrors(t *testing.T) {
	h, _ := setupHandler(t)
	ctx := context.Background()

	if _, err := h.ExtensionGet(ctx, testExtensionID); !errors.Is(err, redis.Nil) {
		t.Errorf("ExtensionGet() miss error = %v, want redis.Nil", err)
	}
	if _, err := h.ExtensionGetByEndpointID(ctx, "missing"); !errors.Is(err, redis.Nil) {
		t.Errorf("ExtensionGetByEndpointID() miss error = %v, want redis.Nil", err)
	}
	if _, err := h.ExtensionGetByCustomerIDANDExtension(ctx, testCustomerID, "9999"); !errors.Is(err, redis.Nil) {
		t.Errorf("ExtensionGetByCustomerIDANDExtension() miss error = %v, want redis.Nil", err)
	}
}

func Test_Extension_setPartialFailure(t *testing.T) {
	tests := []struct {
		name        string
		failAt      int
		wantWritten []string
	}{
		{
			name:        "first key fails",
			failAt:      1,
			wantWritten: nil,
		},
		{
			name:   "second key fails",
			failAt: 2,
			wantWritten: []string{
				"registrar:extension:11111111-1111-1111-1111-111111111111",
			},
		},
		{
			name:   "third key fails",
			failAt: 3,
			wantWritten: []string{
				"registrar:extension:11111111-1111-1111-1111-111111111111",
				"registrar:extension_endpoint_id:endpoint-test@example.invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := setupHandler(t)
			h.Cache.AddHook(&failNthHook{name: "set", n: tt.failAt})

			err := h.ExtensionSet(context.Background(), newTestExtension())
			if !errors.Is(err, errInjected) {
				t.Fatalf("ExtensionSet() error = %v, want injected failure", err)
			}

			keys := mr.Keys()
			if len(keys) != len(tt.wantWritten) {
				t.Fatalf("written keys = %v, want %v", keys, tt.wantWritten)
			}
			for _, want := range tt.wantWritten {
				if !mr.Exists(want) {
					t.Errorf("key %q was not written before the failure", want)
				}
			}
		})
	}
}

func Test_Trunk(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	e := newTestTrunk()
	if err := h.TrunkSet(ctx, e); err != nil {
		t.Fatalf("TrunkSet() error = %v", err)
	}

	keyID := "registrar:trunk:33333333-3333-3333-3333-333333333333"
	keyName := "registrar:trunk_domain_name:test-trunk"
	assertTTL(t, mr, keyID, 24*time.Hour)
	assertTTL(t, mr, keyName, 24*time.Hour)

	byID, err := h.TrunkGet(ctx, testTrunkID)
	if err != nil {
		t.Fatalf("TrunkGet() error = %v", err)
	}
	if !reflect.DeepEqual(byID, e) {
		t.Errorf("TrunkGet() = %+v, want %+v", byID, e)
	}

	byName, err := h.TrunkGetByDomainName(ctx, "test-trunk")
	if err != nil {
		t.Fatalf("TrunkGetByDomainName() error = %v", err)
	}
	if !reflect.DeepEqual(byName, e) {
		t.Errorf("TrunkGetByDomainName() = %+v, want %+v", byName, e)
	}

	if err := h.TrunkDel(ctx, testTrunkID, "test-trunk"); err != nil {
		t.Fatalf("TrunkDel() error = %v", err)
	}
	if mr.Exists(keyID) || mr.Exists(keyName) {
		t.Errorf("TrunkDel() left keys: %v", mr.Keys())
	}
	if _, err := h.TrunkGet(ctx, testTrunkID); !errors.Is(err, redis.Nil) {
		t.Errorf("TrunkGet() after delete error = %v, want redis.Nil", err)
	}
	if _, err := h.TrunkGetByDomainName(ctx, "test-trunk"); !errors.Is(err, redis.Nil) {
		t.Errorf("TrunkGetByDomainName() after delete error = %v, want redis.Nil", err)
	}
	if err := h.TrunkDel(ctx, testTrunkID, "test-trunk"); err != nil {
		t.Errorf("TrunkDel() of missing keys error = %v", err)
	}
}

func Test_Trunk_setPartialFailure(t *testing.T) {
	h, mr := setupHandler(t)
	h.Cache.AddHook(&failNthHook{name: "set", n: 2})

	err := h.TrunkSet(context.Background(), newTestTrunk())
	if !errors.Is(err, errInjected) {
		t.Fatalf("TrunkSet() error = %v, want injected failure", err)
	}
	if !mr.Exists("registrar:trunk:33333333-3333-3333-3333-333333333333") {
		t.Errorf("the id key should be written before the domain name key fails")
	}
	if mr.Exists("registrar:trunk_domain_name:test-trunk") {
		t.Errorf("the domain name key must not exist after its write failed")
	}
}

func Test_Trunk_delPartialFailure(t *testing.T) {
	tests := []struct {
		name        string
		failAt      int
		wantIDKey   bool
		wantNameKey bool
	}{
		{name: "first delete fails", failAt: 1, wantIDKey: true, wantNameKey: true},
		{name: "second delete fails", failAt: 2, wantIDKey: false, wantNameKey: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := setupHandler(t)
			ctx := context.Background()

			if err := h.TrunkSet(ctx, newTestTrunk()); err != nil {
				t.Fatalf("TrunkSet() error = %v", err)
			}
			h.Cache.AddHook(&failNthHook{name: "del", n: tt.failAt})

			err := h.TrunkDel(ctx, testTrunkID, "test-trunk")
			if !errors.Is(err, errInjected) {
				t.Fatalf("TrunkDel() error = %v, want injected failure", err)
			}
			if got := mr.Exists("registrar:trunk:33333333-3333-3333-3333-333333333333"); got != tt.wantIDKey {
				t.Errorf("id key exists = %v, want %v", got, tt.wantIDKey)
			}
			if got := mr.Exists("registrar:trunk_domain_name:test-trunk"); got != tt.wantNameKey {
				t.Errorf("domain name key exists = %v, want %v", got, tt.wantNameKey)
			}
		})
	}
}

func Test_CustomerDomain(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	cd := &customerdomain.CustomerDomain{
		CustomerID:  testCustomerID,
		DomainLabel: "ab12",
		Realm:       "ab12.example.invalid",
	}
	if err := h.CustomerDomainSet(ctx, cd); err != nil {
		t.Fatalf("CustomerDomainSet() error = %v", err)
	}
	assertTTL(t, mr, "registrar:customer_domain_realm:ab12.example.invalid", 24*time.Hour)

	got, err := h.CustomerDomainGetByRealm(ctx, "ab12.example.invalid")
	if err != nil {
		t.Fatalf("CustomerDomainGetByRealm() error = %v", err)
	}
	if !reflect.DeepEqual(got, cd) {
		t.Errorf("CustomerDomainGetByRealm() = %+v, want %+v", got, cd)
	}

	if err := h.CustomerDomainDelByRealm(ctx, "ab12.example.invalid"); err != nil {
		t.Errorf("CustomerDomainDelByRealm() error = %v", err)
	}
	if _, err := h.CustomerDomainGetByRealm(ctx, "ab12.example.invalid"); !errors.Is(err, redis.Nil) {
		t.Errorf("CustomerDomainGetByRealm() after delete error = %v, want redis.Nil", err)
	}
}

func Test_getErrors_corruptedValue(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	tests := []struct {
		name string
		key  string
		val  string
		call func() error
	}{
		{
			name: "endpoint not json",
			key:  "ast_endpoint:bad",
			val:  "{not json",
			call: func() error { _, err := h.AstEndpointGet(ctx, "bad"); return err },
		},
		{
			name: "endpoint wrong type",
			key:  "ast_endpoint:bad2",
			val:  "[1,2]",
			call: func() error { _, err := h.AstEndpointGet(ctx, "bad2"); return err },
		},
		{
			name: "auth not json",
			key:  "ast_auth:bad",
			val:  "{not json",
			call: func() error { _, err := h.AstAuthGet(ctx, "bad"); return err },
		},
		{
			name: "aor not json",
			key:  "ast_aor:bad",
			val:  "{not json",
			call: func() error { _, err := h.AstAORGet(ctx, "bad"); return err },
		},
		{
			name: "contacts object instead of list",
			key:  "ast_contacts:bad",
			val:  `{"id":"x"}`,
			call: func() error { _, err := h.AstContactsGet(ctx, "bad"); return err },
		},
		{
			name: "extension array instead of object",
			key:  "registrar:extension:11111111-1111-1111-1111-111111111111",
			val:  "[]",
			call: func() error { _, err := h.ExtensionGet(ctx, testExtensionID); return err },
		},
		{
			name: "extension by endpoint not json",
			key:  "registrar:extension_endpoint_id:bad",
			val:  "{not json",
			call: func() error { _, err := h.ExtensionGetByEndpointID(ctx, "bad"); return err },
		},
		{
			name: "extension by customer and number not json",
			key:  "registrar:customer_id_extension:22222222-2222-2222-2222-222222222222:7",
			val:  "{not json",
			call: func() error {
				_, err := h.ExtensionGetByCustomerIDANDExtension(ctx, testCustomerID, "7")
				return err
			},
		},
		{
			name: "customer domain not json",
			key:  "registrar:customer_domain_realm:bad",
			val:  "{not json",
			call: func() error { _, err := h.CustomerDomainGetByRealm(ctx, "bad"); return err },
		},
		{
			name: "trunk array instead of object",
			key:  "registrar:trunk:33333333-3333-3333-3333-333333333333",
			val:  "[]",
			call: func() error { _, err := h.TrunkGet(ctx, testTrunkID); return err },
		},
		{
			name: "trunk by domain name not json",
			key:  "registrar:trunk_domain_name:bad",
			val:  "{not json",
			call: func() error { _, err := h.TrunkGetByDomainName(ctx, "bad"); return err },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mr.Set(tt.key, tt.val); err != nil {
				t.Fatalf("could not seed the key: %v", err)
			}

			err := tt.call()
			if err == nil {
				t.Fatalf("expected an unmarshal error, got nil")
			}
			if errors.Is(err, redis.Nil) {
				t.Errorf("error = redis.Nil, want an unmarshal error")
			}
			var syntaxErr *json.SyntaxError
			var typeErr *json.UnmarshalTypeError
			if !errors.As(err, &syntaxErr) && !errors.As(err, &typeErr) {
				t.Errorf("error = %T %v, want a json syntax or type error", err, err)
			}
		})
	}
}

func Test_serverDown(t *testing.T) {
	h, mr := setupHandler(t)
	ctx := context.Background()

	e := newTestExtension()
	if err := h.ExtensionSet(ctx, e); err != nil {
		t.Fatalf("ExtensionSet() error = %v", err)
	}
	mr.Close()

	if err := h.ExtensionSet(ctx, e); err == nil {
		t.Errorf("ExtensionSet() on a stopped server returned nil")
	}
	if _, err := h.ExtensionGet(ctx, testExtensionID); err == nil || errors.Is(err, redis.Nil) {
		t.Errorf("ExtensionGet() on a stopped server error = %v, want a connection error", err)
	}
	if err := h.AstContactsDel(ctx, "ep-1"); err == nil {
		t.Errorf("AstContactsDel() on a stopped server returned nil")
	}
	if err := h.TrunkDel(ctx, testTrunkID, "test-trunk"); err == nil {
		t.Errorf("TrunkDel() on a stopped server returned nil")
	}
}

func Test_setErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(h *handler) error
	}{
		{"AstEndpointSet", func(h *handler) error { return h.AstEndpointSet(ctx, &astendpoint.AstEndpoint{ID: strPtr("a")}) }},
		{"AstAuthSet", func(h *handler) error { return h.AstAuthSet(ctx, &astauth.AstAuth{ID: strPtr("a")}) }},
		{"AstAORSet", func(h *handler) error { return h.AstAORSet(ctx, &astaor.AstAOR{ID: strPtr("a")}) }},
		{"AstContactsSet", func(h *handler) error { return h.AstContactsSet(ctx, "a", nil) }},
		{"CustomerDomainSet", func(h *handler) error {
			return h.CustomerDomainSet(ctx, &customerdomain.CustomerDomain{Realm: "a.example.invalid"})
		}},
		{"TrunkSet", func(h *handler) error { return h.TrunkSet(ctx, newTestTrunk()) }},
		{"ExtensionSet", func(h *handler) error { return h.ExtensionSet(ctx, newTestExtension()) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := setupHandler(t)
			h.Cache.AddHook(&failNthHook{name: "set", n: 1})

			if err := tt.call(h); !errors.Is(err, errInjected) {
				t.Errorf("error = %v, want injected failure", err)
			}
			if len(mr.Keys()) != 0 {
				t.Errorf("keys written despite the failure: %v", mr.Keys())
			}
		})
	}
}

func Test_delErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(h *handler) error
	}{
		{"AstEndpointDel", func(h *handler) error { return h.AstEndpointDel(ctx, "a") }},
		{"AstAuthDel", func(h *handler) error { return h.AstAuthDel(ctx, "a") }},
		{"AstAORDel", func(h *handler) error { return h.AstAORDel(ctx, "a") }},
		{"AstContactsDel", func(h *handler) error { return h.AstContactsDel(ctx, "a") }},
		{"CustomerDomainDelByRealm", func(h *handler) error { return h.CustomerDomainDelByRealm(ctx, "a") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := setupHandler(t)
			h.Cache.AddHook(&failNthHook{name: "del", n: 1})

			if err := tt.call(h); !errors.Is(err, errInjected) {
				t.Errorf("error = %v, want injected failure", err)
			}
		})
	}
}

func Test_setSerializeWithExpiration_unencodableValue(t *testing.T) {
	h, mr := setupHandler(t)

	err := h.setSerializeWithExpiration(context.Background(), "case_key", map[string]any{"case": make(chan int)}, time.Minute)
	if err == nil {
		t.Fatalf("Wrong match. expect an encoding error, got: nil")
	}
	if mr.Exists("case_key") {
		t.Errorf("Wrong match. nothing must be stored when the value cannot be encoded")
	}
}
