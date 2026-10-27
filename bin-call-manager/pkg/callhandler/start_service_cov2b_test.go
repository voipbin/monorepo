package callhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/callapplication"
	"monorepo/bin-call-manager/models/channel"
)

func Test_startServiceFromAMD_cov2b_asyncAndSync(t *testing.T) {
	callID := uuid.FromStringOrNil("ee1b2c3d-0000-4000-8000-00000000c2b1")
	const chID = "ch-cov2b-amd"

	tests := []struct {
		name       string
		async      bool
		expectNext bool
	}{
		{name: "synchronous amd moves the call to the next action", async: false, expectNext: true},
		{name: "asynchronous amd leaves the call alone", async: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			m.db.EXPECT().CallApplicationAMDGet(ctx, chID).Return(&callapplication.AMD{CallID: callID, Async: tt.async}, nil)
			if tt.expectNext {
				m.req.EXPECT().CallV1CallActionNext(ctx, callID, false).Return(nil)
			}
			m.channel.EXPECT().HangingUp(ctx, chID, ari.ChannelCauseNormalClearing).Return(nil, nil)

			if err := h.startServiceFromAMD(ctx, chID, map[channel.StasisDataType]string{channel.StasisDataTypeServiceAMDStatus: "person"}); err != nil {
				t.Errorf("Wrong error. expect: nil, got: %v", err)
			}
		})
	}
}

func Test_startServiceFromDefault_cov2b_actionNext(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()
	callID := uuid.FromStringOrNil("ee1b2c3d-0000-4000-8000-00000000c2b2")

	// the call has no activeflow, so ActionNext hangs the call up
	c := &call.Call{Identity: commonidentity.Identity{ID: callID}, Status: call.StatusProgressing}
	m.db.EXPECT().CallGetByChannelID(ctx, "ch-cov2b").Return(c, nil)
	m.db.EXPECT().CallGet(ctx, callID).Return(nil, context.Canceled)

	if err := h.startServiceFromDefault(ctx, "ch-cov2b", nil); err != nil {
		t.Errorf("Wrong error. expect: nil, got: %v", err)
	}
}
