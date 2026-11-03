package listenhandler

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/callhandler"
)

func Test_processV1CallsIDParticipantsPost(t *testing.T) {
	callID := uuid.FromStringOrNil("8e5acf96-5555-11ef-e001-000000000001")
	address := commonaddress.Address{
		Type:       commonaddress.TypeExtension,
		Target:     "8e5acf96-5555-11ef-e001-000000000002",
		TargetName: "2002",
	}
	responseGroupcall := &groupcall.Groupcall{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("8e5acf96-5555-11ef-e001-000000000003")},
		CallIDs:  []uuid.UUID{uuid.FromStringOrNil("8e5acf96-5555-11ef-e001-000000000004")},
	}
	expectData, err := json.Marshal(responseGroupcall)
	if err != nil {
		t.Fatalf("could not build the expected data: %v", err)
	}

	tests := []struct {
		name string

		request *sock.Request

		callErr error

		expectCall   bool
		expectStatus int
		expectReason string
		expectData   []byte
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/calls/8e5acf96-5555-11ef-e001-000000000001/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"address":{"type":"extension","target":"8e5acf96-5555-11ef-e001-000000000002","target_name":"2002"}}`),
			},
			expectCall:   true,
			expectStatus: 200,
			expectData:   expectData,
		},
		{
			name: "typed conflict error keeps its status and reason",
			request: &sock.Request{
				URI:      "/v1/calls/8e5acf96-5555-11ef-e001-000000000001/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"address":{"type":"extension","target":"8e5acf96-5555-11ef-e001-000000000002","target_name":"2002"}}`),
			},
			callErr:      cerrors.FailedPrecondition("call-manager", "PARTICIPANT_ADD_IN_PROGRESS", "busy"),
			expectCall:   true,
			expectStatus: 409,
			expectReason: "PARTICIPANT_ADD_IN_PROGRESS",
		},
		{
			name: "untyped error is a 500",
			request: &sock.Request{
				URI:      "/v1/calls/8e5acf96-5555-11ef-e001-000000000001/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"address":{"type":"extension","target":"8e5acf96-5555-11ef-e001-000000000002","target_name":"2002"}}`),
			},
			callErr:      fmt.Errorf("boom"),
			expectCall:   true,
			expectStatus: 500,
		},
		{
			name: "malformed body is rejected before the call handler",
			request: &sock.Request{
				URI:      "/v1/calls/8e5acf96-5555-11ef-e001-000000000001/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"address":`),
			},
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			if tt.expectCall {
				var res *groupcall.Groupcall
				if tt.callErr == nil {
					res = responseGroupcall
				}
				mockCall.EXPECT().AddParticipant(gomock.Any(), callID, address).Return(res, tt.callErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Wrong match. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
			if tt.expectData != nil && string(res.Data) != string(tt.expectData) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectData, res.Data)
			}
			if tt.expectReason != "" {
				var ve cerrors.VoipbinError
				if errUnmarshal := json.Unmarshal(res.Data, &ve); errUnmarshal != nil || ve.Reason != tt.expectReason {
					t.Errorf("Wrong match. expect reason: %s, got: %v (%v)", tt.expectReason, ve.Reason, errUnmarshal)
				}
			}
		})
	}
}

func Test_processV1CallsIDParticipantsPost_bad_uri(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		sockHandler: sockhandler.NewMockSockHandler(mc),
		callHandler: callhandler.NewMockCallHandler(mc),
	}

	// only POST is routed; a GET must not reach the call handler
	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/calls/8e5acf96-5555-11ef-e001-000000000001/participants",
		Method: sock.RequestMethodGet,
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("Wrong match. expect: 404, got: %d", res.StatusCode)
	}
}
