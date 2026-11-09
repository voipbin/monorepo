package requesthandler

import (
	"context"
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

// The service name picks the queue, and a service without config support is an error that does
// not publish anything.
func Test_config_unknownService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}
	ctx := context.Background()

	for _, service := range []string{"", "nothing", "EMAIL", "email/../x"} {
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
		if strings.Contains(e.Message+fmt.Sprint(e.Data), testConfigSecret) {
			t.Errorf("Wrong match. a log line holds the secret: %s", e.Message)
		}
	}
}
