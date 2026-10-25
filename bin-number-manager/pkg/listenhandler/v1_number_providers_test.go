package listenhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
)

var testActor = uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000a1")

func newProviderListenHandler(t *testing.T) (*listenHandler, *providerhandler.MockProviderHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockProvider := providerhandler.NewMockProviderHandler(mc)
	return &listenHandler{
		sockHandler:     sockhandler.NewMockSockHandler(mc),
		numberHandler:   numberhandler.NewMockNumberHandler(mc),
		providerHandler: mockProvider,
	}, mockProvider
}

func Test_processRequest_numberProviders(t *testing.T) {
	info := &provider.Info{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 10, HasSecret: true, KeyVersion: 2, CredentialSource: provider.CredentialSourceDB, Usable: true}

	tests := []struct {
		name string

		request *sock.Request
		setup   func(m *providerhandler.MockProviderHandler)

		expectStatus int
	}{
		{
			name:    "list",
			request: &sock.Request{URI: "/v1/number_providers", Method: sock.RequestMethodGet},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().List(gomock.Any()).Return([]*provider.Info{info}, nil)
			},
			expectStatus: 200,
		},
		{
			name:    "get",
			request: &sock.Request{URI: "/v1/number_providers/telnyx", Method: sock.RequestMethodGet},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Get(gomock.Any(), number.ProviderNameTelnyx).Return(info, nil)
			},
			expectStatus: 200,
		},
		{
			name:    "get unknown is 404",
			request: &sock.Request{URI: "/v1/number_providers/nope", Method: sock.RequestMethodGet},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Get(gomock.Any(), number.ProviderName("nope")).Return(nil, fmt.Errorf("x: %w", dbhandler.ErrNotFound))
			},
			expectStatus: 404,
		},
		{
			name: "update",
			request: &sock.Request{
				URI: "/v1/number_providers/telnyx", Method: sock.RequestMethodPut,
				Data: []byte(`{"actor_id":"a0000000-0000-4000-8000-0000000000a1","priority":30,"config":{"connection_id":"c1"}}`),
			},
			setup: func(m *providerhandler.MockProviderHandler) {
				prio := 30
				m.EXPECT().Update(gomock.Any(), number.ProviderNameTelnyx, testActor, nil, &prio, map[string]string{"connection_id": "c1"}).Return(info, nil)
			},
			expectStatus: 200,
		},
		{
			name: "update with invalid input is 400",
			request: &sock.Request{
				URI: "/v1/number_providers/telnyx", Method: sock.RequestMethodPut,
				Data: []byte(`{"actor_id":"a0000000-0000-4000-8000-0000000000a1","priority":-1}`),
			},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Update(gomock.Any(), number.ProviderNameTelnyx, testActor, nil, gomock.Any(), nil).Return(nil, fmt.Errorf("%w: priority", provider.ErrInvalidArgument))
			},
			expectStatus: 400,
		},
		{
			name:         "update with a broken body is 400",
			request:      &sock.Request{URI: "/v1/number_providers/telnyx", Method: sock.RequestMethodPut, Data: []byte(`{`)},
			setup:        func(m *providerhandler.MockProviderHandler) {},
			expectStatus: 400,
		},
		{
			name: "secret put",
			request: &sock.Request{
				URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPut,
				Data: []byte(`{"actor_id":"a0000000-0000-4000-8000-0000000000a1","secret":"fake-secret-value"}`),
			},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().UpdateSecret(gomock.Any(), number.ProviderNameTelnyx, testActor, "fake-secret-value").Return(info, nil)
			},
			expectStatus: 200,
		},
		{
			name: "secret put without a key is 503",
			request: &sock.Request{
				URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPut,
				Data: []byte(`{"actor_id":"a0000000-0000-4000-8000-0000000000a1","secret":"fake-secret-value"}`),
			},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().UpdateSecret(gomock.Any(), number.ProviderNameTelnyx, testActor, "fake-secret-value").Return(nil, provider.ErrSecretKeyNotConfigured)
			},
			expectStatus: 503,
		},
		{
			name: "secret put with an internal failure is 500",
			request: &sock.Request{
				URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPut,
				Data: []byte(`{"actor_id":"a0000000-0000-4000-8000-0000000000a1","secret":"fake-secret-value"}`),
			},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().UpdateSecret(gomock.Any(), number.ProviderNameTelnyx, testActor, "fake-secret-value").Return(nil, errors.New("db failure"))
			},
			expectStatus: 500,
		},
		{
			name:         "secret put with a broken body is 400",
			request:      &sock.Request{URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPut, Data: []byte(`{"secret":"fake-secret-value"`)},
			setup:        func(m *providerhandler.MockProviderHandler) {},
			expectStatus: 400,
		},
		{
			name: "secret delete",
			request: &sock.Request{
				URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodDelete,
				Data: []byte(`{"actor_id":"a0000000-0000-4000-8000-0000000000a1"}`),
			},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().DeleteSecret(gomock.Any(), number.ProviderNameTelnyx, testActor).Return(info, nil)
			},
			expectStatus: 200,
		},
		{
			name:         "secret delete with a broken body is 400",
			request:      &sock.Request{URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodDelete, Data: []byte(`{`)},
			setup:        func(m *providerhandler.MockProviderHandler) {},
			expectStatus: 400,
		},
		{
			name: "secret delete without a row is 404",
			request: &sock.Request{
				URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodDelete,
				Data: []byte(`{"actor_id":"a0000000-0000-4000-8000-0000000000a1"}`),
			},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().DeleteSecret(gomock.Any(), number.ProviderNameTelnyx, testActor).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name:    "list failure is 500",
			request: &sock.Request{URI: "/v1/number_providers", Method: sock.RequestMethodGet},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().List(gomock.Any()).Return(nil, errors.New("x"))
			},
			expectStatus: 500,
		},
		{
			name:    "get failure is 500",
			request: &sock.Request{URI: "/v1/number_providers/telnyx", Method: sock.RequestMethodGet},
			setup: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errors.New("x"))
			},
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newProviderListenHandler(t)
			tt.setup(m)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}

			// no response may carry a secret or a ciphertext field
			if res.StatusCode == http.StatusOK {
				var raw map[string]any
				if err := json.Unmarshal(res.Data, &raw); err != nil {
					var list []map[string]any
					if err2 := json.Unmarshal(res.Data, &list); err2 != nil {
						t.Fatalf("Could not parse the response: %v", err)
					}
					raw = list[0]
				}
				for _, k := range []string{"secret", "secret_ciphertext", "ciphertext"} {
					if _, ok := raw[k]; ok {
						t.Errorf("The response has the field %q", k)
					}
				}
				if raw["has_secret"] != true {
					t.Errorf("The response must tell has_secret")
				}
			}
		})
	}
}

// no URI variation may route a number provider request elsewhere, and unrouted ones are 404
func Test_processRequest_numberProviders_routing(t *testing.T) {
	tests := []struct {
		uri    string
		method sock.RequestMethod
	}{
		{"/v1/number_providers/", sock.RequestMethodGet},
		{"/v1/number_providers/Telnyx", sock.RequestMethodGet},
		{"/v1/number_providers/telnyx/other", sock.RequestMethodGet},
		{"/v1/number_providers/telnyx/secret", sock.RequestMethodGet},
		{"/v1/number_providers/telnyx/secret", sock.RequestMethodPost},
		{"/v1/number_providers", sock.RequestMethodPut},
		{"/x/v1/number_providers", sock.RequestMethodGet},
		{"/v1/number_providers/" + strings.Repeat("a", 33), sock.RequestMethodGet},
		{"/v1/number_providers/telnyx/secret/", sock.RequestMethodPut},
	}

	for _, tt := range tests {
		t.Run(string(tt.method)+" "+tt.uri, func(t *testing.T) {
			h, _ := newProviderListenHandler(t)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, Data: []byte(`{"secret":"x"}`)})
			if err != nil || res.StatusCode != http.StatusNotFound {
				t.Errorf("Wrong match. expect: 404, got: %v, %v", res, err)
			}
		})
	}
}

// the body of a secret request must not reach any log line, whatever the outcome
func Test_processRequest_secretNotLogged(t *testing.T) {
	const secret = "fake-secret-must-not-be-logged"

	hook := logrustest.NewGlobal()
	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetLevel(prevLevel)
		hook.Reset()
	})

	body := []byte(`{"actor_id":"a0000000-0000-4000-8000-0000000000a1","secret":"` + secret + `"}`)
	badBody := []byte(`{"secret":"` + secret + `"`)

	tests := []struct {
		name    string
		request *sock.Request
		setup   func(m *providerhandler.MockProviderHandler, stored *string)
	}{
		{
			name:    "success: the secret still reaches the handler",
			request: &sock.Request{URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPut, Data: body},
			setup: func(m *providerhandler.MockProviderHandler, stored *string) {
				m.EXPECT().UpdateSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, _ number.ProviderName, _ uuid.UUID, s string) (*provider.Info, error) {
						*stored = s
						return &provider.Info{}, nil
					})
			},
		},
		{
			name:    "handler failure",
			request: &sock.Request{URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPut, Data: body},
			setup: func(m *providerhandler.MockProviderHandler, stored *string) {
				m.EXPECT().UpdateSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, _ number.ProviderName, _ uuid.UUID, s string) (*provider.Info, error) {
						*stored = s
						return nil, errors.New("db failure")
					})
			},
		},
		{
			name:    "broken body",
			request: &sock.Request{URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPut, Data: badBody},
			setup:   func(m *providerhandler.MockProviderHandler, stored *string) {},
		},
		{
			name:    "unrouted method on the secret URI",
			request: &sock.Request{URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPost, Data: body},
			setup:   func(m *providerhandler.MockProviderHandler, stored *string) {},
		},
		{
			name:    "unrouted mixed case URI",
			request: &sock.Request{URI: "/v1/Number_Providers/telnyx/Secret", Method: sock.RequestMethodPut, Data: body},
			setup:   func(m *providerhandler.MockProviderHandler, stored *string) {},
		},
		{
			name:    "unrouted URI of another prefix that ends with secret",
			request: &sock.Request{URI: "/v1/other/secret", Method: sock.RequestMethodPut, Data: body},
			setup:   func(m *providerhandler.MockProviderHandler, stored *string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()
			h, m := newProviderListenHandler(t)

			stored := ""
			tt.setup(m, &stored)

			if _, err := h.processRequest(tt.request); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if len(hook.AllEntries()) == 0 {
				t.Fatalf("Nothing was logged, so the check proves nothing")
			}

			for _, e := range hook.AllEntries() {
				// the daemon logs JSON; check both the JSON and the text rendering
				asJSON, _ := (&logrus.JSONFormatter{}).Format(e)
				asText, _ := e.String()
				if bytes.Contains(asJSON, []byte(secret)) || strings.Contains(asText, secret) {
					t.Errorf("A log line holds the secret: %s", asText)
				}
			}

			if strings.Contains(tt.name, "success") || strings.Contains(tt.name, "handler failure") {
				if stored != secret {
					t.Errorf("The secret did not reach the handler. got: %q", stored)
				}
			}
		})
	}
}

func Test_logSafeRequest(t *testing.T) {
	plain := &sock.Request{URI: "/v1/numbers", Method: sock.RequestMethodPost, Data: []byte(`{"a":1}`)}
	if logSafeRequest(plain) != plain {
		t.Errorf("A non provider request must be logged as it is")
	}

	secret := &sock.Request{URI: "/v1/number_providers/telnyx/secret", Method: sock.RequestMethodPut, Data: []byte(`{"secret":"x"}`)}
	safe := logSafeRequest(secret)
	if string(safe.Data) != `"[REDACTED]"` || string(secret.Data) != `{"secret":"x"}` {
		t.Errorf("Wrong redaction. safe: %s, original: %s", safe.Data, secret.Data)
	}
}

func Test_errorResponse_provider(t *testing.T) {
	tests := []struct {
		err    error
		expect int
	}{
		{provider.ErrInvalidArgument, http.StatusBadRequest},
		{fmt.Errorf("wrapped: %w", provider.ErrInvalidArgument), http.StatusBadRequest},
		{provider.ErrSecretKeyNotConfigured, http.StatusServiceUnavailable},
		{fmt.Errorf("wrapped: %w", dbhandler.ErrNotFound), http.StatusNotFound},
		{errors.New("other"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		if got := errorResponse(tt.err).StatusCode; got != tt.expect {
			t.Errorf("Wrong match. err: %v, expect: %d, got: %d", tt.err, tt.expect, got)
		}
	}
}
