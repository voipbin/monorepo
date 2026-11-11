package requesthandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

const (
	testConfigTarget = "bin-manager.email-manager.request"
	testConfigKeyRaw = `{"key":"mailgun.api_key","kind":"secret","type":"string","description":"The key.","source":"db","has_secret":true,"env_name":"MAILGUN_API_KEY","env_shadowed":false,"key_version":1,"revision":3}`
	testConfigSecret = "test-secret-not-real"
)

var testConfigActor = uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")

func testConfigKey() *configrpc.Key {
	return &configrpc.Key{
		Key: "mailgun.api_key", Kind: "secret", Type: "string", Description: "The key.", Source: "db",
		HasSecret: true, EnvName: "MAILGUN_API_KEY", KeyVersion: 1, Revision: 3,
	}
}

func Test_ConfigV1ServiceGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, &sock.Request{
		URI:      "/v1/configs",
		Method:   sock.RequestMethodGet,
		DataType: ContentTypeJSON,
	}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`{"secret_writable":true,"keys":[` + testConfigKeyRaw + `]}`)}, nil)

	res, err := reqHandler.ConfigV1ServiceGet(context.Background(), "email")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := &configrpc.Service{Service: "email", SecretWritable: true, Keys: []configrpc.Key{*testConfigKey()}}
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_ConfigV1KeySecretSet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, &sock.Request{
		URI:      "/v1/configs/mailgun.api_key/secret",
		Method:   sock.RequestMethodPut,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"actor_id":"550e8400-e29b-41d4-a716-446655440000","secret":"` + testConfigSecret + `","revision":3}`),
	}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testConfigKeyRaw)}, nil)

	res, err := reqHandler.ConfigV1KeySecretSet(context.Background(), "email", "mailgun.api_key", testConfigActor, testConfigSecret, 3)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(testConfigKey(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testConfigKey(), res)
	}
}

// A revision of zero is sent: it means "no stored value", not "not given".
func Test_ConfigV1KeySecretSet_revisionZeroIsSent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, &sock.Request{
		URI:      "/v1/configs/mailgun.api_key/secret",
		Method:   sock.RequestMethodPut,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"actor_id":"550e8400-e29b-41d4-a716-446655440000","secret":"` + testConfigSecret + `","revision":0}`),
	}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testConfigKeyRaw)}, nil)

	if _, err := reqHandler.ConfigV1KeySecretSet(context.Background(), "email", "mailgun.api_key", testConfigActor, testConfigSecret, 0); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_ConfigV1KeyClear(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, &sock.Request{
		URI:      "/v1/configs/mailgun.api_key",
		Method:   sock.RequestMethodDelete,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"actor_id":"550e8400-e29b-41d4-a716-446655440000","revision":3}`),
	}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testConfigKeyRaw)}, nil)

	res, err := reqHandler.ConfigV1KeyClear(context.Background(), "email", "mailgun.api_key", testConfigActor, 3)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(testConfigKey(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testConfigKey(), res)
	}
}

func Test_ConfigV1KeyHistoryList(t *testing.T) {
	tests := []struct {
		name string

		pageToken string
		pageSize  int

		expectURI string
	}{
		{name: "no paging", expectURI: "/v1/configs/mailgun.api_key/histories"},
		{name: "size only", pageSize: 5, expectURI: "/v1/configs/mailgun.api_key/histories?page_size=5"},
		{name: "token and size", pageToken: "abc-_DEF", pageSize: 20, expectURI: "/v1/configs/mailgun.api_key/histories?page_size=20&page_token=abc-_DEF"},
		{name: "token is escaped", pageToken: "a&b=c d", expectURI: "/v1/configs/mailgun.api_key/histories?page_token=a%26b%3Dc+d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			id := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
			body := `{"result":[{"id":"` + id + `","key":"mailgun.api_key","action":"set","is_secret":true,"old_key_version":0,"new_key_version":1,"actor_id":"550e8400-e29b-41d4-a716-446655440000","source":"api","tm_create":"2026-03-04T05:06:07Z"}],"next_page_token":"next"}`
			mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, &sock.Request{
				URI:      tt.expectURI,
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(body)}, nil)

			res, err := reqHandler.ConfigV1KeyHistoryList(context.Background(), "email", "mailgun.api_key", tt.pageToken, tt.pageSize)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(res.Result) != 1 || res.NextPageToken != "next" || res.Result[0].ID.String() != id || !res.Result[0].IsSecret {
				t.Errorf("Wrong match. got: %+v", res)
			}
		})
	}
}

// The service ai is sent to the queue of the ai manager and to no other queue, for all four calls,
// with the same uri, method and body as the other services.
func Test_config_aiService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	const target = "bin-manager.ai-manager.request"
	ok := &sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testConfigKeyRaw)}

	tests := []struct {
		name   string
		expect *sock.Request
		resp   *sock.Response
		call   func(r *requestHandler) error
	}{
		{
			"service",
			&sock.Request{URI: "/v1/configs", Method: sock.RequestMethodGet, DataType: ContentTypeJSON},
			&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`{"secret_writable":true,"keys":[` + testConfigKeyRaw + `]}`)},
			func(r *requestHandler) error {
				res, err := r.ConfigV1ServiceGet(context.Background(), "ai")
				if err == nil && res.Service != "ai" {
					return fmt.Errorf("wrong service: %q", res.Service)
				}
				return err
			},
		},
		{
			"secret",
			&sock.Request{
				URI:      "/v1/configs/llm.gemini.api_key/secret",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"actor_id":"` + testConfigActor.String() + `","secret":"` + testConfigSecret + `","revision":2}`),
			},
			ok,
			func(r *requestHandler) error {
				_, err := r.ConfigV1KeySecretSet(context.Background(), "ai", "llm.gemini.api_key", testConfigActor, testConfigSecret, 2)
				return err
			},
		},
		{
			"delete",
			&sock.Request{
				URI:      "/v1/configs/llm.gemini.api_key",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"actor_id":"` + testConfigActor.String() + `","revision":2}`),
			},
			ok,
			func(r *requestHandler) error {
				_, err := r.ConfigV1KeyClear(context.Background(), "ai", "llm.gemini.api_key", testConfigActor, 2)
				return err
			},
		},
		{
			"histories",
			&sock.Request{URI: "/v1/configs/llm.gemini.api_key/histories", Method: sock.RequestMethodGet, DataType: ContentTypeJSON},
			&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`{"result":[],"next_page_token":""}`)},
			func(r *requestHandler) error {
				_, err := r.ConfigV1KeyHistoryList(context.Background(), "ai", "llm.gemini.api_key", "", 0)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublish(gomock.Any(), target, tt.expect).Return(tt.resp, nil)

			if err := tt.call(&reqHandler); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// The label of the metrics is a fixed text per service, never a key name.
func Test_configQueue(t *testing.T) {
	tests := []struct {
		service     string
		expectQueue string
		expectLabel string
		expectErr   error
	}{
		{"email", "bin-manager.email-manager.request", "email/configs", nil},
		{"ai", "bin-manager.ai-manager.request", "ai/configs", nil},
		{"message", "bin-manager.message-manager.request", "message/configs", nil},
		{"tts", "", "", ErrConfigServiceUnknown},
		{"", "", "", ErrConfigServiceUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			queue, label, err := configQueue(tt.service)
			if !errors.Is(err, tt.expectErr) || string(queue) != tt.expectQueue || label != tt.expectLabel {
				t.Errorf("Wrong match. expect: %s %s %v, got: %s %s %v", tt.expectQueue, tt.expectLabel, tt.expectErr, queue, label, err)
			}
		})
	}
}

// The service name picks the queue, and a service without config support is an error that does
// not publish anything.
func Test_config_unknownService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}
	ctx := context.Background()

	for _, service := range []string{"", "nothing", "EMAIL", "AI", "MESSAGE", "ai/../x", "email/../x"} {
		if _, err := reqHandler.ConfigV1ServiceGet(ctx, service); !errors.Is(err, ErrConfigServiceUnknown) {
			t.Errorf("Wrong match. service %q expect: ErrConfigServiceUnknown, got: %v", service, err)
		}
		if _, err := reqHandler.ConfigV1KeySecretSet(ctx, service, "k", testConfigActor, testConfigSecret, 0); !errors.Is(err, ErrConfigServiceUnknown) {
			t.Errorf("Wrong match. service %q expect: ErrConfigServiceUnknown, got: %v", service, err)
		}
		if _, err := reqHandler.ConfigV1KeyClear(ctx, service, "k", testConfigActor, 0); !errors.Is(err, ErrConfigServiceUnknown) {
			t.Errorf("Wrong match. service %q expect: ErrConfigServiceUnknown, got: %v", service, err)
		}
		if _, err := reqHandler.ConfigV1KeyHistoryList(ctx, service, "k", "", 0); !errors.Is(err, ErrConfigServiceUnknown) {
			t.Errorf("Wrong match. service %q expect: ErrConfigServiceUnknown, got: %v", service, err)
		}
	}
}

// The key is part of the URI and is escaped, so a request value can not change the route.
func Test_config_keyIsEscaped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, &sock.Request{
		URI:      "/v1/configs/a%2Fsecret",
		Method:   sock.RequestMethodDelete,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"actor_id":"550e8400-e29b-41d4-a716-446655440000","revision":0}`),
	}).Return(&sock.Response{StatusCode: 404}, nil)

	if _, err := reqHandler.ConfigV1KeyClear(context.Background(), "email", "a/secret", testConfigActor, 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}

// The typed errors of the manager reach the caller as they are, with the reason.
func Test_config_typedErrorsPass(t *testing.T) {
	tests := []struct {
		name         string
		resp         *sock.Response
		expectReason string
	}{
		{"stale", mustResponse(t, cerrors.FailedPrecondition("email-manager", "REVISION_MISMATCH", "stale")), "REVISION_MISMATCH"},
		{"no master key", mustResponse(t, cerrors.Unavailable("email-manager", "ENCRYPTION_KEY_NOT_CONFIGURED", "no key")), "ENCRYPTION_KEY_NOT_CONFIGURED"},
		{"unknown key", mustResponse(t, cerrors.NotFound("email-manager", "CONFIG_KEY_NOT_FOUND", "no key")), "CONFIG_KEY_NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}
			mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, gomock.Any()).Return(tt.resp, nil)

			_, err := reqHandler.ConfigV1KeySecretSet(context.Background(), "email", "mailgun.api_key", testConfigActor, testConfigSecret, 1)
			var ve *cerrors.VoipbinError
			if !errors.As(err, &ve) || ve.Reason != tt.expectReason {
				t.Errorf("Wrong match. expect: a typed error %s, got: %v", tt.expectReason, err)
			}
		})
	}
}

func mustResponse(t *testing.T, e *cerrors.VoipbinError) *sock.Response {
	t.Helper()

	res, err := cerrors.ToResponse(e)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return res
}

// Nothing the requests do writes the secret to a log, to the URI or to an error: a request that
// fails to send, a response that fails to parse, and a request that succeeds.
func Test_config_secretIsNotLogged(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(prev)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	var uris []string
	record := func(_ context.Context, _ string, m *sock.Request) {
		uris = append(uris, m.URI)
	}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, gomock.Any()).Do(record).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testConfigKeyRaw)}, nil)
	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, gomock.Any()).Do(record).Return(nil, fmt.Errorf("could not publish"))
	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, gomock.Any()).Do(record).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`{not json`)}, nil)
	mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, gomock.Any()).Do(record).Return(&sock.Response{StatusCode: 500}, nil)

	var errs []error
	for i := 0; i < 4; i++ {
		_, err := reqHandler.ConfigV1KeySecretSet(context.Background(), "email", "mailgun.api_key", testConfigActor, testConfigSecret, 1)
		errs = append(errs, err)
	}

	for _, uri := range uris {
		if strings.Contains(uri, testConfigSecret) {
			t.Errorf("Wrong match. the secret is in the URI: %s", uri)
		}
	}
	for _, err := range errs {
		if err != nil && strings.Contains(err.Error(), testConfigSecret) {
			t.Errorf("Wrong match. the secret is in an error: %v", err)
		}
	}
	for _, e := range hook.AllEntries() {
		if fields, _ := json.Marshal(e.Data); strings.Contains(e.Message+string(fields), testConfigSecret) {
			t.Errorf("Wrong match. a log line holds the secret: %s", e.Message)
		}
	}
}

// Every method hands the response through the typed error path. The probe of the secret write
// depends on it: a manager that does not know the route answers a bare 404, and that has to
// reach the caller as ErrNotFound, whatever method sent the request.
func Test_config_responseErrors(t *testing.T) {
	type call func(r *requestHandler) error
	calls := map[string]call{
		"service get": func(r *requestHandler) error {
			_, err := r.ConfigV1ServiceGet(context.Background(), "email")
			return err
		},
		"secret set": func(r *requestHandler) error {
			_, err := r.ConfigV1KeySecretSet(context.Background(), "email", "k", testConfigActor, testConfigSecret, 1)
			return err
		},
		"key clear": func(r *requestHandler) error {
			_, err := r.ConfigV1KeyClear(context.Background(), "email", "k", testConfigActor, 1)
			return err
		},
		"history list": func(r *requestHandler) error {
			_, err := r.ConfigV1KeyHistoryList(context.Background(), "email", "k", "", 0)
			return err
		},
	}

	tests := []struct {
		name   string
		resp   *sock.Response
		sock   error
		expect error
		reason string
	}{
		{name: "a manager that does not know the route", resp: &sock.Response{StatusCode: 404}, expect: ErrNotFound},
		{name: "a bare conflict", resp: &sock.Response{StatusCode: 409}, expect: ErrConflict},
		{name: "a bare server error", resp: &sock.Response{StatusCode: 500}, expect: ErrInternal},
		{name: "a typed error", resp: mustResponse(t, cerrors.FailedPrecondition("email-manager", "REVISION_MISMATCH", "stale")), reason: "REVISION_MISMATCH"},
		{name: "a body that is not json", resp: &sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`{not json`)}},
		{name: "the request fails to send", sock: fmt.Errorf("could not publish")},
	}
	for name, c := range calls {
		for _, tt := range tests {
			t.Run(name+"/"+tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				mockSock := sockhandler.NewMockSockHandler(mc)
				reqHandler := requestHandler{sock: mockSock}
				mockSock.EXPECT().RequestPublish(gomock.Any(), testConfigTarget, gomock.Any()).Return(tt.resp, tt.sock)

				err := c(&reqHandler)
				if err == nil {
					t.Fatalf("Wrong match. expect: an error, got: nil")
				}
				if tt.expect != nil && !errors.Is(err, tt.expect) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, err)
				}
				if tt.reason != "" {
					var ve *cerrors.VoipbinError
					if !errors.As(err, &ve) || ve.Reason != tt.reason {
						t.Errorf("Wrong match. expect: a typed error %s, got: %v", tt.reason, err)
					}
				}
			})
		}
	}
}
