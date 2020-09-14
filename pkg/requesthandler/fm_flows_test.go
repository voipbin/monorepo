package requesthandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/rabbitmq"
	rabbitmodels "gitlab.com/voipbin/bin-manager/api-manager.git/pkg/rabbitmq/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
)

func TestFMFlowCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	type test struct {
		name string

		userID     uint64
		flowID     uuid.UUID
		flowName   string
		flowDetail string
		actions    []action.Action
		persist    bool

		response *rabbitmodels.Response

		expectTarget  string
		expectRequest *rabbitmodels.Request
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
			&rabbitmodels.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"persist":true,"tm_create":"2020-09-20T03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.flow-manager.request",
			&rabbitmodels.Request{
				URI:      "/v1/flows",
				Method:   rabbitmodels.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"persist":true}`),
			},
			&fmflow.Flow{
				ID:       uuid.FromStringOrNil("5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0"),
				UserID:   1,
				Name:     "test flow",
				Detail:   "test flow detail",
				Actions:  []action.Action{},
				Persist:  true,
				TMCreate: "2020-09-20T03:23:20.995000",
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
