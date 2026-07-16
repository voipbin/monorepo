package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	wcsession "monorepo/bin-webchat-manager/models/session"
	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
)

// Test_WebchatSessionList_NoFilter verifies the existing behavior is
// unchanged when widgetID is uuid.Nil (no filter requested): only
// customer_id/deleted are applied, and widgetGet is never called.
func Test_WebchatSessionList_NoFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: mockUtil,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	sessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseSessions := []*wcsession.Session{
		{Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID}},
	}

	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
	mockReq.EXPECT().WebchatV1SessionList(ctx, "2026-01-01 00:00:00.000000", uint64(10), map[wcsession.Field]any{
		wcsession.FieldCustomerID: customerID,
		wcsession.FieldDeleted:    false,
	}).Return(responseSessions, nil)

	res, err := h.WebchatSessionList(ctx, a, 10, "", uuid.Nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := []*wcsession.WebhookMessage{responseSessions[0].ConvertWebhookMessage()}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

// Test_WebchatSessionList_WidgetFilter verifies that a non-nil widgetID
// (a) is validated for ownership via widgetGet and (b) is added to the
// filter map passed to WebchatV1SessionList.
func Test_WebchatSessionList_WidgetFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: mockUtil,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
	}
	responseSessions := []*wcsession.Session{}

	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(responseWidget, nil)
	mockReq.EXPECT().WebchatV1SessionList(ctx, "2026-01-01 00:00:00.000000", uint64(10), map[wcsession.Field]any{
		wcsession.FieldCustomerID: customerID,
		wcsession.FieldDeleted:    false,
		wcsession.FieldWidgetID:   widgetID,
	}).Return(responseSessions, nil)

	res, err := h.WebchatSessionList(ctx, a, 10, "", widgetID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: empty, got: %v", res)
	}
}

// Test_WebchatSessionList_WidgetFilter_CrossTenant is a regression guard:
// a caller must not be able to pass another customer's widget_id to
// enumerate that customer's sessions, even though the caller's own
// hasPermission check against a.CustomerID trivially passes.
func Test_WebchatSessionList_WidgetFilter_CrossTenant(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: mockUtil,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	callerCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	victimCustomerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	victimWidgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: callerCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	victimWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: victimWidgetID, CustomerID: victimCustomerID},
	}

	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
	mockReq.EXPECT().WebchatV1WidgetGet(ctx, victimWidgetID).Return(victimWidget, nil)

	if _, err := h.WebchatSessionList(ctx, a, 10, "", victimWidgetID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}
