package requesthandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_AgentV1Login(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *amagent.Agent
	}{
		{
			"normal",

			"test@test.com",
			"testpassword",

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/login",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"username":"test@test.com","password":"testpassword"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"805adc04-8a2e-11ee-8548-57e4277837c0"}`),
			},
			&amagent.Agent{
				ID: uuid.FromStringOrNil("805adc04-8a2e-11ee-8548-57e4277837c0"),
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AgentV1Login(ctx, requestTimeoutDefault, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
