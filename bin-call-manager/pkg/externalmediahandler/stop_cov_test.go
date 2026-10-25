package externalmediahandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/externalmedia"
)

func Test_Stop_errors_cov(t *testing.T) {
	id := uuid.FromStringOrNil("55555555-0001-11f0-8000-000000000001")
	refID := uuid.FromStringOrNil("55555555-0002-11f0-8000-000000000002")

	tests := []struct {
		name string

		refType    externalmedia.ReferenceType
		statusFail bool
		hangupErr  error
		deleteErr  error
		removeErr  error

		expectErr bool
	}{
		{name: "update status error", statusFail: true, refType: externalmedia.ReferenceTypeCall, expectErr: true},
		{name: "hangup error", refType: externalmedia.ReferenceTypeCall, hangupErr: errCov, expectErr: true},
		{name: "delete error", refType: externalmedia.ReferenceTypeCall, deleteErr: errCov, expectErr: true},
		{name: "call remove error is ignored", refType: externalmedia.ReferenceTypeCall, removeErr: errCov},
		{name: "confbridge remove error is ignored", refType: externalmedia.ReferenceTypeConfbridge, removeErr: errCov},
		{name: "unknown reference type", refType: externalmedia.ReferenceType("other")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			em := &externalmedia.ExternalMedia{ID: id, AsteriskID: "ast-1", ChannelID: "ch-1", ReferenceType: tt.refType, ReferenceID: refID}

			if tt.statusFail {
				m.db.EXPECT().ExternalMediaGet(ctx, id).Return(nil, errCov)
			} else {
				m.db.EXPECT().ExternalMediaGet(ctx, id).Return(em, nil).Times(2)
				m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil)
				m.channel.EXPECT().HangingUpWithAsteriskID(ctx, "ast-1", "ch-1", ari.ChannelCauseNormalClearing).Return(tt.hangupErr)
				if tt.hangupErr == nil {
					m.db.EXPECT().ExternalMediaDelete(ctx, id).Return(tt.deleteErr)
					if tt.deleteErr == nil {
						switch tt.refType {
						case externalmedia.ReferenceTypeCall:
							m.db.EXPECT().CallRemoveExternalMediaID(ctx, refID, id).Return(tt.removeErr)
						case externalmedia.ReferenceTypeConfbridge:
							m.db.EXPECT().ConfbridgeRemoveExternalMediaID(ctx, refID, id).Return(tt.removeErr)
						}
					}
				}
			}

			res, err := h.Stop(ctx, id)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: (%v, %v)", tt.expectErr, res, err)
			}
		})
	}
}

func Test_ARIPlaybackFinished_cov(t *testing.T) {
	h, _ := newCovHandler(t)

	if err := h.ARIPlaybackFinished(context.Background(), &bridge.Bridge{}, nil); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}
