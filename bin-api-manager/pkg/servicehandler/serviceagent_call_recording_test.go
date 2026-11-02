package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"
)

var (
	recCustomerID      = uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9")
	recOtherCustomerID = uuid.FromStringOrNil("6e27823d-3b9f-11ef-8a51-f30f1e2ce1e9")
	recAgentID         = uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a")
	recOtherAgentID    = uuid.FromStringOrNil("7f38934e-3b9f-11ef-98ac-db226570f09a")
	recCallID          = uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c")
	recRecordingID     = uuid.FromStringOrNil("3c8e1f20-7a4b-4d6e-9b1a-5f2c8d0e7a31")
)

func recAgentIdentity() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: recAgentID, CustomerID: recCustomerID},
		Permission: amagent.PermissionCustomerAgent,
	})
}

func recCall(owner, customer uuid.UUID, status cmcall.Status, recordingID uuid.UUID) *cmcall.Call {
	return &cmcall.Call{
		Identity:    commonidentity.Identity{ID: recCallID, CustomerID: customer},
		OwnerID:     owner,
		Status:      status,
		RecordingID: recordingID,
	}
}

func recIntPtr(v int) *int { return &v }

func recStrPtr(v string) *string { return &v }

func Test_ServiceAgentCallRecordingStart(t *testing.T) {
	wav := cmrecording.FormatWAV
	flac := cmrecording.Format("flac")
	errRPC := errors.New("rpc failed")

	tests := []struct {
		name string

		agent        *auth.AuthIdentity
		format       *cmrecording.Format
		endOfSilence *int
		endOfKey     *string
		duration     *int

		responseCall *cmcall.Call
		callRPC      bool // whether CallV1CallRecordingStart is expected
		rpcResponse  *cmcall.Call
		rpcErr       error

		expectFormat   cmrecording.Format
		expectSilence  int
		expectKey      string
		expectDuration int

		expectErr error
	}{
		{
			name:           "all omitted: defaults are filled (duration 7200)",
			agent:          recAgentIdentity(),
			responseCall:   recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			callRPC:        true,
			rpcResponse:    recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			expectFormat:   cmrecording.FormatWAV,
			expectDuration: 7200,
		},
		{
			name:           "explicit values are forwarded",
			agent:          recAgentIdentity(),
			format:         &wav,
			endOfSilence:   recIntPtr(10),
			endOfKey:       recStrPtr("#"),
			duration:       recIntPtr(3600),
			responseCall:   recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			callRPC:        true,
			rpcResponse:    recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			expectFormat:   cmrecording.FormatWAV,
			expectSilence:  10,
			expectKey:      "#",
			expectDuration: 3600,
		},
		{
			name:           "duration 7200 is allowed, end_of_silence 0 is allowed",
			agent:          recAgentIdentity(),
			endOfSilence:   recIntPtr(0),
			duration:       recIntPtr(7200),
			responseCall:   recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			callRPC:        true,
			rpcResponse:    recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			expectFormat:   cmrecording.FormatWAV,
			expectDuration: 7200,
		},
		{
			name:           "end_of_silence has no upper bound",
			agent:          recAgentIdentity(),
			endOfSilence:   recIntPtr(100000),
			responseCall:   recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			callRPC:        true,
			rpcResponse:    recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			expectFormat:   cmrecording.FormatWAV,
			expectSilence:  100000,
			expectDuration: 7200,
		},
		{
			name:         "explicit duration 0 is rejected (call-manager treats 0 as unlimited)",
			agent:        recAgentIdentity(),
			duration:     recIntPtr(0),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrInvalidArgument,
		},
		{
			name:         "negative duration is rejected",
			agent:        recAgentIdentity(),
			duration:     recIntPtr(-1),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrInvalidArgument,
		},
		{
			name:         "duration 7201 is rejected",
			agent:        recAgentIdentity(),
			duration:     recIntPtr(7201),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrInvalidArgument,
		},
		{
			name:         "negative end_of_silence is rejected",
			agent:        recAgentIdentity(),
			endOfSilence: recIntPtr(-1),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrInvalidArgument,
		},
		{
			name:         "unsupported format is rejected",
			agent:        recAgentIdentity(),
			format:       &flac,
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrInvalidArgument,
		},
		{
			name:         "ringing call is rejected with invalid argument and is not answered",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusRinging, uuid.Nil),
			expectErr:    serviceerrors.ErrInvalidArgument,
		},
		{
			name:         "already recording is a state conflict",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			expectErr:    serviceerrors.ErrStateInvalid,
		},
		{
			name:         "call owned by another agent is denied",
			agent:        recAgentIdentity(),
			responseCall: recCall(recOtherAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call without an owner is denied (fail-closed)",
			agent:        recAgentIdentity(),
			responseCall: recCall(uuid.Nil, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call of another customer is denied",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recOtherCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call-manager failure is returned",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			callRPC:      true,
			rpcErr:       errRPC,
			expectFormat: cmrecording.FormatWAV, expectDuration: 7200,
			expectErr: errRPC,
		},
		{
			name:         "empty call-manager response is an internal error",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			callRPC:      true,
			expectFormat: cmrecording.FormatWAV, expectDuration: 7200,
			expectErr: serviceerrors.ErrInternal,
		},
		{
			name:      "not an agent identity",
			agent:     auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{}),
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.responseCall != nil {
				mockReq.EXPECT().CallV1CallGet(ctx, recCallID).Return(tt.responseCall, nil)
			}
			if tt.callRPC {
				// on_end_flow_id is always uuid.Nil on the agent path.
				mockReq.EXPECT().CallV1CallRecordingStart(ctx, recCallID, tt.expectFormat, tt.expectSilence, tt.expectKey, tt.expectDuration, uuid.Nil).Return(tt.rpcResponse, tt.rpcErr)
			}

			res, err := h.ServiceAgentCallRecordingStart(ctx, tt.agent, recCallID, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration)
			if tt.expectErr != nil {
				if err == nil || !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil result on error, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			expect := tt.rpcResponse.ConvertWebhookMessage()
			if !reflect.DeepEqual(expect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
			}
		})
	}
}

func Test_ServiceAgentCallRecordingStop(t *testing.T) {
	errRPC := errors.New("rpc failed")

	tests := []struct {
		name string

		agent *auth.AuthIdentity

		responseCall *cmcall.Call
		callRPC      bool
		rpcResponse  *cmcall.Call
		rpcErr       error

		expectErr error
	}{
		{
			name:         "owner stops the recording whoever started it",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			callRPC:      true,
			rpcResponse:  recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
		},
		{
			name:         "no ongoing recording is a state conflict",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil),
			expectErr:    serviceerrors.ErrStateInvalid,
		},
		{
			name:         "call is not progressing",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusHangup, recRecordingID),
			expectErr:    serviceerrors.ErrStateInvalid,
		},
		{
			name:         "call owned by another agent is denied",
			agent:        recAgentIdentity(),
			responseCall: recCall(recOtherAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call without an owner is denied (fail-closed)",
			agent:        recAgentIdentity(),
			responseCall: recCall(uuid.Nil, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call of another customer is denied",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recOtherCustomerID, cmcall.StatusProgressing, recRecordingID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call-manager failure is returned",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			callRPC:      true,
			rpcErr:       errRPC,
			expectErr:    errRPC,
		},
		{
			name:         "empty call-manager response is an internal error",
			agent:        recAgentIdentity(),
			responseCall: recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID),
			callRPC:      true,
			expectErr:    serviceerrors.ErrInternal,
		},
		{
			name:      "not an agent identity",
			agent:     auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{}),
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.responseCall != nil {
				mockReq.EXPECT().CallV1CallGet(ctx, recCallID).Return(tt.responseCall, nil)
			}
			if tt.callRPC {
				mockReq.EXPECT().CallV1CallRecordingStop(ctx, recCallID).Return(tt.rpcResponse, tt.rpcErr)
			}

			res, err := h.ServiceAgentCallRecordingStop(ctx, tt.agent, recCallID)
			if tt.expectErr != nil {
				if err == nil || !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil result on error, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			expect := tt.rpcResponse.ConvertWebhookMessage()
			if !reflect.DeepEqual(expect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
			}
		})
	}
}

// Test_ServiceAgentCallRecording_auditLog checks that start and stop each leave one audit entry
// with the agent id, the call id and nothing about the other party.
func Test_ServiceAgentCallRecording_auditLog(t *testing.T) {
	tests := []struct {
		name string
		run  func(h *serviceHandler, ctx context.Context) error
		rpc  func(mockReq *requesthandler.MockRequestHandler, ctx context.Context)
		want string
	}{
		{
			name: "start",
			run: func(h *serviceHandler, ctx context.Context) error {
				_, err := h.ServiceAgentCallRecordingStart(ctx, recAgentIdentity(), recCallID, nil, nil, nil, nil)
				return err
			},
			rpc: func(mockReq *requesthandler.MockRequestHandler, ctx context.Context) {
				mockReq.EXPECT().CallV1CallGet(ctx, recCallID).Return(recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil), nil)
				mockReq.EXPECT().CallV1CallRecordingStart(ctx, recCallID, cmrecording.FormatWAV, 0, "", 7200, uuid.Nil).Return(recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID), nil)
			},
			want: "Audit: the agent starts the call recording.",
		},
		{
			name: "stop",
			run: func(h *serviceHandler, ctx context.Context) error {
				_, err := h.ServiceAgentCallRecordingStop(ctx, recAgentIdentity(), recCallID)
				return err
			},
			rpc: func(mockReq *requesthandler.MockRequestHandler, ctx context.Context) {
				mockReq.EXPECT().CallV1CallGet(ctx, recCallID).Return(recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, recRecordingID), nil)
				mockReq.EXPECT().CallV1CallRecordingStop(ctx, recCallID).Return(recCall(recAgentID, recCustomerID, cmcall.StatusProgressing, uuid.Nil), nil)
			},
			want: "Audit: the agent stops the call recording.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := logrustest.NewLocal(logrus.StandardLogger())
			t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()
			tt.rpc(mockReq, ctx)

			if err := tt.run(h, ctx); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			found := 0
			for _, entry := range hook.AllEntries() {
				if entry.Message != tt.want {
					continue
				}
				found++

				// only these fields: no address, number or name of the other party.
				if len(entry.Data) != 3 {
					t.Errorf("Wrong fields. expect: func, agent_id, call_id, got: %v", entry.Data)
				}
				if entry.Data["agent_id"] != recAgentID {
					t.Errorf("Wrong agent_id. expect: %v, got: %v", recAgentID, entry.Data["agent_id"])
				}
				if entry.Data["call_id"] != recCallID {
					t.Errorf("Wrong call_id. expect: %v, got: %v", recCallID, entry.Data["call_id"])
				}
			}
			if found != 1 {
				t.Errorf("Wrong audit entries. expect: 1, got: %d", found)
			}
		})
	}
}
