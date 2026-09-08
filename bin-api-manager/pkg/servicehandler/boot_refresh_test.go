package servicehandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	nowStr  = "2026-09-09T00:00:00.000000Z"
	nowTime = time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
)

const (
	refreshHash       = "direct.abcdefabcdef"
	refreshBootExpire = "2999-01-01T00:00:00.000000Z"
)

var (
	refreshDirectID   = uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	refreshCustomerID = uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")
	refreshResourceID = uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")
	refreshAllowedID  = uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")
)

// refreshFingerprint mirrors what AuthBoot would have stored, using the same
// signing key the test handler is built with.
func refreshFingerprint() string {
	h := serviceHandler{jwtKey: []byte("testkey")}
	return h.directHashFingerprint(refreshHash)
}

func refreshScope() *auth.DirectScope {
	return &auth.DirectScope{
		CustomerID:           refreshCustomerID,
		ResourceType:         dmdirect.ResourceTypeAI,
		ResourceID:           refreshResourceID,
		AllowedResourceTypes: []string{"aicall"},
		AllowedResourceID:    refreshAllowedID,
		DirectID:             refreshDirectID,
		HashFingerprint:      refreshFingerprint(),
		BootExpire:           refreshBootExpire,
		ScopeVersion:         DirectScopeVersionCurrent,
	}
}

func refreshIdentity(scope *auth.DirectScope) *auth.AuthIdentity {
	return &auth.AuthIdentity{
		Type:        auth.TypeDirect,
		CustomerID:  refreshCustomerID,
		DirectScope: scope,
	}
}

func refreshDirect() *dmdirect.Direct {
	return &dmdirect.Direct{
		Identity:     commonidentity.Identity{ID: refreshDirectID, CustomerID: refreshCustomerID},
		ResourceType: dmdirect.ResourceTypeAI,
		ResourceID:   refreshResourceID,
		Hash:         refreshHash,
	}
}

func refreshHandler(mc *gomock.Controller) (serviceHandler, *requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler) {
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	return serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: mockUtil,
		jwtKey:      []byte("testkey"),
	}, mockReq, mockUtil
}

// Test_AuthBootRefresh_carriesAssignmentForward is the whole point of the
// endpoint: a refreshed token must keep pointing at the same resource, or the
// conversation the visitor is in the middle of dies at ~3h55m.
func Test_AuthBootRefresh_carriesAssignmentForward(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq, mockUtil := refreshHandler(mc)
	ctx := context.Background()
	scope := refreshScope()

	mockUtil.EXPECT().TimeGetCurTime().Return(nowStr)
	mockUtil.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil)
	mockUtil.EXPECT().TimeParseWithError(refreshBootExpire).Return(nowTime.Add(time.Hour*10), nil)
	mockReq.EXPECT().DirectV1DirectGet(ctx, refreshDirectID).Return(refreshDirect(), nil)
	mockReq.EXPECT().CustomerV1CustomerGet(ctx, refreshCustomerID).Return(&cscustomer.Customer{Status: cscustomer.StatusActive}, nil)
	mockUtil.EXPECT().TimeGetCurTimeAdd(BootExpiration).Return("2026-09-08T04:00:00.000000Z")

	res, err := h.AuthBootRefresh(ctx, refreshIdentity(scope))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if res.AllowedResourceID != refreshAllowedID {
		t.Errorf("Assignment was not carried forward. expected: %v, got: %v", refreshAllowedID, res.AllowedResourceID)
	}
	if res.ResourceData != nil {
		t.Errorf("Refresh must not re-fetch resource data, got: %v", res.ResourceData)
	}

	// Every field must survive, not just the ones the response echoes. Dropping
	// CustomerID breaks the WebSocket customer check; dropping DirectID or
	// HashFingerprint breaks the *second* refresh, silently doubling the
	// effective session ceiling.
	claim := parseDirectScope(t, res.Token, h.jwtKey)
	if claim.AllowedResourceID != refreshAllowedID ||
		claim.DirectID != refreshDirectID ||
		claim.HashFingerprint != scope.HashFingerprint ||
		claim.BootExpire != refreshBootExpire ||
		claim.ScopeVersion != DirectScopeVersionCurrent ||
		claim.CustomerID != refreshCustomerID {
		t.Errorf("Refreshed claim lost a field: %+v", claim)
	}
}

// Test_AuthBootRefresh_rejects covers every gate. They all return 401 except
// the identity check, which is 403, so that the client can tell "boot again"
// from "stop".
func Test_AuthBootRefresh_rejects(t *testing.T) {
	agentIdentity := &auth.AuthIdentity{Type: auth.TypeAgent, CustomerID: refreshCustomerID}

	nilAllowed := refreshScope()
	nilAllowed.AllowedResourceID = uuid.Nil
	nilDirect := refreshScope()
	nilDirect.DirectID = uuid.Nil
	emptyFingerprint := refreshScope()
	emptyFingerprint.HashFingerprint = ""

	tests := []struct {
		name      string
		identity  *auth.AuthIdentity
		setup     func(*requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler)
		expectErr error
	}{
		{
			name:      "non-direct identity is forbidden, not unauthorized",
			identity:  agentIdentity,
			setup:     func(*requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler) {},
			expectErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:      "nil identity",
			identity:  nil,
			setup:     func(*requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler) {},
			expectErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:      "token predates binding, no allowed resource id",
			identity:  refreshIdentity(nilAllowed),
			setup:     func(*requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler) {},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:      "no direct id",
			identity:  refreshIdentity(nilDirect),
			setup:     func(*requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler) {},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:      "no hash fingerprint",
			identity:  refreshIdentity(emptyFingerprint),
			setup:     func(*requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler) {},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:     "boot expire unparseable",
			identity: refreshIdentity(refreshScope()),
			setup: func(_ *requesthandler.MockRequestHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeGetCurTime().Return(nowStr).AnyTimes()
				u.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil).AnyTimes()
				u.EXPECT().TimeParseWithError(refreshBootExpire).Return(time.Time{}, fmt.Errorf("bad"))
			},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:     "absolute ceiling reached",
			identity: refreshIdentity(refreshScope()),
			setup: func(_ *requesthandler.MockRequestHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeGetCurTime().Return(nowStr).AnyTimes()
				u.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil).AnyTimes()
				u.EXPECT().TimeParseWithError(refreshBootExpire).Return(nowTime.Add(-time.Minute), nil)
			},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:     "direct record deleted",
			identity: refreshIdentity(refreshScope()),
			setup: func(r *requesthandler.MockRequestHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeGetCurTime().Return(nowStr).AnyTimes()
				u.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil).AnyTimes()
				u.EXPECT().TimeParseWithError(refreshBootExpire).Return(nowTime.Add(time.Hour), nil)
				r.EXPECT().DirectV1DirectGet(gomock.Any(), refreshDirectID).Return(nil, fmt.Errorf("not found"))
			},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			// The case that matters most: regenerating the hash is how an
			// operator revokes a leaked public link. Refresh must lose to it.
			name:     "hash was regenerated",
			identity: refreshIdentity(refreshScope()),
			setup: func(r *requesthandler.MockRequestHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeGetCurTime().Return(nowStr).AnyTimes()
				u.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil).AnyTimes()
				u.EXPECT().TimeParseWithError(refreshBootExpire).Return(nowTime.Add(time.Hour), nil)
				rotated := refreshDirect()
				rotated.Hash = "direct.rotatedrotated"
				r.EXPECT().DirectV1DirectGet(gomock.Any(), refreshDirectID).Return(rotated, nil)
			},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:     "customer lookup fails, must fail closed",
			identity: refreshIdentity(refreshScope()),
			setup: func(r *requesthandler.MockRequestHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeGetCurTime().Return(nowStr).AnyTimes()
				u.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil).AnyTimes()
				u.EXPECT().TimeParseWithError(refreshBootExpire).Return(nowTime.Add(time.Hour), nil)
				r.EXPECT().DirectV1DirectGet(gomock.Any(), refreshDirectID).Return(refreshDirect(), nil)
				r.EXPECT().CustomerV1CustomerGet(gomock.Any(), refreshCustomerID).Return(nil, fmt.Errorf("rpc down"))
			},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:     "customer not active",
			identity: refreshIdentity(refreshScope()),
			setup: func(r *requesthandler.MockRequestHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeGetCurTime().Return(nowStr).AnyTimes()
				u.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil).AnyTimes()
				u.EXPECT().TimeParseWithError(refreshBootExpire).Return(nowTime.Add(time.Hour), nil)
				r.EXPECT().DirectV1DirectGet(gomock.Any(), refreshDirectID).Return(refreshDirect(), nil)
				r.EXPECT().CustomerV1CustomerGet(gomock.Any(), refreshCustomerID).Return(&cscustomer.Customer{Status: cscustomer.StatusDeleted}, nil)
			},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name: "resource type no longer mappable",
			identity: func() *auth.AuthIdentity {
				return refreshIdentity(refreshScope())
			}(),
			setup: func(r *requesthandler.MockRequestHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeGetCurTime().Return(nowStr).AnyTimes()
				u.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil).AnyTimes()
				u.EXPECT().TimeParseWithError(refreshBootExpire).Return(nowTime.Add(time.Hour), nil)
				unmapped := refreshDirect()
				unmapped.ResourceType = "something_else"
				r.EXPECT().DirectV1DirectGet(gomock.Any(), refreshDirectID).Return(unmapped, nil)
				r.EXPECT().CustomerV1CustomerGet(gomock.Any(), refreshCustomerID).Return(&cscustomer.Customer{Status: cscustomer.StatusActive}, nil)
			},
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockReq, mockUtil := refreshHandler(mc)
			tt.setup(mockReq, mockUtil)

			res, err := h.AuthBootRefresh(context.Background(), tt.identity)
			if err == nil {
				t.Fatalf("Expected error, got response: %+v", res)
			}
			if !stderrors.Is(err, tt.expectErr) {
				t.Errorf("Expected %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// Test_AuthBootRefresh_clampsToCeiling pins that the ceiling is applied on the
// duration. authJWTGenerateWithExpiration takes a duration and computes the
// expiry itself, so an absolute expire computed here would be discarded and the
// ceiling would silently stop bounding anything.
func Test_AuthBootRefresh_clampsToCeiling(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq, mockUtil := refreshHandler(mc)
	ctx := context.Background()

	// 10 minutes left, far less than the 4 hour default.
	mockUtil.EXPECT().TimeGetCurTime().Return(nowStr)
	mockUtil.EXPECT().TimeParseWithError(nowStr).Return(nowTime, nil)
	mockUtil.EXPECT().TimeParseWithError(refreshBootExpire).Return(nowTime.Add(time.Minute*10), nil)
	mockReq.EXPECT().DirectV1DirectGet(ctx, refreshDirectID).Return(refreshDirect(), nil)
	mockReq.EXPECT().CustomerV1CustomerGet(ctx, refreshCustomerID).Return(&cscustomer.Customer{Status: cscustomer.StatusActive}, nil)
	mockUtil.EXPECT().TimeGetCurTimeAdd(gomock.Cond(func(d time.Duration) bool {
		return d > 0 && d <= time.Minute*10
	})).Return("2026-09-08T00:10:00.000000Z")

	if _, err := h.AuthBootRefresh(ctx, refreshIdentity(refreshScope())); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}
