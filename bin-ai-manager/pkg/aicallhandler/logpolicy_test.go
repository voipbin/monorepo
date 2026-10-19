package aicallhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cmdtmf "monorepo/bin-call-manager/models/dtmf"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

const logPolicySentinel = "SENTINEL-CONVERSATION-TEXT-9f3c"

// The dtmf line must not carry the digit the caller pressed (a digit can be a PIN), and must keep the
// aicall id and the message length.
func Test_logPolicy_EventCMDTMFReceived(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)
	h := &aicallHandler{reqHandler: mockReq, db: mockDB, messageHandler: mockMessage}
	ctx := context.Background()

	callID := uuid.Must(uuid.NewV4())
	pcID := uuid.Must(uuid.NewV4())
	ac := &aicall.AIcall{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, Status: aicall.StatusTerminating, PipecatcallID: pcID}

	mockDB.EXPECT().AIcallGetByReferenceID(ctx, callID).Return(ac, nil)
	mockReq.EXPECT().PipecatV1PipecatcallGet(ctx, pcID).Return(&pmpipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: pcID}, HostID: "1.2.3.4"}, nil)
	mockMessage.EXPECT().Create(ctx, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any(), message.DirectionOutgoing, message.RoleUser, gomock.Any(), nil, "", gomock.Any()).
		Return(&message.Message{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}}, nil)
	mockReq.EXPECT().PipecatV1Ping(gomock.Any(), "1.2.3.4").Return(nil)
	mockReq.EXPECT().PipecatV1MessageSend(ctx, "1.2.3.4", pcID, gomock.Any(), gomock.Any(), true, true).Return(nil, nil)

	h.EventCMDTMFReceived(ctx, &cmdtmf.DTMF{CallID: callID, Digit: logPolicySentinel, Duration: 100})

	logs.AssertAbsent(t, logPolicySentinel)
	logs.AssertField(t, "message_len")
}

// The shared logger of ToolHandle must carry the length of the tool arguments, not the arguments, which
// the model writes from the conversation.
func Test_logPolicy_ToolHandle_arguments(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)
	h := &aicallHandler{db: mockDB, messageHandler: mockMessage}
	ctx := context.Background()

	aicallID := uuid.Must(uuid.NewV4())
	mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(&aicall.AIcall{
		Identity:       commonidentity.Identity{ID: aicallID},
		AssistanceType: aicall.AssistanceTypeAI,
		ReferenceType:  aicall.ReferenceTypeCall,
	}, nil)
	mockMessage.EXPECT().Create(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&message.Message{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}}, nil).AnyTimes()
	mockMessage.EXPECT().Create(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&message.Message{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}}, nil).AnyTimes()

	// an unknown tool name reaches log lines of the shared logger without any other service
	_, err := h.ToolHandle(ctx, aicallID, "tool-1", message.ToolTypeFunction,
		message.FunctionCall{Name: "no_such_tool", Arguments: `{"text": "` + logPolicySentinel + `"}`}, uuid.Nil)
	if err == nil {
		t.Fatalf("expected the unknown tool to fail")
	}

	logs.AssertAbsent(t, logPolicySentinel)
	logs.AssertField(t, "arguments_len")
}
