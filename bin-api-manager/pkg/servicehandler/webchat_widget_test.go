package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
)

func Test_WebchatWidgetGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
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
		Name:     "Support Widget",
		Status:   wcwidget.StatusActive,
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(responseWidget, nil)

	res, err := h.WebchatWidgetGet(ctx, a, widgetID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := responseWidget.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

func Test_WebchatWidgetGet_PermissionDenied(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	widgetOwnerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	requesterID := uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	// requester belongs to a DIFFERENT customer than the widget owner --
	// must be rejected regardless of the requester's own permission bits.
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: requesterID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: widgetOwnerID},
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(responseWidget, nil)

	if _, err := h.WebchatWidgetGet(ctx, a, widgetID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

func Test_WebchatWidgetGet_DirectAccessNotSupported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
	}
	ctx := context.Background()

	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		ResourceType:         "webchat_widget",
		ResourceID:           widgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	// A visitor's direct-scope JWT must never be able to read Widget
	// admin config (theme, flow_id, welcome_message) -- only
	// SessionCreate/MessageCreate/SessionEnd are direct-reachable.
	if _, err := h.WebchatWidgetGet(ctx, a, widgetID); err == nil {
		t.Error("Wrong match. expect: direct access not supported error, got: ok")
	}
}
