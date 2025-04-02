package campaigncallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID      uuid.UUID
		campaignID      uuid.UUID
		outplanID       uuid.UUID
		outdialID       uuid.UUID
		outdialTargetID uuid.UUID
		queueID         uuid.UUID

		activeflowID uuid.UUID
		flowID       uuid.UUID

		referenceType    campaigncall.ReferenceType
		referenceID      uuid.UUID
		source           *commonaddress.Address
		destination      *commonaddress.Address
		destinationIndex int
		tryCount         int

		responseUUID uuid.UUID
	}{
		{
			name: "normal",

			customerID:      uuid.FromStringOrNil("39a8de10-bd23-44e8-8ac3-2e47bee450cb"),
			campaignID:      uuid.FromStringOrNil("aea5d8af-c2dc-4ffe-b9f0-c73c073a6b10"),
			outplanID:       uuid.FromStringOrNil("0c6b8021-0be9-4127-96a6-44d4b8ead96b"),
			outdialID:       uuid.FromStringOrNil("9b426642-ba6f-464d-943e-1990380f88fd"),
			outdialTargetID: uuid.FromStringOrNil("8895f112-69e6-4a86-ad11-20c8ceef7c0c"),
			queueID:         uuid.FromStringOrNil("57610fb2-d817-4692-99e2-2069315d1ee1"),

			activeflowID: uuid.FromStringOrNil("2dee1247-4100-4e8f-847a-6b8e24fb8c7a"),
			flowID:       uuid.FromStringOrNil("ae34298a-7d5e-468b-8ea3-52a98e5bbcc6"),

			referenceType: campaigncall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("448ffc1f-b599-4c77-8f18-058a39189b97"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			destinationIndex: 1,
			tryCount:         1,
			responseUUID:     uuid.FromStringOrNil("b843739e-6d08-11ee-b95d-d796bff6f089"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaigncallHandler{
				util:          mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().CampaigncallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CampaigncallGet(gomock.Any(), gomock.Any()).Return(&campaigncall.Campaigncall{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), campaigncall.EventTypeCampaigncallCreated, gomock.Any())
			mockReq.EXPECT().OutdialV1OutdialtargetUpdateStatusProgressing(ctx, tt.outdialTargetID, tt.destinationIndex).Return(&omoutdialtarget.OutdialTarget{}, nil)

			_, err := h.Create(
				ctx,
				tt.customerID,
				tt.campaignID,
				tt.outplanID,
				tt.outdialID,
				tt.outdialTargetID,
				tt.queueID,
				tt.activeflowID,
				tt.flowID,
				tt.referenceType,
				tt.referenceID,
				tt.source,
				tt.destination,
				tt.destinationIndex,
				tt.tryCount,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string
		res  *campaigncall.Campaigncall
	}{
		{
			"test normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f61bbb06-be07-4824-9f0f-d6e4a90e8370"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			mockDB.EXPECT().CampaigncallGet(ctx, tt.res.ID).Return(tt.res, nil)

			_, err := h.Get(ctx, tt.res.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetByReferenceID(t *testing.T) {

	tests := []struct {
		name string
		res  *campaigncall.Campaigncall
	}{
		{
			"test normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7e2e3282-7500-4ee6-8ea1-718e95650138"),
				},
				ReferenceID: uuid.FromStringOrNil("2b8a469f-a0b0-4ab9-8057-08e54a950a58"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			mockDB.EXPECT().CampaigncallGetByReferenceID(ctx, tt.res.ReferenceID).Return(tt.res, nil)

			_, err := h.GetByReferenceID(ctx, tt.res.ReferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetByActiveflowID(t *testing.T) {

	tests := []struct {
		name string
		res  *campaigncall.Campaigncall
	}{
		{
			"test normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3969e306-062c-4406-8a2c-edb677113e44"),
				},
				ActiveflowID: uuid.FromStringOrNil("4d71f838-4b5a-460d-bd0c-0454397eb4ef"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			mockDB.EXPECT().CampaigncallGetByActiveflowID(ctx, tt.res.ActiveflowID).Return(tt.res, nil)

			_, err := h.GetByActiveflowID(ctx, tt.res.ActiveflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		token      string
		limit      uint64
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("ebe2c7c6-6e30-11ee-9db0-c37efb343d1d"),
			token:      "2020-10-10 03:30:17.000000",
			limit:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			mockDB.EXPECT().CampaigncallGetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit).Return(nil, nil)

			_, err := h.GetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetsByCampaignID(t *testing.T) {

	tests := []struct {
		name       string
		campaignID uuid.UUID
		token      string
		limit      uint64
	}{
		{
			"normal",
			uuid.FromStringOrNil("0820b0ef-3014-431e-80a6-a45dd223a310"),
			"2020-10-10 03:30:17.000000",
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			mockDB.EXPECT().CampaigncallGetsByCampaignID(ctx, tt.campaignID, tt.token, tt.limit).Return(nil, nil)

			_, err := h.GetsByCampaignID(ctx, tt.campaignID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetsByCampaignIDAndStatus(t *testing.T) {

	tests := []struct {
		name       string
		campaignID uuid.UUID
		status     campaigncall.Status
		token      string
		limit      uint64
	}{
		{
			"normal",
			uuid.FromStringOrNil("76f6f058-3ae2-41d2-9c33-ad1785b3b024"),
			campaigncall.StatusDialing,
			"2020-10-10 03:30:17.000000",
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			mockDB.EXPECT().CampaigncallGetsByCampaignIDAndStatus(ctx, tt.campaignID, tt.status, tt.token, tt.limit).Return(nil, nil)

			_, err := h.GetsByCampaignIDAndStatus(ctx, tt.campaignID, tt.status, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetsOngoingByCampaignID(t *testing.T) {

	tests := []struct {
		name       string
		campaignID uuid.UUID
		token      string
		limit      uint64
	}{
		{
			"normal",
			uuid.FromStringOrNil("2086fd09-f91d-4ddc-9aa4-0f300959a1e7"),
			"2020-10-10 03:30:17.000000",
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			mockDB.EXPECT().CampaigncallGetsOngoingByCampaignID(ctx, tt.campaignID, tt.token, tt.limit).Return(nil, nil)

			_, err := h.GetsOngoingByCampaignID(ctx, tt.campaignID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_updateStatus(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status campaigncall.Status

		responseCampaigncall *campaigncall.Campaigncall
		expectRes            *campaigncall.Campaigncall
	}{
		{
			"normal",

			uuid.FromStringOrNil("fb2d86f8-6324-466e-9db3-7f6aca309c41"),
			campaigncall.StatusProgressing,

			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fb2d86f8-6324-466e-9db3-7f6aca309c41"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fb2d86f8-6324-466e-9db3-7f6aca309c41"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaigncallUpdateStatus(ctx, tt.id, tt.status).Return(nil)
			mockDB.EXPECT().CampaigncallGet(ctx, tt.id).Return(tt.responseCampaigncall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaigncall.CustomerID, campaigncall.EventTypeCampaigncallUpdated, tt.responseCampaigncall)

			res, err := h.updateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_updateStatusDone(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		result campaigncall.Result

		responseCampaigncall *campaigncall.Campaigncall
		expectRes            *campaigncall.Campaigncall
	}{
		{
			"result success",

			uuid.FromStringOrNil("7e73112c-704b-4f22-8c20-231bc9b4b03e"),
			campaigncall.ResultSuccess,

			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7e73112c-704b-4f22-8c20-231bc9b4b03e"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7e73112c-704b-4f22-8c20-231bc9b4b03e"),
				},
			},
		},
		{
			"result fail",

			uuid.FromStringOrNil("c62b883d-294b-4a96-a276-63911776ac28"),
			campaigncall.ResultFail,

			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c62b883d-294b-4a96-a276-63911776ac28"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c62b883d-294b-4a96-a276-63911776ac28"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, tt.id, campaigncall.StatusDone, tt.result).Return(nil)
			mockDB.EXPECT().CampaigncallGet(ctx, tt.id).Return(tt.responseCampaigncall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaigncall.CustomerID, campaigncall.EventTypeCampaigncallUpdated, tt.responseCampaigncall)

			res, err := h.updateStatusDone(ctx, tt.id, tt.result)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCampaigncall *campaigncall.Campaigncall
	}{
		{
			"normal",

			uuid.FromStringOrNil("ef3feb86-db79-4dab-a55d-41d65a231c10"),

			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ef3feb86-db79-4dab-a55d-41d65a231c10"),
					CustomerID: uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
				},
				FlowID: uuid.FromStringOrNil("60e0f90a-db73-4aaf-add8-6b7cd8edc82c"),
				Status: campaigncall.StatusDone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().CampaigncallGet(ctx, tt.id).Return(tt.responseCampaigncall, nil)
			mockDB.EXPECT().CampaigncallDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().CampaigncallGet(ctx, gomock.Any()).Return(tt.responseCampaigncall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaigncall.CustomerID, campaigncall.EventTypeCampaigncallDeleted, tt.responseCampaigncall)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseCampaigncall) {
				t.Errorf("Wrong match.\ngot: %v\n, expect: %v\n", res, tt.responseCampaigncall)
			}
		})
	}
}
