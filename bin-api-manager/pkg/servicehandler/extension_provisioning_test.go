package servicehandler

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	rmextension "monorepo/bin-registrar-manager/models/extension"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/cachehandler"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const testPublicBaseURL = "https://api.voipbin.net"

func Test_ExtensionProvisioningTokenCreate(t *testing.T) {

	agentAdmin := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	extensionID := uuid.FromStringOrNil("2c1c73e2-80fc-11f0-9a34-9f5be0e5cc4b")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &serviceHandler{
		reqHandler:    mockReq,
		dbHandler:     mockDB,
		cacheHandler:  mockCache,
		publicBaseURL: testPublicBaseURL,
	}
	ctx := context.Background()

	mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(&rmextension.Extension{
		Identity: commonidentity.Identity{
			ID:         extensionID,
			CustomerID: agentAdmin.CustomerID,
		},
	}, nil)

	var storedToken string
	mockCache.EXPECT().ProvisioningTokenSet(ctx, gomock.Any(), extensionID, ProvisioningTokenTTL).DoAndReturn(
		func(_ context.Context, token string, _ uuid.UUID, _ time.Duration) error {
			storedToken = token
			return nil
		},
	)

	before := time.Now().UTC()
	res, err := h.ExtensionProvisioningTokenCreate(ctx, agentAdmin, extensionID)
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// token: 64 lowercase hex chars, same value stored in the cache.
	if len(res.Token) != provisioningTokenLen*2 {
		t.Errorf("wrong token length. expect: %d, got: %d", provisioningTokenLen*2, len(res.Token))
	}
	if _, errDecode := hex.DecodeString(res.Token); errDecode != nil {
		t.Errorf("token is not hex: %v", errDecode)
	}
	if res.Token != storedToken {
		t.Errorf("returned token does not match the stored token.\nstored: %s\ngot: %s", storedToken, res.Token)
	}

	// url: complete public provisioning URL.
	expectURL := testPublicBaseURL + "/provisioning/extension?token=" + res.Token
	if res.URL != expectURL {
		t.Errorf("wrong url.\nexpect: %s\ngot: %s", expectURL, res.URL)
	}

	// expire: RFC3339, now+TTL (allow the call duration as slack).
	expire, errParse := time.Parse(time.RFC3339, res.Expire)
	if errParse != nil {
		t.Fatalf("expire is not RFC3339: %v", errParse)
	}
	if expire.Before(before.Add(ProvisioningTokenTTL).Truncate(time.Second)) || expire.After(after.Add(ProvisioningTokenTTL)) {
		t.Errorf("expire out of range. got: %v (window: %v - %v)", expire, before.Add(ProvisioningTokenTTL), after.Add(ProvisioningTokenTTL))
	}
}

func Test_ExtensionProvisioningTokenCreate_error(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	extensionID := uuid.FromStringOrNil("6e46a1a8-80fc-11f0-a2b8-5f4a2b7ff9cc")

	type test struct {
		name  string
		agent *auth.AuthIdentity

		responseExtension *rmextension.Extension
		responseError     error

		expectError error
	}

	tests := []test{
		{
			name: "direct access is rejected",
			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:   customerID,
				ResourceType: "extension",
				ResourceID:   extensionID,
			}),

			expectError: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),

			responseExtension: &rmextension.Extension{
				Identity: commonidentity.Identity{
					ID:         extensionID,
					CustomerID: customerID,
				},
			},

			expectError: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "other customer's extension",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			responseExtension: &rmextension.Extension{
				Identity: commonidentity.Identity{
					ID:         extensionID,
					CustomerID: uuid.FromStringOrNil("88d353b2-80fc-11f0-8f38-2b8fefcbbde1"),
				},
			},

			expectError: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "extension not found",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			responseError: fmt.Errorf("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &serviceHandler{
				reqHandler:    mockReq,
				dbHandler:     mockDB,
				cacheHandler:  mockCache,
				publicBaseURL: testPublicBaseURL,
			}
			ctx := context.Background()

			if tt.responseExtension != nil || tt.responseError != nil {
				mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(tt.responseExtension, tt.responseError)
			}

			res, err := h.ExtensionProvisioningTokenCreate(ctx, tt.agent, extensionID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: %v", res)
			}
			if tt.expectError != nil && !errors.Is(err, tt.expectError) {
				t.Errorf("Wrong error.\nexpect: %v\ngot: %v", tt.expectError, err)
			}
		})
	}
}

func Test_ExtensionProvisioningXMLGet(t *testing.T) {

	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	extensionID := uuid.FromStringOrNil("b0a1d55e-80fc-11f0-8d10-6f6e2f6de1a1")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &serviceHandler{
		reqHandler:   mockReq,
		dbHandler:    mockDB,
		cacheHandler: mockCache,
	}
	ctx := context.Background()

	mockCache.EXPECT().ProvisioningTokenGet(ctx, token).Return(extensionID, nil)

	// DomainName carries a legacy customer UUID; the Realm-first serving rule
	// (applied by ConvertWebhookMessage) must win. Serving the raw DomainName
	// is a bug this test guards against.
	mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(&rmextension.Extension{
		Identity: commonidentity.Identity{
			ID:         extensionID,
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Extension:  "1001",
		DomainName: "5f621078-8e5f-11ee-97b2-cfe7337b701c",
		Realm:      "ab12.reg.voipbin.net",
		Username:   "1001",
		Password:   "secret123",
	}, nil)

	res, err := h.ExtensionProvisioningXMLGet(ctx, token)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := `<?xml version="1.0" encoding="UTF-8"?>
<config xmlns="http://www.linphone.org/xsds/lpconfig.xsd">
  <section name="misc">
    <entry name="transient_provisioning" overwrite="true">1</entry>
  </section>
  <section name="proxy_0">
    <entry name="reg_proxy" overwrite="true">&lt;sip:ab12.reg.voipbin.net;transport=udp&gt;</entry>
    <entry name="reg_identity" overwrite="true">sip:1001@ab12.reg.voipbin.net</entry>
    <entry name="reg_expires" overwrite="true">3600</entry>
    <entry name="reg_sendregister" overwrite="true">1</entry>
    <entry name="publish" overwrite="true">0</entry>
  </section>
  <section name="auth_info_0">
    <entry name="username" overwrite="true">1001</entry>
    <entry name="domain" overwrite="true">ab12.reg.voipbin.net</entry>
    <entry name="passwd" overwrite="true">secret123</entry>
    <entry name="realm" overwrite="true">ab12.reg.voipbin.net</entry>
  </section>
</config>
`

	if string(res) != expect {
		t.Errorf("Wrong match.\nexpect:\n%s\ngot:\n%s", expect, string(res))
	}
}

func Test_ExtensionProvisioningXMLGet_specialCharPassword(t *testing.T) {

	token := "abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	extensionID := uuid.FromStringOrNil("cf7e990c-80fc-11f0-a3ce-3b6dd1ecdb8b")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &serviceHandler{
		reqHandler:   mockReq,
		dbHandler:    mockDB,
		cacheHandler: mockCache,
	}
	ctx := context.Background()

	mockCache.EXPECT().ProvisioningTokenGet(ctx, token).Return(extensionID, nil)
	mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(&rmextension.Extension{
		Identity: commonidentity.Identity{
			ID: extensionID,
		},
		Extension: "1002",
		Realm:     "cd34.reg.voipbin.net",
		Username:  "1002",
		Password:  `p<a&s"s`,
	}, nil)

	res, err := h.ExtensionProvisioningXMLGet(ctx, token)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// encoding/xml escapes < as &lt;, & as &amp;, and " as &#34; in chardata.
	expectPasswd := `<entry name="passwd" overwrite="true">p&lt;a&amp;s&#34;s</entry>`
	if !strings.Contains(string(res), expectPasswd) {
		t.Errorf("password not escaped as expected.\nexpect fragment: %s\ngot:\n%s", expectPasswd, string(res))
	}

	// The raw (unescaped) password must never appear in the output.
	if strings.Contains(string(res), `p<a&s"s`) {
		t.Errorf("raw password leaked unescaped into the XML:\n%s", string(res))
	}
}

func Test_ExtensionProvisioningXMLGet_error(t *testing.T) {

	token := "1111111111111111111111111111111111111111111111111111111111111111"
	extensionID := uuid.FromStringOrNil("ec6a9c26-80fc-11f0-b0dd-eb1f0ff9ff01")
	tmDelete := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)

	type test struct {
		name string

		cacheID    uuid.UUID
		cacheError error

		responseExtension *rmextension.Extension
		responseError     error

		expectError error
	}

	tests := []test{
		{
			name:       "token miss returns error",
			cacheError: fmt.Errorf("redis: nil"),
		},
		{
			name:    "deleted extension returns error",
			cacheID: extensionID,
			responseExtension: &rmextension.Extension{
				Identity: commonidentity.Identity{
					ID: extensionID,
				},
				Extension: "1003",
				Realm:     "ef56.reg.voipbin.net",
				Username:  "1003",
				Password:  "pass",
				TMDelete:  &tmDelete,
			},

			expectError: serviceerrors.ErrNotFound,
		},
		{
			name:          "extension fetch failure returns error",
			cacheID:       extensionID,
			responseError: fmt.Errorf("rpc failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &serviceHandler{
				reqHandler:   mockReq,
				dbHandler:    mockDB,
				cacheHandler: mockCache,
			}
			ctx := context.Background()

			mockCache.EXPECT().ProvisioningTokenGet(ctx, token).Return(tt.cacheID, tt.cacheError)
			if tt.cacheError == nil {
				mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, tt.cacheID).Return(tt.responseExtension, tt.responseError)
			}

			res, err := h.ExtensionProvisioningXMLGet(ctx, token)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: %s", string(res))
			}
			if tt.expectError != nil && !errors.Is(err, tt.expectError) {
				t.Errorf("Wrong error.\nexpect: %v\ngot: %v", tt.expectError, err)
			}
		})
	}
}
