package listenhandler

import (
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/directhandler"
)

func TestProcessV1DirectsGet(t *testing.T) {
	tmCreate := func() *time.Time { t := time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC); return &t }()

	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string
		filters   map[direct.Field]any

		directs   []*direct.Direct
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/directs?page_size=10&page_token=2021-11-23T17:55:39.712000Z",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			10,
			"2021-11-23T17:55:39.712000Z",
			map[direct.Field]any{},

			[]*direct.Direct{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
						CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
					},
					ResourceType: "extension",
					ResourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					Hash:         "direct.abcdef123456",
					TMCreate:     tmCreate,
					TMUpdate:     nil,
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","resource_type":"extension","resource_id":"c31676f0-4e69-11ec-afe3-77ba49fae527","hash":"direct.abcdef123456","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockDirect := directhandler.NewMockDirectHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				directHandler: mockDirect,
			}

			mockDirect.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, gomock.Any()).Return(tt.directs, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestProcessV1DirectsPost(t *testing.T) {
	tmCreate := func() *time.Time { t := time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC); return &t }()

	tests := []struct {
		name    string
		request *sock.Request

		customerID   uuid.UUID
		resourceType string
		resourceID   uuid.UUID

		direct    *direct.Direct
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/directs",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","resource_type":"extension","resource_id":"c31676f0-4e69-11ec-afe3-77ba49fae527"}`),
			},

			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
			"extension",
			uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),

			&direct.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				ResourceType: "extension",
				ResourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				Hash:         "direct.abcdef123456",
				TMCreate:     tmCreate,
				TMUpdate:     nil,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","resource_type":"extension","resource_id":"c31676f0-4e69-11ec-afe3-77ba49fae527","hash":"direct.abcdef123456","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockDirect := directhandler.NewMockDirectHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				directHandler: mockDirect,
			}

			mockDirect.EXPECT().Create(gomock.Any(), tt.customerID, tt.resourceType, tt.resourceID).Return(tt.direct, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestProcessV1DirectsIDGet(t *testing.T) {
	tmCreate := func() *time.Time { t := time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC); return &t }()

	tests := []struct {
		name    string
		request *sock.Request

		id uuid.UUID

		direct    *direct.Direct
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/directs/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),

			&direct.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				ResourceType: "extension",
				ResourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				Hash:         "direct.abcdef123456",
				TMCreate:     tmCreate,
				TMUpdate:     nil,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","resource_type":"extension","resource_id":"c31676f0-4e69-11ec-afe3-77ba49fae527","hash":"direct.abcdef123456","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockDirect := directhandler.NewMockDirectHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				directHandler: mockDirect,
			}

			mockDirect.EXPECT().Get(gomock.Any(), tt.id).Return(tt.direct, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestProcessV1DirectsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id uuid.UUID

		responseDirect *direct.Direct
		expectRes      *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/directs/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),

			&direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"00000000-0000-0000-0000-000000000000","resource_type":"","resource_id":"00000000-0000-0000-0000-000000000000","hash":"","tm_create":null,"tm_update":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockDirect := directhandler.NewMockDirectHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				directHandler: mockDirect,
			}

			mockDirect.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseDirect, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestProcessV1DirectsByHashGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		hash string

		responseDirect *direct.Direct
		expectRes      *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/directs/by-hash/direct.abcdef123456",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"direct.abcdef123456",

			&direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				},
				Hash: "direct.abcdef123456",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"00000000-0000-0000-0000-000000000000","resource_type":"","resource_id":"00000000-0000-0000-0000-000000000000","hash":"direct.abcdef123456","tm_create":null,"tm_update":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockDirect := directhandler.NewMockDirectHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				directHandler: mockDirect,
			}

			mockDirect.EXPECT().GetByHash(gomock.Any(), tt.hash).Return(tt.responseDirect, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestProcessV1DirectsIDRegenerate(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id uuid.UUID

		responseDirect *direct.Direct
		expectRes      *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/directs/c31676f0-4e69-11ec-afe3-77ba49fae527/regenerate",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),

			&direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				},
				Hash: "direct.newgenerated",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"00000000-0000-0000-0000-000000000000","resource_type":"","resource_id":"00000000-0000-0000-0000-000000000000","hash":"direct.newgenerated","tm_create":null,"tm_update":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockDirect := directhandler.NewMockDirectHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				directHandler: mockDirect,
			}

			mockDirect.EXPECT().Regenerate(gomock.Any(), tt.id).Return(tt.responseDirect, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
