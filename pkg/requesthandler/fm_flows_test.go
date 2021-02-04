package requesthandler

import (
	"fmt"
	"net/url"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmaction"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestFMFlowCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request", "bin-manager.storage-manager.request")

	type test struct {
		name string

		userID     uint64
		flowID     uuid.UUID
		flowName   string
		flowDetail string
		actions    []action.Action
		persist    bool

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *fmflow.Flow
	}

	tests := []test{
		{
			"normal",

			1,
			uuid.FromStringOrNil("5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0"),
			"test flow",
			"test flow detail",
			[]action.Action{},
			true,
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"persist":true,"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"persist":true}`),
			},
			&fmflow.Flow{
				ID:       uuid.FromStringOrNil("5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0"),
				UserID:   1,
				Name:     "test flow",
				Detail:   "test flow detail",
				Actions:  []fmaction.Action{},
				Persist:  true,
				TMCreate: "2020-09-20 03:23:20.995000",
				TMUpdate: "",
				TMDelete: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMFlowCreate(tt.userID, tt.flowID, tt.flowName, tt.flowDetail, tt.actions, tt.persist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong matchdfdsfd.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestFMFlowGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request", "bin-manager.storage-manager.request")

	type test struct {
		name string

		userID uint64
		flowID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *fmflow.Flow
	}

	tests := []test{
		{
			"normal",

			1,
			uuid.FromStringOrNil("9e0b76f6-0c54-11eb-bd75-1794701ba654"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"158e4b2c-0c55-11eb-b4f2-37c93a78a6a0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/9e0b76f6-0c54-11eb-bd75-1794701ba654",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&fmflow.Flow{
				ID:       uuid.FromStringOrNil("158e4b2c-0c55-11eb-b4f2-37c93a78a6a0"),
				UserID:   1,
				Name:     "test flow",
				Detail:   "test flow detail",
				Actions:  []fmaction.Action{},
				TMCreate: "2020-09-20 03:23:20.995000",
				TMUpdate: "",
				TMDelete: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMFlowGet(tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestFMFlowGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request", "bin-manager.storage-manager.request")

	type test struct {
		name string

		userID    uint64
		pageToken string
		pageSize  uint64

		flowID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []fmflow.Flow
	}

	tests := []test{
		{
			"normal",

			1,
			"2020-09-20 03:23:20.995000",
			10,

			uuid.FromStringOrNil("9e0b76f6-0c54-11eb-bd75-1794701ba654"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"158e4b2c-0c55-11eb-b4f2-37c93a78a6a0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}]`),
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/flows?page_token=%s&page_size=10&user_id=1", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]fmflow.Flow{
				{
					ID:       uuid.FromStringOrNil("158e4b2c-0c55-11eb-b4f2-37c93a78a6a0"),
					UserID:   1,
					Name:     "test flow",
					Detail:   "test flow detail",
					Actions:  []fmaction.Action{},
					TMCreate: "2020-09-20 03:23:20.995000",
					TMUpdate: "",
					TMDelete: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMFlowGets(tt.userID, tt.pageToken, tt.pageSize)
			// res, err := reqHandler.FMFlowGets(tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
