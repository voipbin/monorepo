package messagehandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	nmnumber "monorepo/bin-number-manager/models/number"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/telnyx"
	"monorepo/bin-message-manager/pkg/dbhandler"
)

type inboundFixture struct {
	h *messageHandler

	mockUtil    *utilhandler.MockUtilHandler
	mockDB      *dbhandler.MockDBHandler
	mockNotify  *notifyhandler.MockNotifyHandler
	mockReq     *requesthandler.MockRequestHandler
	mockFetcher *MockMediaFetcher

	customerID uuid.UUID
	messageID  uuid.UUID
}

func newInboundFixture(t *testing.T, mc *gomock.Controller) *inboundFixture {
	t.Helper()

	f := &inboundFixture{
		mockUtil:    utilhandler.NewMockUtilHandler(mc),
		mockDB:      dbhandler.NewMockDBHandler(mc),
		mockNotify:  notifyhandler.NewMockNotifyHandler(mc),
		mockReq:     requesthandler.NewMockRequestHandler(mc),
		mockFetcher: NewMockMediaFetcher(mc),

		customerID: uuid.FromStringOrNil("5b0f7e46-1a2b-4c3d-8e9f-000000000001"),
		messageID:  uuid.FromStringOrNil("5b0f7e46-1a2b-4c3d-8e9f-000000000002"),
	}

	f.h = &messageHandler{
		utilHandler:   f.mockUtil,
		db:            f.mockDB,
		notifyHandler: f.mockNotify,
		reqHandler:    f.mockReq,

		mediaFetcher: f.mockFetcher,
		ingestSlots:  make(chan struct{}, inboundMediaMaxIngest),
	}

	return f
}

func (f *inboundFixture) message() *message.Message {
	return &message.Message{
		Identity: commonidentity.Identity{ID: f.messageID, CustomerID: f.customerID},
		Type:     message.TypeMMS,
		Medias:   []string{},
	}
}

func counterValue(result FetchResult) float64 {
	return testutil.ToFloat64(promInboundMediaTotal.WithLabelValues(string(result)))
}

func Test_ingestInboundMedias(t *testing.T) {
	ctx := context.Background()

	fileA := uuid.FromStringOrNil("5b0f7e46-1a2b-4c3d-8e9f-0000000000a1")
	fileC := uuid.FromStringOrNil("5b0f7e46-1a2b-4c3d-8e9f-0000000000a3")

	mkMedia := func(n int) []telnyx.PayloadMedia {
		res := []telnyx.PayloadMedia{}
		for i := 0; i < n; i++ {
			res = append(res, telnyx.PayloadMedia{URL: "https://media.telnyx.com/example/" + string(rune('a'+i)), ContentType: "image/jpeg", Size: 10})
		}
		return res
	}

	type fetchStep struct {
		result  FetchResult
		content string
		budget  int64
	}

	tests := []struct {
		name string

		medias []telnyx.PayloadMedia
		steps  []fetchStep // one per Fetch call, in order
		// storeResults has one entry per fetched media (result ok): the file id or an error.
		storeIDs []uuid.UUID
		storeErr []bool

		expectMedias  []string
		expectCounts  map[FetchResult]float64
		expectUpdated bool
	}{
		{
			name:   "all stored in order",
			medias: mkMedia(2),
			steps: []fetchStep{
				{result: FetchResultOK, content: "AAAA", budget: inboundMediaMaxTotalSize},
				{result: FetchResultOK, content: "BB", budget: inboundMediaMaxTotalSize - 4},
			},
			storeIDs:      []uuid.UUID{fileA, fileC},
			storeErr:      []bool{false, false},
			expectMedias:  []string{fileA.String(), fileC.String()},
			expectCounts:  map[FetchResult]float64{FetchResultStored: 2},
			expectUpdated: true,
		},
		{
			name:   "a failed media is skipped and the others are kept",
			medias: mkMedia(3),
			steps: []fetchStep{
				{result: FetchResultOK, content: "AAAA", budget: inboundMediaMaxTotalSize},
				{result: FetchResultHost, budget: inboundMediaMaxTotalSize - 4},
				{result: FetchResultOK, content: "CC", budget: inboundMediaMaxTotalSize - 4},
			},
			storeIDs:      []uuid.UUID{fileA, fileC},
			storeErr:      []bool{false, false},
			expectMedias:  []string{fileA.String(), fileC.String()},
			expectCounts:  map[FetchResult]float64{FetchResultStored: 2, FetchResultHost: 1},
			expectUpdated: true,
		},
		{
			name:   "storage error is counted and the budget is not spent",
			medias: mkMedia(3),
			steps: []fetchStep{
				{result: FetchResultOK, content: "AAAA", budget: inboundMediaMaxTotalSize},
				{result: FetchResultOK, content: "BB", budget: inboundMediaMaxTotalSize},
				{result: FetchResultSize, budget: inboundMediaMaxTotalSize - 2},
			},
			storeIDs:      []uuid.UUID{uuid.Nil, fileC},
			storeErr:      []bool{true, false},
			expectMedias:  []string{fileC.String()},
			expectCounts:  map[FetchResult]float64{FetchResultStored: 1, FetchResultStorage: 1, FetchResultSize: 1},
			expectUpdated: true,
		},
		{
			name:         "nothing stored writes nothing",
			medias:       mkMedia(2),
			steps:        []fetchStep{{result: FetchResultAuth, budget: inboundMediaMaxTotalSize}, {result: FetchResultStatus, budget: inboundMediaMaxTotalSize}},
			expectCounts: map[FetchResult]float64{FetchResultAuth: 1, FetchResultStatus: 1},
		},
		{
			name:   "more than the maximum count are ignored",
			medias: mkMedia(inboundMediaMaxCount + 3),
			steps: func() []fetchStep {
				res := []fetchStep{}
				for i := 0; i < inboundMediaMaxCount; i++ {
					res = append(res, fetchStep{result: FetchResultNetwork, budget: inboundMediaMaxTotalSize})
				}
				return res
			}(),
			expectCounts: map[FetchResult]float64{FetchResultNetwork: inboundMediaMaxCount, FetchResultOverflow: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newInboundFixture(t, mc)
			m := f.message()

			before := map[FetchResult]float64{}
			for _, r := range []FetchResult{FetchResultStored, FetchResultHost, FetchResultStorage, FetchResultSize, FetchResultAuth, FetchResultStatus, FetchResultNetwork, FetchResultOverflow, FetchResultTimeout} {
				before[r] = counterValue(r)
			}

			storeIdx := 0
			var calls []*gomock.Call
			for i, step := range tt.steps {
				media := tt.medias[i]
				var res *FetchedMedia
				if step.result == FetchResultOK {
					res = &FetchedMedia{Content: []byte(step.content), ContentType: "image/jpeg", Filename: "media-x.jpg"}
				}
				calls = append(calls, f.mockFetcher.EXPECT().Fetch(gomock.Any(), media, i, step.budget).Return(res, step.result))

				if step.result != FetchResultOK {
					continue
				}
				idx := storeIdx
				storeIdx++
				var call *gomock.Call
				if tt.storeErr[idx] {
					call = f.mockReq.EXPECT().StorageV1FileCreateContent(gomock.Any(), f.customerID, uuid.Nil, smfile.TypeMedia, fmt.Sprintf("media-%d", i), "received with the message "+f.messageID.String(), "media-x.jpg", "image/jpeg", []byte(step.content), inboundMediaStorageTimeout).Return(nil, errors.New("simulated"))
				} else {
					call = f.mockReq.EXPECT().StorageV1FileCreateContent(gomock.Any(), f.customerID, uuid.Nil, smfile.TypeMedia, fmt.Sprintf("media-%d", i), "received with the message "+f.messageID.String(), "media-x.jpg", "image/jpeg", []byte(step.content), inboundMediaStorageTimeout).Return(&smfile.File{Identity: commonidentity.Identity{ID: tt.storeIDs[idx]}}, nil)
				}
				calls = append(calls, call)
			}
			if tt.expectUpdated {
				updated := f.message()
				updated.Medias = tt.expectMedias
				calls = append(calls,
					f.mockDB.EXPECT().MessageUpdate(gomock.Any(), f.messageID, map[message.Field]any{message.FieldMedias: tt.expectMedias}).Return(nil),
					f.mockDB.EXPECT().MessageGet(gomock.Any(), f.messageID).Return(updated, nil),
					f.mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), f.customerID, message.EventTypeMessageUpdated, updated),
				)
			}
			for i := 1; i < len(calls); i++ {
				calls[i].After(calls[i-1])
			}

			f.h.ingestInboundMedias(ctx, m, tt.medias)

			for r, n := range tt.expectCounts {
				if got := counterValue(r) - before[r]; got != n {
					t.Errorf("Wrong match. result: %s, expect: %v, got: %v", r, n, got)
				}
			}
		})
	}
}

func Test_ingestInboundMedias_deadline(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)

	// the context is over before the first media: nothing is fetched, nothing is written, all are counted.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	before := counterValue(FetchResultTimeout)
	f.h.ingestInboundMedias(ctx, f.message(), []telnyx.PayloadMedia{{URL: "https://media.telnyx.com/a"}, {URL: "https://media.telnyx.com/b"}})

	if got := counterValue(FetchResultTimeout) - before; got != 2 {
		t.Errorf("Wrong match. expect: 2, got: %v", got)
	}
}

func Test_ingestInboundMedias_updateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)
	fileA := uuid.FromStringOrNil("5b0f7e46-1a2b-4c3d-8e9f-0000000000a1")

	f.mockFetcher.EXPECT().Fetch(gomock.Any(), gomock.Any(), 0, inboundMediaMaxTotalSize).Return(&FetchedMedia{Content: []byte("A"), ContentType: "image/jpeg", Filename: "media-0.jpg"}, FetchResultOK)
	f.mockReq.EXPECT().StorageV1FileCreateContent(gomock.Any(), f.customerID, uuid.Nil, smfile.TypeMedia, gomock.Any(), gomock.Any(), "media-0.jpg", "image/jpeg", []byte("A"), inboundMediaStorageTimeout).Return(&smfile.File{Identity: commonidentity.Identity{ID: fileA}}, nil)
	f.mockDB.EXPECT().MessageUpdate(gomock.Any(), f.messageID, gomock.Any()).Return(errors.New("simulated"))

	// no read and no event after a failed write
	f.h.ingestInboundMedias(context.Background(), f.message(), []telnyx.PayloadMedia{{URL: "https://media.telnyx.com/a"}})
}

func Test_startInboundMediaIngest_busy(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)
	// fill every slot: no goroutine starts, no fetch happens.
	for i := 0; i < cap(f.h.ingestSlots); i++ {
		f.h.ingestSlots <- struct{}{}
	}

	before := counterValue(FetchResultBusy)
	f.h.startInboundMediaIngest(context.Background(), f.message(), []telnyx.PayloadMedia{{URL: "https://media.telnyx.com/a"}})

	if got := counterValue(FetchResultBusy) - before; got != 1 {
		t.Errorf("Wrong match. expect: 1, got: %v", got)
	}
	if len(f.h.ingestSlots) != cap(f.h.ingestSlots) {
		t.Errorf("Wrong match. the slots must stay as they were")
	}

	// a handler built without slots (nil channel) never blocks either.
	f.h.ingestSlots = nil
	f.h.startInboundMediaIngest(context.Background(), f.message(), []telnyx.PayloadMedia{{URL: "https://media.telnyx.com/a"}})
}

func Test_startInboundMediaIngest_releasesSlot(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)

	done := make(chan struct{})
	f.mockFetcher.EXPECT().Fetch(gomock.Any(), gomock.Any(), 0, inboundMediaMaxTotalSize).DoAndReturn(func(context.Context, telnyx.PayloadMedia, int, int64) (*FetchedMedia, FetchResult) {
		if len(f.h.ingestSlots) != 1 {
			t.Errorf("Wrong match. expect: the slot is taken while the ingest runs")
		}
		close(done)
		return nil, FetchResultHost
	})

	f.h.startInboundMediaIngest(context.Background(), f.message(), []telnyx.PayloadMedia{{URL: "https://media.telnyx.com/a"}})

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. the ingest did not run")
	}

	deadline := time.Now().Add(5 * time.Second)
	for len(f.h.ingestSlots) != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(f.h.ingestSlots) != 0 {
		t.Errorf("Wrong match. expect: the slot is released")
	}
}

func Test_dbUpdateMedias(t *testing.T) {
	ctx := context.Background()

	t.Run("normal", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		f := newInboundFixture(t, mc)
		updated := f.message()
		updated.Medias = []string{"a"}

		f.mockDB.EXPECT().MessageUpdate(ctx, f.messageID, map[message.Field]any{message.FieldMedias: []string{"a"}}).Return(nil)
		f.mockDB.EXPECT().MessageGet(ctx, f.messageID).Return(updated, nil)
		f.mockNotify.EXPECT().PublishWebhookEvent(ctx, f.customerID, message.EventTypeMessageUpdated, updated)

		res, err := f.h.dbUpdateMedias(ctx, f.messageID, []string{"a"})
		if err != nil || res != updated {
			t.Errorf("Wrong match. got: %v, %v", res, err)
		}
	})

	t.Run("get fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		f := newInboundFixture(t, mc)
		f.mockDB.EXPECT().MessageUpdate(ctx, f.messageID, gomock.Any()).Return(nil)
		f.mockDB.EXPECT().MessageGet(ctx, f.messageID).Return(nil, errors.New("simulated"))

		if res, err := f.h.dbUpdateMedias(ctx, f.messageID, []string{"a"}); err == nil || res != nil {
			t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
		}
	})
}

const (
	testHookMediaURL = "https://media.telnyx.com/example/secret-token-123/photo.jpg"
	testHookTo       = "+15550001118"
)

func mmsHookBody(eventType string, msgType string, media string) []byte {
	return []byte(`{
		"data": {
			"event_type": "` + eventType + `",
			"id": "19539336-11ba-4792-abd8-26d4f8745c4c",
			"payload": {
				"direction": "inbound",
				"from": {"phone_number": "+15550001234"},
				"id": "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
				"media": ` + media + `,
				"parts": 1,
				"text": "hello",
				"to": [{"phone_number": "` + testHookTo + `", "status": "webhook_delivered"}],
				"type": "` + msgType + `"
			},
			"record_type": "event"
		},
		"meta": {"attempt": 1}
	}`)
}

func (f *inboundFixture) expectNumberLookup() {
	numbers := []nmnumber.Number{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("67afdd50-a65d-11ec-84fa-8b61a2028c6a"), CustomerID: f.customerID}, Number: testHookTo}}
	f.mockReq.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), map[nmnumber.Field]any{nmnumber.FieldNumber: testHookTo, nmnumber.FieldDeleted: false}).Return(numbers, nil)
}

func (f *inboundFixture) expectCreate(t *testing.T, expectType message.Type) {
	f.mockUtil.EXPECT().UUIDCreate().Return(f.messageID)
	f.mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, m *message.Message) error {
		if m.Type != expectType || m.Direction != message.DirectionInbound || len(m.Medias) != 0 {
			t.Errorf("Wrong match. type: %s, direction: %s, medias: %v", m.Type, m.Direction, m.Medias)
		}
		return nil
	})
	created := f.message()
	created.Type = expectType
	f.mockDB.EXPECT().MessageGet(gomock.Any(), f.messageID).Return(created, nil)
	f.mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), f.customerID, message.EventTypeMessageCreated, created)
}

func Test_hookTelnyx_mms(t *testing.T) {
	logHook := captureLogs(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)
	fileA := uuid.FromStringOrNil("5b0f7e46-1a2b-4c3d-8e9f-0000000000a1")

	f.expectNumberLookup()
	f.expectCreate(t, message.TypeMMS)

	media := telnyx.PayloadMedia{URL: testHookMediaURL, ContentType: "image/jpeg", Size: 1234, SHA256: "00"}
	f.mockFetcher.EXPECT().Fetch(gomock.Any(), media, 0, inboundMediaMaxTotalSize).Return(&FetchedMedia{Content: []byte("A"), ContentType: "image/jpeg", Filename: "media-0.jpg"}, FetchResultOK)
	f.mockReq.EXPECT().StorageV1FileCreateContent(gomock.Any(), f.customerID, uuid.Nil, smfile.TypeMedia, gomock.Any(), gomock.Any(), "media-0.jpg", "image/jpeg", []byte("A"), inboundMediaStorageTimeout).Return(&smfile.File{Identity: commonidentity.Identity{ID: fileA}}, nil)
	f.mockDB.EXPECT().MessageUpdate(gomock.Any(), f.messageID, map[message.Field]any{message.FieldMedias: []string{fileA.String()}}).Return(nil)
	updated := f.message()
	updated.Medias = []string{fileA.String()}
	f.mockDB.EXPECT().MessageGet(gomock.Any(), f.messageID).Return(updated, nil)

	done := make(chan struct{})
	f.mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), f.customerID, message.EventTypeMessageUpdated, updated).Do(func(_ context.Context, _ uuid.UUID, _ string, _ any) { close(done) })

	body := mmsHookBody("message.received", "MMS", `[{"url":"`+testHookMediaURL+`","content_type":"image/jpeg","size":1234,"sha256":"00"}]`)
	if err := f.h.Hook(context.Background(), "https://hook.example.net/v1.0/hooks/telnyx", body); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. the ingest did not finish")
	}

	assertNoURLInLogs(t, logHook)
}

func Test_hookTelnyx_noIngest(t *testing.T) {
	tests := []struct {
		name string

		eventType  string
		msgType    string
		media      string
		expectType message.Type
		skipCreate bool
	}{
		{name: "sms without media", eventType: "message.received", msgType: "SMS", media: `[]`, expectType: message.TypeSMS},
		{name: "sms without the field value", eventType: "message.received", msgType: "SMS", media: `null`, expectType: message.TypeSMS},
		{name: "mms with an empty list", eventType: "message.received", msgType: "MMS", media: `[]`, expectType: message.TypeMMS},
		{name: "mms with an unreadable list", eventType: "message.received", msgType: "MMS", media: `{"url":"x"}`, expectType: message.TypeMMS},
		{name: "mms with unreadable items", eventType: "message.received", msgType: "MMS", media: `[{"url":"x","size":"big"}]`, expectType: message.TypeMMS},
		{name: "sent event", eventType: "message.sent", msgType: "MMS", media: `[{"url":"` + testHookMediaURL + `","content_type":"image/jpeg"}]`, skipCreate: true},
		{name: "finalized event", eventType: "message.finalized", msgType: "MMS", media: `[{"url":"` + testHookMediaURL + `","content_type":"image/jpeg"}]`, skipCreate: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no fetcher expectation and no storage expectation: any ingest fails the test.
			f := newInboundFixture(t, mc)
			if !tt.skipCreate {
				f.expectNumberLookup()
				f.expectCreate(t, tt.expectType)
			}

			if err := f.h.Hook(context.Background(), "https://hook.example.net/v1.0/hooks/telnyx", mmsHookBody(tt.eventType, tt.msgType, tt.media)); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			time.Sleep(20 * time.Millisecond) // a wrongly started ingest would call a mock that has no expectation
		})
	}
}

func Test_hookTelnyx_parseCounter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)
	f.expectNumberLookup()
	f.expectCreate(t, message.TypeMMS)

	before := counterValue(FetchResultParse)
	body := mmsHookBody("message.received", "MMS", `[{"url":"x","size":"big"},{"url":"y","size":"big"}]`)
	if err := f.h.Hook(context.Background(), "https://hook.example.net/v1.0/hooks/telnyx", body); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got := counterValue(FetchResultParse) - before; got != 2 {
		t.Errorf("Wrong match. expect: 2, got: %v", got)
	}
}

func Test_hookTelnyx_createError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)
	f.expectNumberLookup()
	f.mockUtil.EXPECT().UUIDCreate().Return(f.messageID)
	f.mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(errors.New("simulated"))

	body := mmsHookBody("message.received", "MMS", `[{"url":"`+testHookMediaURL+`","content_type":"image/jpeg"}]`)
	if err := f.h.Hook(context.Background(), "https://hook.example.net/v1.0/hooks/telnyx", body); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	time.Sleep(20 * time.Millisecond)
}

func Test_hookTelnyx_logsNoURL(t *testing.T) {
	logHook := captureLogs(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)

	// an unreadable body that still carries a url: the error text and the logs must not repeat it.
	bad := []byte(`{"data": {"payload": {"media": [{"url":"` + testHookMediaURL + `"}], "to": 5`)
	err := f.h.Hook(context.Background(), "https://hook.example.net/v1.0/hooks/telnyx", bad)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if strings.Contains(err.Error(), "secret-token-123") {
		t.Errorf("Wrong match. the error carries the url: %v", err)
	}

	// a body that parses but names an unknown number.
	f.mockReq.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), gomock.Any()).Return(nil, nil)
	body := mmsHookBody("message.received", "MMS", `[{"url":"`+testHookMediaURL+`","content_type":"image/jpeg"}]`)
	err = f.h.Hook(context.Background(), "https://hook.example.net/v1.0/hooks/telnyx", body)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if strings.Contains(err.Error(), "secret-token-123") {
		t.Errorf("Wrong match. the error carries the url: %v", err)
	}

	assertNoURLInLogs(t, logHook)
}

// captureLogs returns a hook that records every log line, debug lines included.
func captureLogs(t *testing.T) *logrustest.Hook {
	t.Helper()

	old := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() { logrus.SetLevel(old) })

	logHook := logrustest.NewGlobal()
	t.Cleanup(logHook.Reset)

	return logHook
}

func assertNoURLInLogs(t *testing.T, logHook *logrustest.Hook) {
	t.Helper()

	for _, e := range logHook.AllEntries() {
		line, _ := e.String()
		if strings.Contains(line, "secret-token-123") || strings.Contains(line, "media.telnyx.com") && strings.Contains(line, "photo.jpg") {
			t.Errorf("Wrong match. a log line carries the media url: %s", line)
		}
	}
}

// Test_ingestInboundMedias_context checks the lifetime rules of the ingest: every call runs under the deadline of
// the ingest, and the write of the stored ids still happens when that context is over.
func Test_ingestInboundMedias_context(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)
	fileA := uuid.FromStringOrNil("5b0f7e46-1a2b-4c3d-8e9f-0000000000a1")

	parent, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	f.mockFetcher.EXPECT().Fetch(gomock.Any(), gomock.Any(), 0, inboundMediaMaxTotalSize).DoAndReturn(func(ctx context.Context, _ telnyx.PayloadMedia, _ int, _ int64) (*FetchedMedia, FetchResult) {
		dl, ok := ctx.Deadline()
		if !ok || time.Until(dl) > inboundMediaIngestTimeout {
			t.Errorf("Wrong match. expect: a deadline within %v, got: %v, %v", inboundMediaIngestTimeout, dl, ok)
		}
		return &FetchedMedia{Content: []byte("A"), ContentType: "image/jpeg", Filename: "media-0.jpg"}, FetchResultOK
	})
	f.mockReq.EXPECT().StorageV1FileCreateContent(gomock.Any(), f.customerID, uuid.Nil, smfile.TypeMedia, gomock.Any(), gomock.Any(), "media-0.jpg", "image/jpeg", []byte("A"), inboundMediaStorageTimeout).DoAndReturn(
		func(context.Context, uuid.UUID, uuid.UUID, smfile.Type, string, string, string, string, []byte, int) (*smfile.File, error) {
			// the context of the ingest ends here, after the file was stored.
			cancelParent()
			return &smfile.File{Identity: commonidentity.Identity{ID: fileA}}, nil
		})
	f.mockDB.EXPECT().MessageUpdate(gomock.Any(), f.messageID, gomock.Any()).DoAndReturn(func(ctx context.Context, _ uuid.UUID, _ map[message.Field]any) error {
		if ctx.Err() != nil {
			t.Errorf("Wrong match. the write must not be cancelled with the ingest: %v", ctx.Err())
		}
		return nil
	})
	updated := f.message()
	f.mockDB.EXPECT().MessageGet(gomock.Any(), f.messageID).Return(updated, nil)
	f.mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), f.customerID, message.EventTypeMessageUpdated, updated)

	f.h.ingestInboundMedias(parent, f.message(), []telnyx.PayloadMedia{{URL: "https://media.telnyx.com/a"}})
}

// Test_startInboundMediaIngest_requestContext checks that the ingest does not stop when the context of the request
// (which ends with the answer of the hook) is cancelled.
func Test_startInboundMediaIngest_requestContext(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)

	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	f.mockFetcher.EXPECT().Fetch(gomock.Any(), gomock.Any(), 0, inboundMediaMaxTotalSize).DoAndReturn(func(ctx context.Context, _ telnyx.PayloadMedia, _ int, _ int64) (*FetchedMedia, FetchResult) {
		defer close(done)
		if ctx.Err() != nil {
			t.Errorf("Wrong match. the ingest must not see the cancelled request context: %v", ctx.Err())
		}
		return nil, FetchResultHost
	})

	f.h.startInboundMediaIngest(reqCtx, f.message(), []telnyx.PayloadMedia{{URL: "https://media.telnyx.com/a"}})

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. the ingest did not run")
	}
}

// Test_hookTelnyx_mmsWithoutMedia checks that an MMS whose media list is empty or unreadable does not start an
// ingest, so it neither takes a slot nor counts as busy even when every slot is taken.
func Test_hookTelnyx_mmsWithoutMedia(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newInboundFixture(t, mc)
	for i := 0; i < cap(f.h.ingestSlots); i++ {
		f.h.ingestSlots <- struct{}{}
	}
	f.expectNumberLookup()
	f.expectCreate(t, message.TypeMMS)

	before := counterValue(FetchResultBusy)
	if err := f.h.Hook(context.Background(), "https://hook.example.net/v1.0/hooks/telnyx", mmsHookBody("message.received", "MMS", `[]`)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got := counterValue(FetchResultBusy) - before; got != 0 {
		t.Errorf("Wrong match. expect: no busy count, got: %v", got)
	}
}
