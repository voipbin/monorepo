package dbhandler

import (
	"context"
	"database/sql/driver"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

func Test_ChannelSetDataItem_cov(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	id := "e1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01"
	sel := `SELECT .* FROM call_channels WHERE id = \?`

	runCovSQLCases(t, []covSQLCase{
		{
			name: "ordinary key",
			do:   func(h *handler) error { return h.ChannelSetDataItem(ctx, id, "key1", "value1") },
			// the JSON path and the value are both bound, never interpolated.
			wantSQL:     "UPDATE call_channels SET data = json_set(data, ?, ?), tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{`$."key1"`, "value1", nil, id},
			cacheSelect: sel,
		},
		{
			name:        "key with path metacharacters is quoted",
			do:          func(h *handler) error { return h.ChannelSetDataItem(ctx, id, "a.b", int64(7)) },
			wantSQL:     "UPDATE call_channels SET data = json_set(data, ?, ?), tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{`$."a.b"`, int64(7), nil, id},
			cacheSelect: sel,
		},
	})
}

// covCreateChannel creates the channel with a mocked clock and returns the handler.
func covChannelHandler(t *testing.T, ch *channel.Channel) (*handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

	mockUtil.EXPECT().TimeNow().Return(testhelper.TimePtr("2020-04-20T03:22:17.995000Z"))
	mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Cond(func(x *channel.Channel) bool { return x != nil && x.ID == ch.ID }))
	if err := h.ChannelCreate(context.Background(), ch); err != nil {
		t.Fatalf("could not create the channel. err: %v", err)
	}

	return h, mockUtil, mockCache
}

func Test_ChannelSetSIPData_cov(t *testing.T) {
	resetTestDB(t)

	ch := &channel.Channel{AsteriskID: "3e:50:6b:43:bb:30", ID: "e2f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01"}
	h, mockUtil, mockCache := covChannelHandler(t, ch)
	ctx := context.Background()

	sipData := map[string]string{"call-id": "cov-sip-call", "user-agent": "cov-agent"}

	mockUtil.EXPECT().TimeNow().Return(testhelper.TimePtr("2020-04-20T03:22:18.995000Z"))
	mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Cond(func(x *channel.Channel) bool { return x != nil && x.ID == ch.ID }))
	if err := h.ChannelSetSIPData(ctx, ch.ID, sipData); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.channelGetFromDB(ctx, ch.ID)
	if err != nil {
		t.Fatalf("could not get the channel. err: %v", err)
	}
	if !reflect.DeepEqual(res.SIPData, sipData) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", sipData, res.SIPData)
	}
}

func Test_ChannelSetPlaybackID_cov(t *testing.T) {
	resetTestDB(t)

	ch := &channel.Channel{AsteriskID: "3e:50:6b:43:bb:30", ID: "e2f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e02"}
	h, mockUtil, mockCache := covChannelHandler(t, ch)
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(testhelper.TimePtr("2020-04-20T03:22:18.995000Z"))
	mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Cond(func(x *channel.Channel) bool { return x != nil && x.ID == ch.ID }))
	if err := h.ChannelSetPlaybackID(ctx, ch.ID, "cov-playback-id"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.channelGetFromDB(ctx, ch.ID)
	if err != nil {
		t.Fatalf("could not get the channel. err: %v", err)
	}
	if res.PlaybackID != "cov-playback-id" {
		t.Errorf("Wrong match. expect: cov-playback-id, got: %v", res.PlaybackID)
	}
}
