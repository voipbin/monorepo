package requesthandler

import (
	"context"
	reflect "reflect"
	"testing"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_FlowV1FlowCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		flowType   fmflow.Type
		flowName   string
		flowDetail string
		actions    []fmaction.Action
		persist    bool

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *fmflow.Flow
	}{
		{
			"normal",

			uuid.FromStringOrNil("857f154e-7f4d-11ec-b669-a7aa025fbeaf"),
			fmflow.TypeFlow,
			"test flow",
			"test flow detail",
			[]fmaction.Action{},
			true,
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0","customer_id":"857f154e-7f4d-11ec-b669-a7aa025fbeaf","type":"flow","name":"test flow","detail":"test flow detail","actions":[],"persist":true,"tm_create":"2020-09-20T03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/flows",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"857f154e-7f4d-11ec-b669-a7aa025fbeaf","type":"flow","name":"test flow","detail":"test flow detail","actions":[],"persist":true}`),
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0"),
				CustomerID: uuid.FromStringOrNil("857f154e-7f4d-11ec-b669-a7aa025fbeaf"),
				Type:       fmflow.TypeFlow,
				Name:       "test flow",
				Detail:     "test flow detail",
				Actions:    []fmaction.Action{},
				Persist:    true,
				TMCreate:   "2020-09-20T03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1FlowCreate(ctx, tt.customerID, tt.flowType, tt.flowName, tt.flowDetail, tt.actions, tt.persist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong matchdfdsfd.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_FlowV1FlowUpdate(t *testing.T) {

	tests := []struct {
		name string

		requestFlow *fmflow.Flow
		response    *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *fmflow.Flow
	}{
		{
			"empty action",
			&fmflow.Flow{
				ID:      uuid.FromStringOrNil("7dc3a1b2-6789-11eb-9f30-1b1cc6d13e51"),
				Name:    "update name",
				Detail:  "update detail",
				Actions: []fmaction.Action{},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7dc3a1b2-6789-11eb-9f30-1b1cc6d13e51","customer_id":"bb832464-7f4d-11ec-aab5-8f3e1e3958d5","name":"update name","detail":"update detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/flows/7dc3a1b2-6789-11eb-9f30-1b1cc6d13e51",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail","actions":[]}`),
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("7dc3a1b2-6789-11eb-9f30-1b1cc6d13e51"),
				CustomerID: uuid.FromStringOrNil("bb832464-7f4d-11ec-aab5-8f3e1e3958d5"),
				Name:       "update name",
				Detail:     "update detail",
				Actions:    []fmaction.Action{},
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1FlowUpdate(ctx, tt.requestFlow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_FlowV1FlowGet(t *testing.T) {

	tests := []struct {
		name string

		flowID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *fmflow.Flow
	}{
		{
			"normal",

			uuid.FromStringOrNil("be66d9a6-6ed6-11eb-8152-0bb66bad7293"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"be66d9a6-6ed6-11eb-8152-0bb66bad7293","customer_id":"c36412ba-7f4d-11ec-a6ec-67db89124047","name":"test flow","detail":"test flow detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/flows/be66d9a6-6ed6-11eb-8152-0bb66bad7293",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("be66d9a6-6ed6-11eb-8152-0bb66bad7293"),
				CustomerID: uuid.FromStringOrNil("c36412ba-7f4d-11ec-a6ec-67db89124047"),
				Name:       "test flow",
				Detail:     "test flow detail",
				Actions:    []fmaction.Action{},
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1FlowGet(ctx, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_FlowV1FlowDelete(t *testing.T) {

	tests := []struct {
		name string

		flowID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *fmflow.Flow
	}{
		{
			"normal",

			uuid.FromStringOrNil("4193c3a2-67ca-11eb-a892-0b6d18cda91a"),
			&sock.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"4193c3a2-67ca-11eb-a892-0b6d18cda91a"}`),
			},

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/flows/4193c3a2-67ca-11eb-a892-0b6d18cda91a",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&fmflow.Flow{
				ID: uuid.FromStringOrNil("4193c3a2-67ca-11eb-a892-0b6d18cda91a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1FlowDelete(ctx, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_FlowV1FlowGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *sock.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectResult  []fmflow.Flow
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"customer_id": "c971cc06-7f4d-11ec-b0dc-5ff21ea97f57",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"158e4b2c-0c55-11eb-b4f2-37c93a78a6a0","customer_id":"c971cc06-7f4d-11ec-b0dc-5ff21ea97f57","name":"test flow","detail":"test flow detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}]`),
			},

			"/v1/flows?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/flows?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&filter_customer_id=c971cc06-7f4d-11ec-b0dc-5ff21ea97f57",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]fmflow.Flow{
				{
					ID:         uuid.FromStringOrNil("158e4b2c-0c55-11eb-b4f2-37c93a78a6a0"),
					CustomerID: uuid.FromStringOrNil("c971cc06-7f4d-11ec-b0dc-5ff21ea97f57"),
					Name:       "test flow",
					Detail:     "test flow detail",
					Actions:    []fmaction.Action{},
					TMCreate:   "2020-09-20 03:23:20.995000",
					TMUpdate:   "",
					TMDelete:   "",
				},
			},
		},
		{
			"get type conference",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"type": string(fmflow.TypeConference),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"158e4b2c-0c55-11eb-b4f2-37c93a78a6a0","customer_id":"d9fceace-7f4d-11ec-8949-cf7a5dce40c9","name":"test flow","detail":"test flow detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}]`),
			},

			"/v1/flows?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/flows?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&filter_type=conference",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]fmflow.Flow{
				{
					ID:         uuid.FromStringOrNil("158e4b2c-0c55-11eb-b4f2-37c93a78a6a0"),
					CustomerID: uuid.FromStringOrNil("d9fceace-7f4d-11ec-8949-cf7a5dce40c9"),
					Name:       "test flow",
					Detail:     "test flow detail",
					Actions:    []fmaction.Action{},
					TMCreate:   "2020-09-20 03:23:20.995000",
					TMUpdate:   "",
					TMDelete:   "",
				},
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1FlowGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_FlowV1FlowUpdateActions(t *testing.T) {

	tests := []struct {
		name string

		flowID  uuid.UUID
		actions []fmaction.Action

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *fmflow.Flow
	}{
		{
			"normal",

			uuid.FromStringOrNil("a645703d-4cd7-4c5d-af76-d2f9f2fafcd0"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a645703d-4cd7-4c5d-af76-d2f9f2fafcd0","customer_id":"bb832464-7f4d-11ec-aab5-8f3e1e3958d5","name":"update name","detail":"update detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/flows/a645703d-4cd7-4c5d-af76-d2f9f2fafcd0/actions",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}]}`),
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("a645703d-4cd7-4c5d-af76-d2f9f2fafcd0"),
				CustomerID: uuid.FromStringOrNil("bb832464-7f4d-11ec-aab5-8f3e1e3958d5"),
				Name:       "update name",
				Detail:     "update detail",
				Actions:    []fmaction.Action{},
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
			},
		},
		{
			"empty actions",

			uuid.FromStringOrNil("0fb53139-3e5d-4ce7-8de6-d39420a18cf5"),
			[]fmaction.Action{},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0fb53139-3e5d-4ce7-8de6-d39420a18cf5","customer_id":"bb832464-7f4d-11ec-aab5-8f3e1e3958d5","name":"update name","detail":"update detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/flows/0fb53139-3e5d-4ce7-8de6-d39420a18cf5/actions",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"actions":[]}`),
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("0fb53139-3e5d-4ce7-8de6-d39420a18cf5"),
				CustomerID: uuid.FromStringOrNil("bb832464-7f4d-11ec-aab5-8f3e1e3958d5"),
				Name:       "update name",
				Detail:     "update detail",
				Actions:    []fmaction.Action{},
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1FlowUpdateActions(ctx, tt.flowID, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
