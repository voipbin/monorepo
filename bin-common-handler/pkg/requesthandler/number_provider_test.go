package requesthandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	nmnumber "monorepo/bin-number-manager/models/number"
	nmprovider "monorepo/bin-number-manager/models/provider"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

const (
	testNumberProviderTarget = "bin-manager.number-manager.request"
	testNumberProviderBody   = `{"provider_name":"telnyx","enabled":true,"priority":10,"has_secret":true,"key_version":1,"credential_source":"db","usable":true}`
	testSecretValue          = "test-secret-value"
)

var testNumberProviderInfo = &nmprovider.Info{
	ProviderName:     "telnyx",
	Enabled:          true,
	Priority:         10,
	HasSecret:        true,
	KeyVersion:       1,
	CredentialSource: nmprovider.CredentialSourceDB,
	Usable:           true,
}

func Test_NumberV1NumberProviderList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testNumberProviderTarget, &sock.Request{
		URI:      "/v1/number_providers",
		Method:   sock.RequestMethodGet,
		DataType: ContentTypeJSON,
	}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte("[" + testNumberProviderBody + "]")}, nil)

	res, err := reqHandler.NumberV1NumberProviderList(context.Background())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual([]*nmprovider.Info{testNumberProviderInfo}, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testNumberProviderInfo, res)
	}
}

func Test_NumberV1NumberProviderGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testNumberProviderTarget, &sock.Request{
		URI:      "/v1/number_providers/telnyx",
		Method:   sock.RequestMethodGet,
		DataType: ContentTypeJSON,
	}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testNumberProviderBody)}, nil)

	res, err := reqHandler.NumberV1NumberProviderGet(context.Background(), nmnumber.ProviderName("telnyx"))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(testNumberProviderInfo, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testNumberProviderInfo, res)
	}
}

func Test_NumberV1NumberProviderGet_escapesName(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testNumberProviderTarget, &sock.Request{
		URI:      "/v1/number_providers/a%2Fsecret",
		Method:   sock.RequestMethodGet,
		DataType: ContentTypeJSON,
	}).Return(&sock.Response{StatusCode: 404}, nil)

	if _, err := reqHandler.NumberV1NumberProviderGet(context.Background(), nmnumber.ProviderName("a/secret")); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}

func Test_NumberV1NumberProviderUpdate(t *testing.T) {
	actor := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")
	falseValue := false
	zero := 0
	ten := 10

	tests := []struct {
		name string

		enabled  *bool
		priority *int
		config   map[string]string

		expectData string
	}{
		{
			name:       "zero values are kept",
			enabled:    &falseValue,
			priority:   &zero,
			expectData: `{"actor_id":"550e8400-e29b-41d4-a716-446655440000","enabled":false,"priority":0}`,
		},
		{
			name:       "all fields",
			enabled:    &falseValue,
			priority:   &ten,
			config:     map[string]string{"connection_id": "test-connection", "profile_id": ""},
			expectData: `{"actor_id":"550e8400-e29b-41d4-a716-446655440000","enabled":false,"priority":10,"config":{"connection_id":"test-connection","profile_id":""}}`,
		},
		{
			name:       "nothing but the actor",
			expectData: `{"actor_id":"550e8400-e29b-41d4-a716-446655440000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublish(gomock.Any(), testNumberProviderTarget, &sock.Request{
				URI:      "/v1/number_providers/telnyx",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(tt.expectData),
			}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testNumberProviderBody)}, nil)

			res, err := reqHandler.NumberV1NumberProviderUpdate(context.Background(), nmnumber.ProviderName("telnyx"), actor, tt.enabled, tt.priority, tt.config)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(testNumberProviderInfo, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testNumberProviderInfo, res)
			}
		})
	}
}

func Test_NumberV1NumberProviderUpdateSecret(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	actor := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")
	mockSock.EXPECT().RequestPublish(gomock.Any(), testNumberProviderTarget, &sock.Request{
		URI:      "/v1/number_providers/telnyx/secret",
		Method:   sock.RequestMethodPut,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"actor_id":"550e8400-e29b-41d4-a716-446655440000","secret":"` + testSecretValue + `"}`),
	}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testNumberProviderBody)}, nil)

	res, err := reqHandler.NumberV1NumberProviderUpdateSecret(context.Background(), nmnumber.ProviderName("telnyx"), actor, testSecretValue)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(testNumberProviderInfo, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testNumberProviderInfo, res)
	}
}

// the delete request must carry the actor in its body: number-manager answers 400 without it
func Test_NumberV1NumberProviderDeleteSecret(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	actor := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")
	mockSock.EXPECT().RequestPublish(gomock.Any(), testNumberProviderTarget, &sock.Request{
		URI:      "/v1/number_providers/telnyx/secret",
		Method:   sock.RequestMethodDelete,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"actor_id":"550e8400-e29b-41d4-a716-446655440000"}`),
	}).Return(&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testNumberProviderBody)}, nil)

	res, err := reqHandler.NumberV1NumberProviderDeleteSecret(context.Background(), nmnumber.ProviderName("telnyx"), actor)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(testNumberProviderInfo, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testNumberProviderInfo, res)
	}
}

func Test_NumberV1NumberProvider_statusErrors(t *testing.T) {
	actor := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")

	calls := map[string]func(r *requestHandler) error{
		"list": func(r *requestHandler) error {
			_, err := r.NumberV1NumberProviderList(context.Background())
			return err
		},
		"get": func(r *requestHandler) error {
			_, err := r.NumberV1NumberProviderGet(context.Background(), "telnyx")
			return err
		},
		"update": func(r *requestHandler) error {
			_, err := r.NumberV1NumberProviderUpdate(context.Background(), "telnyx", actor, nil, nil, nil)
			return err
		},
		"update secret": func(r *requestHandler) error {
			_, err := r.NumberV1NumberProviderUpdateSecret(context.Background(), "telnyx", actor, testSecretValue)
			return err
		},
		"delete secret": func(r *requestHandler) error {
			_, err := r.NumberV1NumberProviderDeleteSecret(context.Background(), "telnyx", actor)
			return err
		},
	}
	statuses := map[int]error{400: ErrBadRequest, 404: ErrNotFound, 503: ErrServiceUnavailable, 500: ErrInternal}

	for name, call := range calls {
		for status, expect := range statuses {
			t.Run(fmt.Sprintf("%s %d", name, status), func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				mockSock := sockhandler.NewMockSockHandler(mc)
				reqHandler := requestHandler{sock: mockSock}

				mockSock.EXPECT().RequestPublish(gomock.Any(), testNumberProviderTarget, gomock.Any()).Return(&sock.Response{StatusCode: status}, nil)

				if err := call(&reqHandler); !errors.Is(err, expect) {
					t.Errorf("Wrong match. status: %d, expect: %v, got: %v", status, expect, err)
				}
			})
		}
	}
}

// a transport failure must not echo the request body, which holds the secret
func Test_NumberV1NumberProviderUpdateSecret_transportErrorHasNoSecret(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), testNumberProviderTarget, gomock.Any()).Return(nil, errors.New("connection lost"))

	_, err := reqHandler.NumberV1NumberProviderUpdateSecret(context.Background(), "telnyx", uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"), testSecretValue)
	if err == nil {
		t.Fatal("Wrong match. expect: error, got: ok")
	}

	if strings.Contains(err.Error(), testSecretValue) {
		t.Errorf("The error holds the secret: %v", err)
	}
}
