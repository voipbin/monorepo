package resourcehandler

import (
	"context"
	"monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		responseResources []*resource.Resource
	}{
		{
			"normal",

			10,
			"2021-11-23 17:55:39.712000",
			map[string]string{
				"deleted": "false",
			},

			[]*resource.Resource{
				{
					ID: uuid.FromStringOrNil("75e047f2-2405-11ef-abee-cb381cab6868"),
				},
				{
					ID: uuid.FromStringOrNil("760688ea-2405-11ef-924a-8bf953132abe"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &resourceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ResourceGets(gomock.Any(), tt.size, tt.token, tt.filters).Return(tt.responseResources, nil)
			_, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseResource *resource.Resource
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("c4a52c18-2405-11ef-92fd-2fb65db6bd13"),

			responseResource: &resource.Resource{
				ID: uuid.FromStringOrNil("c4a52c18-2405-11ef-92fd-2fb65db6bd13"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &resourceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ResourceGet(ctx, tt.id).Return(tt.responseResource, nil)
			_, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		ownerID       uuid.UUID
		referenceType resource.ReferenceType
		referenceID   uuid.UUID
		data          interface{}

		responseUUID   uuid.UUID
		expectResource *resource.Resource
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("29a90fb2-2406-11ef-8a33-438eafdd7b0c"),
			ownerID:       uuid.FromStringOrNil("2a270746-2406-11ef-8582-3fefacea4c2b"),
			referenceType: resource.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("2a58ba48-2406-11ef-812d-3bdc17575fa1"),
			data: map[string]interface{}{
				"test": "test",
			},

			responseUUID: uuid.FromStringOrNil("2a8d9dd0-2406-11ef-a78d-cf72585c2887"),
			expectResource: &resource.Resource{
				ID:            uuid.FromStringOrNil("2a8d9dd0-2406-11ef-a78d-cf72585c2887"),
				CustomerID:    uuid.FromStringOrNil("29a90fb2-2406-11ef-8a33-438eafdd7b0c"),
				OwnerID:       uuid.FromStringOrNil("2a270746-2406-11ef-8582-3fefacea4c2b"),
				ReferenceType: resource.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("2a58ba48-2406-11ef-812d-3bdc17575fa1"),
				Data: map[string]interface{}{
					"test": "test",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &resourceHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().ResourceCreate(ctx, tt.expectResource).Return(nil)
			mockDB.EXPECT().ResourceGet(ctx, tt.responseUUID).Return(tt.expectResource, nil)
			mockNotify.EXPECT().PublishEvent(ctx, resource.EventTypeResourceCreated, tt.expectResource)

			res, err := h.Create(ctx, tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.expectResource) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResource, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseResource *resource.Resource
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("583066d4-240e-11ef-80ae-b7c3af9b8a67"),

			responseResource: &resource.Resource{
				ID: uuid.FromStringOrNil("583066d4-240e-11ef-80ae-b7c3af9b8a67"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &resourceHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ResourceDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ResourceGet(ctx, tt.id).Return(tt.responseResource, nil)
			mockNotify.EXPECT().PublishEvent(ctx, resource.EventTypeResourceDeleted, tt.responseResource)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.responseResource) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseResource, res)
			}
		})
	}
}

func Test_UpdateData(t *testing.T) {

	tests := []struct {
		name string

		id   uuid.UUID
		data interface{}

		responseResource *resource.Resource
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("b00d29be-240e-11ef-9b6c-5f97b9475ad3"),
			data: map[string]interface{}{
				"test": "test",
			},

			responseResource: &resource.Resource{
				ID: uuid.FromStringOrNil("b00d29be-240e-11ef-9b6c-5f97b9475ad3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &resourceHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ResourceSetData(ctx, tt.id, tt.data).Return(nil)
			mockDB.EXPECT().ResourceGet(ctx, tt.id).Return(tt.responseResource, nil)
			mockNotify.EXPECT().PublishEvent(ctx, resource.EventTypeResourceUpdated, tt.responseResource)

			res, err := h.UpdateData(ctx, tt.id, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.responseResource) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseResource, res)
			}
		})
	}
}
