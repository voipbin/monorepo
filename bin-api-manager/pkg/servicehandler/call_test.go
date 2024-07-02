package servicehandler

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/websockhandler"
)

func Test_callGet(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		callID uuid.UUID

		responseCall *cmcall.Call
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)

			res, err := h.callGet(ctx, tt.agent, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseCall) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseCall, res)
			}
		})
	}
}

func Test_callGet_error(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		callID uuid.UUID

		responseCall      *cmcall.Call
		responseCallError error
	}{
		{
			name: "call get returns an error",
			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			callID: uuid.FromStringOrNil("7b7e58de-8f37-11ed-8852-0f407ad6849f"),

			responseCallError: fmt.Errorf(""),
		},
		{
			name: "deleted call info",
			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			callID: uuid.FromStringOrNil("7b7e58de-8f37-11ed-8852-0f407ad6849f"),

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b7e58de-8f37-11ed-8852-0f407ad6849f"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				TMDelete: "2020-09-20 03:23:20.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, tt.responseCallError)

			_, err := h.callGet(ctx, tt.agent, tt.callID)
			if err == nil {
				t.Error("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_CallCreate(t *testing.T) {

	tests := []struct {
		name string

		agent        *amagent.Agent
		flowID       uuid.UUID
		actions      []fmaction.Action
		source       *commonaddress.Address
		destinations []commonaddress.Address

		responseFlow       *fmflow.Flow
		responseCalls      []*cmcall.Call
		responseGroupcalls []*cmgroupcall.Groupcall

		expectResCalls      []*cmcall.WebhookMessage
		expectResGroupcalls []*cmgroupcall.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			flowID:  uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
			actions: []fmaction.Action{},
			source: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
					Target: "testdestination@test.com",
				},
			},

			responseFlow: &fmflow.Flow{
				ID:         uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			responseCalls: []*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
					},
				},
			},
			responseGroupcalls: []*cmgroupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("44b6d84f-48bd-4189-aad2-b9271de78ca7"),
				},
			},

			expectResCalls: []*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
					},
				},
			},
			expectResGroupcalls: []*cmgroupcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("44b6d84f-48bd-4189-aad2-b9271de78ca7"),
				},
			},
		},
		{
			name: "with actions only",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			flowID: uuid.Nil,
			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			source: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
					Target: "testdestination@test.com",
				},
			},

			responseFlow: &fmflow.Flow{
				ID:         uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			responseCalls: []*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
					},
				},
			},
			responseGroupcalls: []*cmgroupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("44b6d84f-48bd-4189-aad2-b9271de78ca7"),
				},
			},

			expectResCalls: []*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
					},
				},
			},
			expectResGroupcalls: []*cmgroupcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("44b6d84f-48bd-4189-aad2-b9271de78ca7"),
				},
			},
		},
		{
			name: "if both has given, flowid has more priority",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			flowID: uuid.FromStringOrNil("2ca43d36-8df9-11ec-846a-ebf271da36c8"),
			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			source: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
					Target: "testdestination@test.com",
				},
			},

			responseFlow: &fmflow.Flow{
				ID:         uuid.FromStringOrNil("2ca43d36-8df9-11ec-846a-ebf271da36c8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			responseCalls: []*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
					},
				},
			},
			responseGroupcalls: []*cmgroupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("44b6d84f-48bd-4189-aad2-b9271de78ca7"),
				},
			},

			expectResCalls: []*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
					},
				},
			},
			expectResGroupcalls: []*cmgroupcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("44b6d84f-48bd-4189-aad2-b9271de78ca7"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			flowID := tt.flowID
			if flowID == uuid.Nil {
				mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.agent.CustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.actions, false).Return(tt.responseFlow, nil)
				flowID = tt.responseFlow.ID
			}
			mockReq.EXPECT().FlowV1FlowGet(ctx, flowID).Return(tt.responseFlow, nil)

			mockReq.EXPECT().CallV1CallsCreate(ctx, tt.agent.CustomerID, tt.responseFlow.ID, uuid.Nil, tt.source, tt.destinations, false, false).Return(tt.responseCalls, tt.responseGroupcalls, nil)

			resCalls, resGroupcalls, err := h.CallCreate(ctx, tt.agent, tt.flowID, tt.actions, tt.source, tt.destinations)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(resCalls, tt.expectResCalls) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectResCalls, resCalls)
			}
			if !reflect.DeepEqual(resGroupcalls, tt.expectResGroupcalls) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectResGroupcalls, resGroupcalls)
			}

		})
	}
}

func Test_CallDelete(t *testing.T) {

	tests := []struct {
		name   string
		agent  *amagent.Agent
		callID uuid.UUID

		responseCall *cmcall.Call

		expectRes *cmcall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("eccc7bf4-8926-11ed-b638-0fcef48a97d2"),
			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eccc7bf4-8926-11ed-b638-0fcef48a97d2"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},

			&cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eccc7bf4-8926-11ed-b638-0fcef48a97d2"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1CallDelete(ctx, tt.callID).Return(tt.responseCall, nil)

			res, err := h.CallDelete(ctx, tt.agent, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallHangup(t *testing.T) {

	tests := []struct {
		name   string
		agent  *amagent.Agent
		callID uuid.UUID

		responseCall *cmcall.Call

		expectRes *cmcall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("9e9ed0b6-6791-11eb-9810-87fda8377194"),
			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9e9ed0b6-6791-11eb-9810-87fda8377194"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},

			&cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9e9ed0b6-6791-11eb-9810-87fda8377194"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1CallHangup(ctx, tt.callID).Return(tt.responseCall, nil)

			res, err := h.CallHangup(ctx, tt.agent, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallTalk(t *testing.T) {

	tests := []struct {
		name     string
		agent    *amagent.Agent
		callID   uuid.UUID
		text     string
		gender   string
		language string

		responseCall *cmcall.Call
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("89f97b66-a4b6-11ed-b3a8-9732500c39be"),
			"hello world",
			"female",
			"en-US",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("89f97b66-a4b6-11ed-b3a8-9732500c39be"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.callID, tt.text, tt.gender, tt.language, 10000).Return(nil)

			if err := h.CallTalk(ctx, tt.agent, tt.callID, tt.text, tt.gender, tt.language); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_CallHoldOn(t *testing.T) {

	tests := []struct {
		name   string
		agent  *amagent.Agent
		callID uuid.UUID

		responseCall *cmcall.Call
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("4db40768-cef8-11ed-bb96-8fbbe25ae0fa"),

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4db40768-cef8-11ed-bb96-8fbbe25ae0fa"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1CallHoldOn(ctx, tt.callID).Return(nil)

			if err := h.CallHoldOn(ctx, tt.agent, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_CallHoldOff(t *testing.T) {

	tests := []struct {
		name   string
		agent  *amagent.Agent
		callID uuid.UUID

		responseCall *cmcall.Call
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("7079cc38-cef8-11ed-9410-b35f9ccb992c"),

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7079cc38-cef8-11ed-9410-b35f9ccb992c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1CallHoldOff(ctx, tt.callID).Return(nil)

			if err := h.CallHoldOff(ctx, tt.agent, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_CallMuteOn(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		callID    uuid.UUID
		direction cmcall.MuteDirection

		responseCall *cmcall.Call
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("70a879e8-cef8-11ed-a112-13d831e46695"),
			cmcall.MuteDirectionBoth,

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("70a879e8-cef8-11ed-a112-13d831e46695"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1CallMuteOn(ctx, tt.callID, tt.direction).Return(nil)

			if err := h.CallMuteOn(ctx, tt.agent, tt.callID, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallMuteOff(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		callID    uuid.UUID
		direction cmcall.MuteDirection

		responseCall *cmcall.Call
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("70d6557a-cef8-11ed-95b3-0b608cbf435e"),
			cmcall.MuteDirectionBoth,

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("70d6557a-cef8-11ed-95b3-0b608cbf435e"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1CallMuteOff(ctx, tt.callID, tt.direction).Return(nil)

			if err := h.CallMuteOff(ctx, tt.agent, tt.callID, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallGets(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		pageToken string
		pageSize  uint64

		responseCalls []cmcall.Call
		expectFilters map[string]string
		expectRes     []*cmcall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			"2021-03-01 01:00:00.995000",
			10,

			[]cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1fbeb120-b08c-11ee-9298-8373260919fa"),
					},
				},
			},
			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			[]*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1fbeb120-b08c-11ee-9298-8373260919fa"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseCalls, nil)

			res, err := h.CallGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_CallMediaStreamStart(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		callID        uuid.UUID
		encapsulation string
		writer        http.ResponseWriter
		request       *http.Request

		responseCall *cmcall.Call

		expectRes []*cmcall.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("1299b152-e921-11ee-889f-7b65e5d7a225"),
			"rtp",
			&mockResponseWriter{},
			&http.Request{},

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1299b152-e921-11ee-889f-7b65e5d7a225"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
			[]*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1fbeb120-b08c-11ee-9298-8373260919fa"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockWebsock := websockhandler.NewMockWebsockHandler(mc)

			h := serviceHandler{
				reqHandler:     mockReq,
				dbHandler:      mockDB,
				websockHandler: mockWebsock,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockWebsock.EXPECT().RunMediaStream(ctx, tt.writer, tt.request, cmexternalmedia.ReferenceTypeCall, tt.callID, tt.encapsulation).Return(nil)

			if err := h.CallMediaStreamStart(ctx, tt.agent, tt.callID, tt.encapsulation, tt.writer, tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
