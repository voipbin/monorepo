package dbhandler

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

func Test_ConfbridgeJSONFunctions_cov(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("f1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01")
	otherID := uuid.FromStringOrNil("f1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e02")
	channelID := "f1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e03"
	sel := `SELECT .* FROM call_confbridges WHERE id = \?`

	runCovSQLCases(t, []covSQLCase{
		{
			name: "ConfbridgeAddRecordingIDs",
			do:   func(h *handler) error { return h.ConfbridgeAddRecordingIDs(ctx, id, otherID) },
			wantSQL: "UPDATE call_confbridges SET recording_ids = json_array_append(recording_ids, '$', ?), " +
				"tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
		{
			name: "ConfbridgeAddExternalMediaID",
			do:   func(h *handler) error { return h.ConfbridgeAddExternalMediaID(ctx, id, otherID) },
			wantSQL: "UPDATE call_confbridges SET external_media_ids = json_array_append(external_media_ids, '$', ?), " +
				"tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
		{
			name: "ConfbridgeRemoveExternalMediaID",
			do:   func(h *handler) error { return h.ConfbridgeRemoveExternalMediaID(ctx, id, otherID) },
			wantSQL: "UPDATE call_confbridges SET external_media_ids = IF(json_search(external_media_ids, 'one', ?) IS NOT NULL, " +
				"json_remove(external_media_ids, replace(json_search(external_media_ids, 'one', ?), '\"', '')), " +
				"external_media_ids), tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{otherID.String(), otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
		{
			name: "ConfbridgeAddChannelCallID",
			do:   func(h *handler) error { return h.ConfbridgeAddChannelCallID(ctx, id, channelID, otherID) },
			wantSQL: "UPDATE call_confbridges SET channel_call_ids = json_insert(channel_call_ids, ?, ?), " +
				"tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{`$."` + channelID + `"`, otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
		{
			name: "ConfbridgeRemoveChannelCallID",
			do:   func(h *handler) error { return h.ConfbridgeRemoveChannelCallID(ctx, id, channelID) },
			wantSQL: "UPDATE call_confbridges SET channel_call_ids = json_remove(channel_call_ids, ?), " +
				"tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{`$."` + channelID + `"`, nil, id.Bytes()},
			cacheSelect: sel,
		},
	})
}

func covConfbridgeHandler(t *testing.T, c *confbridge.Confbridge) (*handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

	mockUtil.EXPECT().TimeNow().Return(testhelper.TimePtr("2023-01-18T03:22:18.995000Z"))
	mockCache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Cond(func(x *confbridge.Confbridge) bool { return x != nil && x.ID == c.ID }))
	if err := h.ConfbridgeCreate(context.Background(), c); err != nil {
		t.Fatalf("could not create the confbridge. err: %v", err)
	}

	return h, mockUtil, mockCache
}

func Test_ConfbridgeSetBridgeID_cov(t *testing.T) {
	c := &confbridge.Confbridge{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("f2f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01")}}
	h, mockUtil, mockCache := covConfbridgeHandler(t, c)
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(testhelper.TimePtr("2023-01-18T03:22:19.995000Z"))
	mockCache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Cond(func(x *confbridge.Confbridge) bool { return x != nil && x.ID == c.ID }))
	if err := h.ConfbridgeSetBridgeID(ctx, c.ID, "cov-bridge-id"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.confbridgeGetFromDB(ctx, c.ID)
	if err != nil {
		t.Fatalf("could not get the confbridge. err: %v", err)
	}
	if res.BridgeID != "cov-bridge-id" {
		t.Errorf("Wrong match. expect: cov-bridge-id, got: %v", res.BridgeID)
	}
}

func Test_ConfbridgeDelete_cov(t *testing.T) {
	c := &confbridge.Confbridge{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("f2f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e02")}}
	h, mockUtil, mockCache := covConfbridgeHandler(t, c)
	ctx := context.Background()

	deleted := testhelper.TimePtr("2023-01-18T03:22:19.995000Z")
	mockUtil.EXPECT().TimeNow().Return(deleted).MinTimes(1)
	mockCache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Cond(func(x *confbridge.Confbridge) bool { return x != nil && x.ID == c.ID }))
	if err := h.ConfbridgeDelete(ctx, c.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.confbridgeGetFromDB(ctx, c.ID)
	if err != nil {
		t.Fatalf("could not get the confbridge. err: %v", err)
	}
	if res.TMDelete == nil || !res.TMDelete.Equal(*deleted) {
		t.Errorf("Wrong match. expect tm_delete: %v, got: %v", deleted, res.TMDelete)
	}
}
