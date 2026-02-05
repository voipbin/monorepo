package listenhandler

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/pkg/outdialhandler"
	"monorepo/bin-outdial-manager/pkg/outdialtargethandler"
)

func Test_v1OutdialsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID  uuid.UUID
		campaignID  uuid.UUID
		outdialName string
		detail      string
		data        string
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdials",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "2204d132-b36b-11ec-86e1-07e0556766eb", "campaign_id": "225d0bae-b36b-11ec-82cd-9bcad44a2d49", "name": "test name", "detail": "test detail", "data": "test data"}`),
			},

			uuid.FromStringOrNil("2204d132-b36b-11ec-86e1-07e0556766eb"),
			uuid.FromStringOrNil("225d0bae-b36b-11ec-82cd-9bcad44a2d49"),
			"test name",
			"test detail",
			"test data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdial.EXPECT().Create(gomock.Any(), tt.customerID, tt.campaignID, tt.outdialName, tt.detail, tt.data).Return(&outdial.Outdial{}, nil)

			if _, err := h.processRequest(tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_v1OutdialsGet(t *testing.T) {

	tests := []struct {
		name      string
		filters   map[outdial.Field]any
		pageToken string
		pageSize  uint64
		request   *sock.Request
		outdials  []*outdial.Outdial

		expectRes *sock.Response
	}{
		{
			"1 item",
			map[outdial.Field]any{
				outdial.FieldCustomerID: uuid.FromStringOrNil("3ffc0038-b36c-11ec-8de7-df466e08d7fc"),
				outdial.FieldDeleted:    false,
			},
			"2020-10-10T03:30:17.000000Z",
			10,
			&sock.Request{
				URI:      "/v1/outdials?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"3ffc0038-b36c-11ec-8de7-df466e08d7fc","deleted":false}`),
			},
			[]*outdial.Outdial{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5024139c-b36c-11ec-9b26-9b18d7d76e07"),
					},
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5024139c-b36c-11ec-9b26-9b18d7d76e07","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			"2 items",
			map[outdial.Field]any{
				outdial.FieldCustomerID: uuid.FromStringOrNil("974f7298-b36c-11ec-9d42-07020f9318fb"),
				outdial.FieldDeleted:    false,
			},
			"2020-10-10T03:30:17.000000Z",
			10,
			&sock.Request{
				URI:      "/v1/outdials?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"974f7298-b36c-11ec-9d42-07020f9318fb","deleted":false}`),
			},
			[]*outdial.Outdial{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("977b9012-b36c-11ec-bd33-0bcd25d394aa"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("97a59bd2-b36c-11ec-93c9-eb2b606d229c"),
					},
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"977b9012-b36c-11ec-bd33-0bcd25d394aa","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":null,"tm_update":null,"tm_delete":null},{"id":"97a59bd2-b36c-11ec-93c9-eb2b606d229c","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdial.EXPECT().List(gomock.Any(), tt.pageToken, tt.pageSize, tt.filters).Return(tt.outdials, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialsIDGet(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		outdial   *outdial.Outdial
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdials/323e28ee-b36d-11ec-86bb-6b0ff7646ddc",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},
			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("323e28ee-b36d-11ec-86bb-6b0ff7646ddc"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"323e28ee-b36d-11ec-86bb-6b0ff7646ddc","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdial.EXPECT().Get(gomock.Any(), tt.outdial.ID).Return(tt.outdial, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialsIDPut(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		id          uuid.UUID
		outdialName string
		detail      string

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdials/ee6c0268-b62c-11ec-8ce9-a796a6b09de1",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name": "test name", "detail": "test detail"}`),
			},

			uuid.FromStringOrNil("ee6c0268-b62c-11ec-8ce9-a796a6b09de1"),
			"test name",
			"test detail",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdial.EXPECT().UpdateBasicInfo(gomock.Any(), tt.id, tt.outdialName, tt.detail).Return(&outdial.Outdial{}, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		id uuid.UUID

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdials/6e918d58-b643-11ec-9263-b35286fcc303",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("6e918d58-b643-11ec-9263-b35286fcc303"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdial.EXPECT().Delete(gomock.Any(), tt.id).Return(&outdial.Outdial{}, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialsIDAvailableGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		outdialID uuid.UUID
		tryCount0 int
		tryCount1 int
		tryCount2 int
		tryCount3 int
		tryCount4 int
		limit     uint64

		outdialtargets []*outdialtarget.OutdialTarget
		expectRes      *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdials/c9668112-b36d-11ec-9e02-0f190974012d/available",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"try_count_0":3,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"limit":1}`),
			},

			uuid.FromStringOrNil("c9668112-b36d-11ec-9e02-0f190974012d"),
			3,
			0,
			0,
			0,
			0,
			1,

			[]*outdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("96abf56c-b36e-11ec-a539-d76994cf6863"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"96abf56c-b36e-11ec-a539-d76994cf6863","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdialTarget.EXPECT().GetAvailable(gomock.Any(), tt.outdialID, tt.tryCount0, tt.tryCount1, tt.tryCount2, tt.tryCount3, tt.tryCount4, tt.limit).Return(tt.outdialtargets, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialsIDTargetsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		outdialID    uuid.UUID
		targetName   string
		detail       string
		data         string
		destination0 *commonaddress.Address
		destination1 *commonaddress.Address
		destination2 *commonaddress.Address
		destination3 *commonaddress.Address
		destination4 *commonaddress.Address

		outdialtarget *outdialtarget.OutdialTarget
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdials/9995c64e-b36f-11ec-9be0-d387edb25d6b/targets",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name": "test name", "detail": "test detail", "data": "test data", "destination_0": {"type": "tel", "target": "+821100000001"}, "destination_1": {"type": "tel", "target": "+821100000002"}, "destination_2": {"type": "tel", "target": "+821100000003"}, "destination_3": {"type": "tel", "target": "+821100000004"}, "destination_4": {"type": "tel", "target": "+821100000005"}}`),
			},

			uuid.FromStringOrNil("9995c64e-b36f-11ec-9be0-d387edb25d6b"),
			"test name",
			"test detail",
			"test data",
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000003",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000004",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000005",
			},

			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("be545d6a-b36f-11ec-8ad5-03ccb4c40eeb"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"be545d6a-b36f-11ec-8ad5-03ccb4c40eeb","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			"have 1 destination",
			&sock.Request{
				URI:      "/v1/outdials/276d9c20-b371-11ec-8e7f-fb5a893378f4/targets",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name": "test name", "detail": "test detail", "data": "test data", "destination_0": {"type": "tel", "target": "+821100000001"}}`),
			},

			uuid.FromStringOrNil("276d9c20-b371-11ec-8e7f-fb5a893378f4"),
			"test name",
			"test detail",
			"test data",
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			nil,
			nil,
			nil,
			nil,

			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("be545d6a-b36f-11ec-8ad5-03ccb4c40eeb"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"be545d6a-b36f-11ec-8ad5-03ccb4c40eeb","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdialTarget.EXPECT().Create(gomock.Any(), tt.outdialID, tt.targetName, tt.detail, tt.data, tt.destination0, tt.destination1, tt.destination2, tt.destination3, tt.destination4).Return(tt.outdialtarget, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialsIDTargetsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		outdialID      uuid.UUID
		pageToken      string
		pageSize       uint64
		outdialTargets []*outdialtarget.OutdialTarget

		expectRes *sock.Response
	}{
		{
			"1 item",
			&sock.Request{
				URI:      "/v1/outdials/e3a71d6c-b371-11ec-b69c-2b0e0342d71a/targets?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("e3a71d6c-b371-11ec-b69c-2b0e0342d71a"),
			"2020-10-10T03:30:17.000000Z",
			10,
			[]*outdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("5024139c-b36c-11ec-9b26-9b18d7d76e07"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5024139c-b36c-11ec-9b26-9b18d7d76e07","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			"2 items",
			&sock.Request{
				URI:      "/v1/outdials/e7f5ddd0-b372-11ec-8516-4bc424f70ef9/targets?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("e7f5ddd0-b372-11ec-8516-4bc424f70ef9"),
			"2020-10-10T03:30:17.000000Z",
			10,
			[]*outdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("e822590a-b372-11ec-b755-239020a9003b"),
				},
				{
					ID: uuid.FromStringOrNil("e84d3828-b372-11ec-9936-af8200c58c02"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"e822590a-b372-11ec-b755-239020a9003b","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":null,"tm_update":null,"tm_delete":null},{"id":"e84d3828-b372-11ec-9936-af8200c58c02","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdialTarget.EXPECT().GetsByOutdialID(gomock.Any(), tt.outdialID, tt.pageToken, tt.pageSize).Return(tt.outdialTargets, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialsIDCampaignIDPut(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		outdialID  uuid.UUID
		campaignID uuid.UUID

		outdial   *outdial.Outdial
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdials/643ef1e6-b563-11ec-bacf-d356bc75b302/campaign_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"campaign_id":"646baed4-b563-11ec-980c-f3bbe82e67fb"}`),
			},

			uuid.FromStringOrNil("643ef1e6-b563-11ec-bacf-d356bc75b302"),
			uuid.FromStringOrNil("646baed4-b563-11ec-980c-f3bbe82e67fb"),

			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("643ef1e6-b563-11ec-bacf-d356bc75b302"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"643ef1e6-b563-11ec-bacf-d356bc75b302","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdial.EXPECT().UpdateCampaignID(gomock.Any(), tt.outdialID, tt.campaignID).Return(tt.outdial, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialsIDDataPut(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		outdialID uuid.UUID
		data      string

		outdial   *outdial.Outdial
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdials/beddb5ce-b563-11ec-99e9-dfdfa34a3196/data",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"data":"test_data"}`),
			},

			uuid.FromStringOrNil("beddb5ce-b563-11ec-99e9-dfdfa34a3196"),
			"test_data",

			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("beddb5ce-b563-11ec-99e9-dfdfa34a3196"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"beddb5ce-b563-11ec-99e9-dfdfa34a3196","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
			mockOutdialTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialHandler:       mockOutdial,
				outdialTargetHandler: mockOutdialTarget,
			}

			mockOutdial.EXPECT().UpdateData(gomock.Any(), tt.outdialID, tt.data).Return(tt.outdial, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
