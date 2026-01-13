package listenhandler

import (
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/taghandler"
)

func TestProcessV1TagsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string
		filters   map[tag.Field]any

		tags      []*tag.Tag
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/tags?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			10,
			"2021-11-23 17:55:39.712000",
			map[tag.Field]any{
				tag.FieldCustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				tag.FieldDeleted:    false,
			},

			[]*tag.Tag{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
						CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
					},
					Name:     "test tag 1",
					Detail:   "test tag 1 detail",
					TMCreate: "2021-11-23 17:55:39.712000",
					TMUpdate: "9999-01-01 00:00:00.000000",
					TMDelete: "9999-01-01 00:00:00.000000",
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","name":"test tag 1","detail":"test tag 1 detail","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
		{
			"have 2 results",
			&sock.Request{
				URI:      "/v1/tags?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			10,
			"2021-11-23 17:55:39.712000",
			map[tag.Field]any{
				tag.FieldCustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				tag.FieldDeleted:    false,
			},

			[]*tag.Tag{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
						CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
					},
					Name:     "test tag 1",
					Detail:   "test tag 1 detail",
					TMCreate: "2021-11-23 17:55:39.712000",
					TMUpdate: "9999-01-01 00:00:00.000000",
					TMDelete: "9999-01-01 00:00:00.000000",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("7379c73c-4e69-11ec-b667-4313a9abe846"),
						CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
					},
					Name:     "test tag 2",
					Detail:   "test tag 2 detail",
					TMCreate: "2021-11-23 17:55:39.712000",
					TMUpdate: "9999-01-01 00:00:00.000000",
					TMDelete: "9999-01-01 00:00:00.000000",
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","name":"test tag 1","detail":"test tag 1 detail","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"},{"id":"7379c73c-4e69-11ec-b667-4313a9abe846","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","name":"test tag 2","detail":"test tag 2 detail","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				tagHandler: mockTag,
			}

			mockTag.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.filters).Return(tt.tags, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1TagsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID uuid.UUID
		tagName    string
		detail     string

		tag       *tag.Tag
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/tags",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","name": "test tag1", "detail": "test tag1 detail"}`),
			},

			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
			"test tag1",
			"test tag1 detail",

			&tag.Tag{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				Name:     "test tag1",
				Detail:   "test tag1 detail",
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","name":"test tag1","detail":"test tag1 detail","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				tagHandler: mockTag,
			}

			mockTag.EXPECT().Create(gomock.Any(), tt.customerID, tt.tagName, tt.detail).Return(tt.tag, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1TagsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id uuid.UUID

		tag       *tag.Tag
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/tags/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),

			&tag.Tag{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				Name:     "test tag1",
				Detail:   "test tag1 detail",
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","name":"test tag1","detail":"test tag1 detail","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				tagHandler: mockTag,
			}

			mockTag.EXPECT().Get(gomock.Any(), tt.id).Return(tt.tag, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1TagsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id      uuid.UUID
		tagName string
		detail  string

		resonseTag *tag.Tag
		expectRes  *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/tags/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name", "detail": "update detail"}`),
			},

			uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			"update name",
			"update detail",

			&tag.Tag{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			mockTag.EXPECT().UpdateBasicInfo(gomock.Any(), tt.id, tt.tagName, tt.detail).Return(tt.resonseTag, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1TagsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id uuid.UUID

		responseTag *tag.Tag
		expectRes   *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/tags/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),

			&tag.Tag{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			mockTag.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseTag, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
