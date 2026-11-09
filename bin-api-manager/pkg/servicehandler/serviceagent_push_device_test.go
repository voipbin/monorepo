package servicehandler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	pmdevice "monorepo/bin-push-manager/models/device"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	pdTestCustomerID      = uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	pdTestOtherCustomerID = uuid.FromStringOrNil("6a1b3c4d-8e5f-11ee-97b2-cfe7337b701c")
	pdTestAgentID         = uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	pdTestOtherAgentID    = uuid.FromStringOrNil("e263f7af-105b-11ee-b395-eb18426de979")
	pdTestDeviceID        = uuid.FromStringOrNil("3f6c1d2e-8a4b-4c7d-9e10-5b2a7c8d9e01")
)

const pdTestToken = "fake-push-token-0123456789"

func pdTestAgent(perm amagent.Permission) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: pdTestAgentID, CustomerID: pdTestCustomerID},
		Permission: perm,
	})
}

func pdTestNilIDAgent() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{CustomerID: pdTestCustomerID},
		Permission: amagent.PermissionCustomerAgent,
	})
}

func pdTestDevice(customerID, agentID uuid.UUID) *pmdevice.Device {
	created := time.Date(2026, 11, 9, 9, 15, 30, 123456000, time.UTC)
	return &pmdevice.Device{
		Identity: commonidentity.Identity{ID: pdTestDeviceID, CustomerID: customerID},
		AgentID:  agentID,
		Platform: pmdevice.PlatformAndroid,
		Token:    pdTestToken,
		TMCreate: &created,
		TMUpdate: &created,
	}
}

func Test_ServiceAgentPushDeviceCreate(t *testing.T) {
	rpcErr := errors.New("rpc failed")

	tests := []struct {
		name     string
		agent    *auth.AuthIdentity
		platform pmdevice.Platform
		token    string

		expectCall bool
		callRes    *pmdevice.Device
		callErr    error

		expectErr  error
		expectCode cerrors.Status
	}{
		{
			name:       "ok",
			agent:      pdTestAgent(amagent.PermissionCustomerAgent),
			platform:   pmdevice.PlatformAndroid,
			token:      pdTestToken,
			expectCall: true,
			callRes:    pdTestDevice(pdTestCustomerID, pdTestAgentID),
		},
		{
			name:       "token of the maximum length",
			agent:      pdTestAgent(amagent.PermissionCustomerAgent),
			platform:   pmdevice.PlatformAndroid,
			token:      strings.Repeat("a", pmdevice.MaxTokenLength),
			expectCall: true,
			callRes:    pdTestDevice(pdTestCustomerID, pdTestAgentID),
		},
		{
			name:      "access key identity is not an agent",
			agent:     auth.NewDirectIdentity(&auth.DirectScope{CustomerID: pdTestCustomerID}),
			platform:  pmdevice.PlatformAndroid,
			token:     pdTestToken,
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:      "agent with a nil id",
			agent:     pdTestNilIDAgent(),
			platform:  pmdevice.PlatformAndroid,
			token:     pdTestToken,
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			// PermissionAll is the "any authenticated agent" sentinel, so an agent without permission bits passes.
			name:       "agent without permission bits is still an agent",
			agent:      pdTestAgent(amagent.PermissionNone),
			platform:   pmdevice.PlatformAndroid,
			token:      pdTestToken,
			expectCall: true,
			callRes:    pdTestDevice(pdTestCustomerID, pdTestAgentID),
		},
		{
			name:      "unknown platform",
			agent:     pdTestAgent(amagent.PermissionCustomerAgent),
			platform:  pmdevice.Platform("ios"),
			token:     pdTestToken,
			expectErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name:      "empty platform",
			agent:     pdTestAgent(amagent.PermissionCustomerAgent),
			platform:  pmdevice.Platform(""),
			token:     pdTestToken,
			expectErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name:      "empty token",
			agent:     pdTestAgent(amagent.PermissionCustomerAgent),
			platform:  pmdevice.PlatformAndroid,
			token:     "",
			expectErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name:      "token of 513 bytes",
			agent:     pdTestAgent(amagent.PermissionCustomerAgent),
			platform:  pmdevice.PlatformAndroid,
			token:     strings.Repeat("a", pmdevice.MaxTokenLength+1),
			expectErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name:       "rpc error is propagated",
			agent:      pdTestAgent(amagent.PermissionCustomerAgent),
			platform:   pmdevice.PlatformAndroid,
			token:      pdTestToken,
			expectCall: true,
			callErr:    rpcErr,
			expectErr:  rpcErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectCall {
				mockReq.EXPECT().PushV1DeviceCreate(ctx, pdTestCustomerID, pdTestAgentID, tt.platform, tt.token).Return(tt.callRes, tt.callErr)
			}

			res, err := h.ServiceAgentPushDeviceCreate(ctx, tt.agent, tt.platform, tt.token)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || res != nil {
					t.Fatalf("Wrong error. expect: %v, got: %v, %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error. err: %v", err)
			}

			if !reflect.DeepEqual(res, tt.callRes.ConvertWebhookMessage()) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.callRes.ConvertWebhookMessage(), res)
			}

			// the token is a credential and must not be part of the response
			b, err := json.Marshal(res)
			if err != nil {
				t.Fatalf("Could not marshal the response. err: %v", err)
			}
			if strings.Contains(string(b), tt.token) || strings.Contains(string(b), "token") {
				t.Errorf("The response must not contain the token. got: %s", b)
			}
		})
	}
}

func Test_ServiceAgentPushDeviceDelete(t *testing.T) {
	rpcErr := errors.New("rpc failed")
	notFound := cerrors.NotFound("push-manager", "DEVICE_NOT_FOUND", "The device was not found.")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		expectGet bool
		getRes    *pmdevice.Device
		getErr    error

		expectDelete bool
		deleteRes    *pmdevice.Device
		deleteErr    error

		expectErr      error
		expectNotFound bool
	}{
		{
			name:         "ok",
			agent:        pdTestAgent(amagent.PermissionCustomerAgent),
			expectGet:    true,
			getRes:       pdTestDevice(pdTestCustomerID, pdTestAgentID),
			expectDelete: true,
			deleteRes:    pdTestDevice(pdTestCustomerID, pdTestAgentID),
		},
		{
			name:      "access key identity is not an agent",
			agent:     auth.NewDirectIdentity(&auth.DirectScope{CustomerID: pdTestCustomerID}),
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:      "agent with a nil id",
			agent:     pdTestNilIDAgent(),
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:         "agent without permission bits is still an agent",
			agent:        pdTestAgent(amagent.PermissionNone),
			expectGet:    true,
			getRes:       pdTestDevice(pdTestCustomerID, pdTestAgentID),
			expectDelete: true,
			deleteRes:    pdTestDevice(pdTestCustomerID, pdTestAgentID),
		},
		{
			name:           "device of another agent",
			agent:          pdTestAgent(amagent.PermissionCustomerAgent),
			expectGet:      true,
			getRes:         pdTestDevice(pdTestCustomerID, pdTestOtherAgentID),
			expectNotFound: true,
		},
		{
			name:           "device of another customer",
			agent:          pdTestAgent(amagent.PermissionCustomerAgent),
			expectGet:      true,
			getRes:         pdTestDevice(pdTestOtherCustomerID, pdTestAgentID),
			expectNotFound: true,
		},
		{
			name:           "device does not exist (typed error)",
			agent:          pdTestAgent(amagent.PermissionCustomerAgent),
			expectGet:      true,
			getErr:         notFound,
			expectNotFound: true,
		},
		{
			name:           "device does not exist (bare status)",
			agent:          pdTestAgent(amagent.PermissionCustomerAgent),
			expectGet:      true,
			getErr:         requesthandler.ErrNotFound,
			expectNotFound: true,
		},
		{
			name:      "get rpc error is propagated",
			agent:     pdTestAgent(amagent.PermissionCustomerAgent),
			expectGet: true,
			getErr:    rpcErr,
			expectErr: rpcErr,
		},
		{
			name:           "device deleted by another request between the check and the delete",
			agent:          pdTestAgent(amagent.PermissionCustomerAgent),
			expectGet:      true,
			getRes:         pdTestDevice(pdTestCustomerID, pdTestAgentID),
			expectDelete:   true,
			deleteErr:      notFound,
			expectNotFound: true,
		},
		{
			name:         "delete rpc error is propagated",
			agent:        pdTestAgent(amagent.PermissionCustomerAgent),
			expectGet:    true,
			getRes:       pdTestDevice(pdTestCustomerID, pdTestAgentID),
			expectDelete: true,
			deleteErr:    rpcErr,
			expectErr:    rpcErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectGet {
				mockReq.EXPECT().PushV1DeviceGet(ctx, pdTestDeviceID).Return(tt.getRes, tt.getErr)
			}
			if tt.expectDelete {
				mockReq.EXPECT().PushV1DeviceDelete(ctx, pdTestDeviceID).Return(tt.deleteRes, tt.deleteErr)
			}

			res, err := h.ServiceAgentPushDeviceDelete(ctx, tt.agent, pdTestDeviceID)

			if tt.expectNotFound {
				expect := newPushDeviceNotFound()
				var ve *cerrors.VoipbinError
				if res != nil || !errors.As(err, &ve) || !reflect.DeepEqual(ve, expect) {
					t.Fatalf("Wrong error.\nexpect: %v\ngot: %v, %v", expect, res, err)
				}
				return
			}
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || res != nil {
					t.Fatalf("Wrong error. expect: %v, got: %v, %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error. err: %v", err)
			}

			if !reflect.DeepEqual(res, tt.deleteRes.ConvertWebhookMessage()) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.deleteRes.ConvertWebhookMessage(), res)
			}
			b, _ := json.Marshal(res)
			if strings.Contains(string(b), pdTestToken) {
				t.Errorf("The response must not contain the token. got: %s", b)
			}
		})
	}
}
