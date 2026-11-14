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
	testNumberProviderBody   = `{"provider_name":"telnyx","enabled":true,"priority":10,"has_secret":true,"credential_source":"env","usable":true}`
)

var testNumberProviderInfo = &nmprovider.Info{
	ProviderName:     "telnyx",
	Enabled:          true,
	Priority:         10,
	HasSecret:        true,
	CredentialSource: nmprovider.CredentialSourceEnv,
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
