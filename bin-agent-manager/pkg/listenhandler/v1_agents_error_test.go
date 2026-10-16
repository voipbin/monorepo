package listenhandler

import (
	"fmt"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_processV1AgentsIDPermissionPut_error(t *testing.T) {
	id := uuid.FromStringOrNil("f1ba04b0-951e-11ee-a0a2-7b8600a1ee45")

	lastAdmin := cerrors.FailedPrecondition(commonoutline.ServiceNameAgentManager, "AGENT_LAST_ADMIN", "The last admin of the customer cannot be removed.")
	notFound := cerrors.NotFound(commonoutline.ServiceNameAgentManager, "AGENT_NOT_FOUND", "The agent was not found.")
	guest := cerrors.InvalidArgument(commonoutline.ServiceNameAgentManager, "AGENT_GUEST_IMMUTABLE", "The guest agent cannot be changed.")

	tests := []struct {
		name string

		data        []byte
		callHandler bool
		handlerErr  error

		expectStatus int
		expectTyped  *cerrors.VoipbinError
	}{
		{
			name:         "last admin is a typed 409",
			data:         []byte(`{"permission":0}`),
			callHandler:  true,
			handlerErr:   lastAdmin,
			expectStatus: 409,
			expectTyped:  lastAdmin,
		},
		{
			name:         "agent not found is a typed 404",
			data:         []byte(`{"permission":0}`),
			callHandler:  true,
			handlerErr:   notFound,
			expectStatus: 404,
			expectTyped:  notFound,
		},
		{
			name:         "guest agent is a typed 400",
			data:         []byte(`{"permission":0}`),
			callHandler:  true,
			handlerErr:   guest,
			expectStatus: 400,
			expectTyped:  guest,
		},
		{
			name:         "unclassified error is 500",
			data:         []byte(`{"permission":0}`),
			callHandler:  true,
			handlerErr:   fmt.Errorf("db connection lost"),
			expectStatus: 500,
		},
		{
			name:         "malformed body stays 400",
			data:         []byte(`{"permission":`),
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				agentHandler: mockAgent,
			}

			if tt.callHandler {
				mockAgent.EXPECT().UpdatePermission(gomock.Any(), id, agent.PermissionNone).Return(nil, tt.handlerErr)
			}

			res, err := h.processRequest(&sock.Request{
				URI:      "/v1/agents/" + id.String() + "/permission",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     tt.data,
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}

			if tt.expectTyped != nil {
				expect, errExpect := cerrors.ToResponse(tt.expectTyped)
				if errExpect != nil {
					t.Fatalf("Could not build the expected response. err: %v", errExpect)
				}
				if res.DataType != expect.DataType || string(res.Data) != string(expect.Data) {
					t.Errorf("Wrong body.\nexpect: %s %s\ngot: %s %s", expect.DataType, expect.Data, res.DataType, res.Data)
				}
			}
		})
	}
}

func Test_processV1AgentsIDDelete_error(t *testing.T) {
	id := uuid.FromStringOrNil("f1ba04b0-951e-11ee-a0a2-7b8600a1ee45")

	lastAdmin := cerrors.FailedPrecondition(commonoutline.ServiceNameAgentManager, "AGENT_LAST_ADMIN", "The last admin of the customer cannot be removed.")
	notFound := cerrors.NotFound(commonoutline.ServiceNameAgentManager, "AGENT_NOT_FOUND", "The agent was not found.")

	tests := []struct {
		name string

		handlerErr error

		expectStatus int
		expectTyped  *cerrors.VoipbinError
	}{
		{
			name:         "last admin is a typed 409",
			handlerErr:   lastAdmin,
			expectStatus: 409,
			expectTyped:  lastAdmin,
		},
		{
			name:         "agent not found is a typed 404",
			handlerErr:   notFound,
			expectStatus: 404,
			expectTyped:  notFound,
		},
		{
			name:         "unclassified error is 500",
			handlerErr:   fmt.Errorf("db connection lost"),
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().Delete(gomock.Any(), id).Return(nil, tt.handlerErr)

			res, err := h.processRequest(&sock.Request{
				URI:    "/v1/agents/" + id.String(),
				Method: sock.RequestMethodDelete,
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}

			if tt.expectTyped != nil {
				expect, errExpect := cerrors.ToResponse(tt.expectTyped)
				if errExpect != nil {
					t.Fatalf("Could not build the expected response. err: %v", errExpect)
				}
				if res.DataType != expect.DataType || string(res.Data) != string(expect.Data) {
					t.Errorf("Wrong body.\nexpect: %s %s\ngot: %s %s", expect.DataType, expect.Data, res.DataType, res.Data)
				}
			}
		})
	}
}
