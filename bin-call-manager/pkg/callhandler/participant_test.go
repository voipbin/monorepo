package callhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

var (
	testPartCallID       = uuid.FromStringOrNil("4a1c8b52-1111-11ef-a001-000000000001")
	testPartCustomerID   = uuid.FromStringOrNil("4a1c8b52-1111-11ef-a001-000000000002")
	testPartConfbridgeID = uuid.FromStringOrNil("4a1c8b52-1111-11ef-a001-000000000003")
	testPartMasterCallID = uuid.FromStringOrNil("4a1c8b52-1111-11ef-a001-000000000004")
	testPartFlowID       = uuid.FromStringOrNil("4a1c8b52-1111-11ef-a001-000000000005")
	testPartGroupcallID  = uuid.FromStringOrNil("4a1c8b52-1111-11ef-a001-000000000006")
	testPartExtensionID  = "4a1c8b52-1111-11ef-a001-000000000007"
)

func testPartDestination() commonaddress.Address {
	return commonaddress.Address{
		Type:       commonaddress.TypeExtension,
		Target:     testPartExtensionID,
		TargetName: "2002",
	}
}

func testPartCall() *call.Call {
	return &call.Call{
		Identity: commonidentity.Identity{
			ID:         testPartCallID,
			CustomerID: testPartCustomerID,
		},
		Status:       call.StatusProgressing,
		ConfbridgeID: testPartConfbridgeID,
		MasterCallID: testPartMasterCallID,
		Source:       commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"},
	}
}

func testPartConfbridge(channels int) *confbridge.Confbridge {
	ids := map[string]uuid.UUID{}
	for i := 0; i < channels; i++ {
		ids[fmt.Sprintf("channel-%d", i)] = uuid.Must(uuid.NewV4())
	}

	return &confbridge.Confbridge{
		Identity:       commonidentity.Identity{ID: testPartConfbridgeID, CustomerID: testPartCustomerID},
		Type:           confbridge.TypeConnect,
		Status:         confbridge.StatusProgressing,
		ChannelCallIDs: ids,
	}
}

func testPartFlow() *fmflow.Flow {
	return &fmflow.Flow{Identity: commonidentity.Identity{ID: testPartFlowID, CustomerID: testPartCustomerID}}
}

func testPartAssertReason(t *testing.T, err error, expectReason string, expectStatus cerrors.Status) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: typed error %s, got: %v", expectReason, err)
	}
	if ve.Reason != expectReason || ve.Status != expectStatus {
		t.Errorf("Wrong match. expect: %s/%v, got: %s/%v", expectReason, expectStatus, ve.Reason, ve.Status)
	}
}

type testPartMocks struct {
	util       *utilhandler.MockUtilHandler
	req        *requesthandler.MockRequestHandler
	notify     *notifyhandler.MockNotifyHandler
	db         *dbhandler.MockDBHandler
	confbridge *confbridgehandler.MockConfbridgeHandler
	groupcall  *groupcallhandler.MockGroupcallHandler
}

func testPartHandler(mc *gomock.Controller) (*callHandler, *testPartMocks) {
	m := &testPartMocks{
		util:       utilhandler.NewMockUtilHandler(mc),
		req:        requesthandler.NewMockRequestHandler(mc),
		notify:     notifyhandler.NewMockNotifyHandler(mc),
		db:         dbhandler.NewMockDBHandler(mc),
		confbridge: confbridgehandler.NewMockConfbridgeHandler(mc),
		groupcall:  groupcallhandler.NewMockGroupcallHandler(mc),
	}

	h := &callHandler{
		utilHandler:       m.util,
		reqHandler:        m.req,
		notifyHandler:     m.notify,
		db:                m.db,
		confbridgeHandler: m.confbridge,
		groupcallHandler:  m.groupcall,
	}

	return h, m
}

func Test_AddParticipant(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := testPartHandler(mc)
	ctx := context.Background()
	c := testPartCall()
	dest := testPartDestination()
	expectGroupcall := &groupcall.Groupcall{
		Identity: commonidentity.Identity{ID: testPartGroupcallID, CustomerID: testPartCustomerID},
		CallIDs:  []uuid.UUID{uuid.FromStringOrNil("4a1c8b52-1111-11ef-a001-0000000000a1")},
	}

	expectActions := []fmaction.Action{
		{
			Type: fmaction.TypeConfbridgeJoin,
			Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
				ConfbridgeID: testPartConfbridgeID,
			}),
		},
	}

	// no other expectation is registered: any moh, mute, hold or hangup request to the existing legs, any claim
	// release and any flow delete on the success path fails this test.
	gomock.InOrder(
		m.db.EXPECT().CallGet(ctx, testPartCallID).Return(c, nil),
		m.confbridge.EXPECT().Get(ctx, testPartConfbridgeID).Return(testPartConfbridge(2), nil),
		m.db.EXPECT().ConfbridgeParticipantClaim(ctx, testPartConfbridgeID, participantClaimTTL).Return(true, nil),
		m.req.EXPECT().FlowV1FlowCreate(ctx, testPartCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), expectActions, uuid.Nil, false).Return(testPartFlow(), nil),
		m.groupcall.EXPECT().Start(
			ctx,
			uuid.Nil,
			testPartCustomerID,
			testPartFlowID,
			&c.Source,
			[]commonaddress.Address{dest},
			testPartMasterCallID, // the master of the original call, not the call itself
			uuid.Nil,
			groupcall.RingMethodRingAll,
			groupcall.AnswerMethodHangupOthers,
			"",
			nil,
		).Return(expectGroupcall, nil),
	)

	res, err := h.AddParticipant(ctx, testPartCallID, dest)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != expectGroupcall {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectGroupcall, res)
	}
}

func Test_AddParticipant_master_nil(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := testPartHandler(mc)
	ctx := context.Background()
	c := testPartCall()
	c.MasterCallID = uuid.Nil
	dest := testPartDestination()

	m.db.EXPECT().CallGet(ctx, testPartCallID).Return(c, nil)
	m.confbridge.EXPECT().Get(ctx, testPartConfbridgeID).Return(testPartConfbridge(2), nil)
	m.db.EXPECT().ConfbridgeParticipantClaim(ctx, testPartConfbridgeID, participantClaimTTL).Return(true, nil)
	m.req.EXPECT().FlowV1FlowCreate(ctx, testPartCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(testPartFlow(), nil)
	// an agent call without a master call: the new leg gets no master, the original call is never the master
	m.groupcall.EXPECT().Start(ctx, uuid.Nil, testPartCustomerID, testPartFlowID, &c.Source, []commonaddress.Address{dest}, uuid.Nil, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", nil).Return(&groupcall.Groupcall{}, nil)

	if _, err := h.AddParticipant(ctx, testPartCallID, dest); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_AddParticipant_invalid_address(t *testing.T) {
	tests := []struct {
		name string
		dest commonaddress.Address
	}{
		{"type tel", commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230002", TargetName: "2002"}},
		{"type sip", commonaddress.Address{Type: commonaddress.TypeSIP, Target: "sip:a@203.0.113.1", TargetName: "2002"}},
		{"type agent", commonaddress.Address{Type: commonaddress.TypeAgent, Target: testPartExtensionID, TargetName: "2002"}},
		{"empty target", commonaddress.Address{Type: commonaddress.TypeExtension, Target: "", TargetName: "2002"}},
		{"empty target name", commonaddress.Address{Type: commonaddress.TypeExtension, Target: testPartExtensionID, TargetName: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := testPartHandler(mc)

			// nothing is looked up or created for an invalid address
			_, err := h.AddParticipant(context.Background(), testPartCallID, tt.dest)
			testPartAssertReason(t, err, "INVALID_PARTICIPANT_ADDRESS", cerrors.StatusInvalidArgument)
		})
	}
}

func Test_AddParticipant_call_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := testPartHandler(mc)
	ctx := context.Background()

	m.db.EXPECT().CallGet(ctx, testPartCallID).Return(nil, dbhandler.ErrNotFound)

	_, err := h.AddParticipant(ctx, testPartCallID, testPartDestination())
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: not found, got: %v", err)
	}
}

func Test_AddParticipant_call_not_joinable(t *testing.T) {
	tests := []struct {
		name   string
		modify func(c *call.Call)
	}{
		{"ringing", func(c *call.Call) { c.Status = call.StatusRinging }},
		{"dialing", func(c *call.Call) { c.Status = call.StatusDialing }},
		{"terminating", func(c *call.Call) { c.Status = call.StatusTerminating }},
		{"hangup", func(c *call.Call) { c.Status = call.StatusHangup }},
		{"no confbridge", func(c *call.Call) { c.ConfbridgeID = uuid.Nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := testPartHandler(mc)
			ctx := context.Background()
			c := testPartCall()
			tt.modify(c)

			// no confbridge lookup, claim, flow or groupcall for a call that is not joinable
			m.db.EXPECT().CallGet(ctx, testPartCallID).Return(c, nil)

			_, err := h.AddParticipant(ctx, testPartCallID, testPartDestination())
			testPartAssertReason(t, err, "CALL_NOT_JOINABLE", cerrors.StatusFailedPrecondition)
		})
	}
}

func Test_AddParticipant_confbridge_check(t *testing.T) {
	tests := []struct {
		name         string
		cb           *confbridge.Confbridge
		cbErr        error
		expectReason string
	}{
		{"conference type", func() *confbridge.Confbridge {
			cb := testPartConfbridge(2)
			cb.Type = confbridge.TypeConference
			return cb
		}(), nil, "CALL_NOT_JOINABLE"},
		{"terminating", func() *confbridge.Confbridge {
			cb := testPartConfbridge(2)
			cb.Status = confbridge.StatusTerminating
			return cb
		}(), nil, "CALL_NOT_JOINABLE"},
		{"one participant", testPartConfbridge(1), nil, "PARTICIPANT_COUNT_INVALID"},
		{"zero participants", testPartConfbridge(0), nil, "PARTICIPANT_COUNT_INVALID"},
		{"three participants", testPartConfbridge(3), nil, "PARTICIPANT_COUNT_INVALID"},
		{"four participants", testPartConfbridge(4), nil, "PARTICIPANT_COUNT_INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := testPartHandler(mc)
			ctx := context.Background()

			// the validation fails before the claim: no claim is taken, so nothing is left locked
			m.db.EXPECT().CallGet(ctx, testPartCallID).Return(testPartCall(), nil)
			m.confbridge.EXPECT().Get(ctx, testPartConfbridgeID).Return(tt.cb, tt.cbErr)

			_, err := h.AddParticipant(ctx, testPartCallID, testPartDestination())
			testPartAssertReason(t, err, tt.expectReason, cerrors.StatusFailedPrecondition)
		})
	}
}

func Test_AddParticipant_confbridge_get_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := testPartHandler(mc)
	ctx := context.Background()

	m.db.EXPECT().CallGet(ctx, testPartCallID).Return(testPartCall(), nil)
	m.confbridge.EXPECT().Get(ctx, testPartConfbridgeID).Return(nil, fmt.Errorf("db down"))

	if _, err := h.AddParticipant(ctx, testPartCallID, testPartDestination()); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_AddParticipant_claim(t *testing.T) {
	tests := []struct {
		name       string
		claimed    bool
		claimErr   error
		expectCode string
	}{
		{"held by another request", false, nil, "PARTICIPANT_ADD_IN_PROGRESS"},
		{"cache error is fail-closed", false, fmt.Errorf("redis down"), "PARTICIPANT_ADD_IN_PROGRESS"},
		{"cache error with a true value is still fail-closed", true, fmt.Errorf("redis down"), "PARTICIPANT_ADD_IN_PROGRESS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := testPartHandler(mc)
			ctx := context.Background()

			// the claim is not ours: no release, no flow and no groupcall
			m.db.EXPECT().CallGet(ctx, testPartCallID).Return(testPartCall(), nil)
			m.confbridge.EXPECT().Get(ctx, testPartConfbridgeID).Return(testPartConfbridge(2), nil)
			m.db.EXPECT().ConfbridgeParticipantClaim(ctx, testPartConfbridgeID, participantClaimTTL).Return(tt.claimed, tt.claimErr)

			_, err := h.AddParticipant(ctx, testPartCallID, testPartDestination())
			testPartAssertReason(t, err, tt.expectCode, cerrors.StatusFailedPrecondition)
		})
	}
}

func Test_AddParticipant_flow_create_error_releases_claim(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := testPartHandler(mc)
	ctx := context.Background()

	gomock.InOrder(
		m.db.EXPECT().CallGet(ctx, testPartCallID).Return(testPartCall(), nil),
		m.confbridge.EXPECT().Get(ctx, testPartConfbridgeID).Return(testPartConfbridge(2), nil),
		m.db.EXPECT().ConfbridgeParticipantClaim(ctx, testPartConfbridgeID, participantClaimTTL).Return(true, nil),
		m.req.EXPECT().FlowV1FlowCreate(ctx, testPartCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("flow-manager down")),
		m.db.EXPECT().ConfbridgeParticipantRelease(gomock.Any(), testPartConfbridgeID).Return(nil),
	)

	if _, err := h.AddParticipant(ctx, testPartCallID, testPartDestination()); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_AddParticipant_start_error_releases_claim_and_deletes_flow(t *testing.T) {
	tests := []struct {
		name          string
		startErr      error
		releaseErr    error
		flowDeleteErr error
		expectReason  string
	}{
		{
			name: "extension not registered keeps its reason",
			startErr: cerrors.FailedPrecondition(
				"call-manager", groupcallhandler.ReasonExtensionNotRegistered, "not registered"),
			expectReason: groupcallhandler.ReasonExtensionNotRegistered,
		},
		{
			name:         "generic error",
			startErr:     fmt.Errorf("could not create groupcall"),
			expectReason: "",
		},
		{
			name:         "release failure is only logged",
			startErr:     fmt.Errorf("could not create groupcall"),
			releaseErr:   fmt.Errorf("redis down"),
			expectReason: "",
		},
		{
			name:          "flow delete failure is only logged",
			startErr:      fmt.Errorf("could not create groupcall"),
			flowDeleteErr: fmt.Errorf("flow-manager down"),
			expectReason:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := testPartHandler(mc)
			ctx := context.Background()
			c := testPartCall()

			gomock.InOrder(
				m.db.EXPECT().CallGet(ctx, testPartCallID).Return(c, nil),
				m.confbridge.EXPECT().Get(ctx, testPartConfbridgeID).Return(testPartConfbridge(2), nil),
				m.db.EXPECT().ConfbridgeParticipantClaim(ctx, testPartConfbridgeID, participantClaimTTL).Return(true, nil),
				m.req.EXPECT().FlowV1FlowCreate(ctx, testPartCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(testPartFlow(), nil),
				m.groupcall.EXPECT().Start(ctx, uuid.Nil, testPartCustomerID, testPartFlowID, &c.Source, gomock.Any(), testPartMasterCallID, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", nil).Return(nil, tt.startErr),
			)
			m.db.EXPECT().ConfbridgeParticipantRelease(gomock.Any(), testPartConfbridgeID).Return(tt.releaseErr)
			m.req.EXPECT().FlowV1FlowDelete(gomock.Any(), testPartFlowID).Return(testPartFlow(), tt.flowDeleteErr)

			_, err := h.AddParticipant(ctx, testPartCallID, testPartDestination())
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectReason != "" {
				testPartAssertReason(t, err, tt.expectReason, cerrors.StatusFailedPrecondition)
			}
		})
	}
}

// The release and the flow delete must not depend on the request context: a canceled request still has to free the claim.
func Test_AddParticipant_release_survives_canceled_context(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := testPartHandler(mc)
	ctx, cancel := context.WithCancel(context.Background())
	c := testPartCall()

	m.db.EXPECT().CallGet(gomock.Any(), testPartCallID).Return(c, nil)
	m.confbridge.EXPECT().Get(gomock.Any(), testPartConfbridgeID).Return(testPartConfbridge(2), nil)
	m.db.EXPECT().ConfbridgeParticipantClaim(gomock.Any(), testPartConfbridgeID, participantClaimTTL).Return(true, nil)
	m.req.EXPECT().FlowV1FlowCreate(gomock.Any(), testPartCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(testPartFlow(), nil)
	m.groupcall.EXPECT().Start(gomock.Any(), uuid.Nil, testPartCustomerID, testPartFlowID, &c.Source, gomock.Any(), testPartMasterCallID, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", nil).DoAndReturn(
		func(ctx context.Context, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID, _ *commonaddress.Address, _ []commonaddress.Address, _ uuid.UUID, _ uuid.UUID, _ groupcall.RingMethod, _ groupcall.AnswerMethod, _ string, _ map[string]string) (*groupcall.Groupcall, error) {
			cancel() // the request is canceled while the groupcall is being started
			return nil, fmt.Errorf("canceled")
		})
	m.db.EXPECT().ConfbridgeParticipantRelease(gomock.Any(), testPartConfbridgeID).DoAndReturn(
		func(ctx context.Context, _ uuid.UUID) error {
			if ctx.Err() != nil {
				t.Errorf("Wrong match. the release ran with a canceled context: %v", ctx.Err())
			}
			return nil
		})
	m.req.EXPECT().FlowV1FlowDelete(gomock.Any(), testPartFlowID).DoAndReturn(
		func(ctx context.Context, _ uuid.UUID) (*fmflow.Flow, error) {
			if ctx.Err() != nil {
				t.Errorf("Wrong match. the flow delete ran with a canceled context: %v", ctx.Err())
			}
			return testPartFlow(), nil
		})

	if _, err := h.AddParticipant(ctx, testPartCallID, testPartDestination()); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
