package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_ProcessV1LoginPost(t *testing.T) {

	tests := []struct {
		name     string
		request  *rabbitmqhandler.Request
		username string
		password string

		responseAgent *agent.Agent
		expectRes     *rabbitmqhandler.Response
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/login",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"username":"test@test.com","password":"password"}`),
			},
			username: "test@test.com",
			password: "password",

			responseAgent: &agent.Agent{
				ID:       uuid.FromStringOrNil("e58a9424-7dc0-11ec-82b6-d387115f2157"),
				Username: "test@test.com",
			},
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e58a9424-7dc0-11ec-82b6-d387115f2157","customer_id":"00000000-0000-0000-0000-000000000000","username":"test@test.com","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCustomer := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockCustomer,
			}

			mockCustomer.EXPECT().Login(gomock.Any(), tt.username, tt.password).Return(tt.responseAgent, nil)

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
