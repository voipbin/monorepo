package analysishandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"monorepo/bin-timeline-manager/models/event"
)

// reduceCE builds a canonicalEvent whose approximate size (Summary + EventType +
// ResourceID + 64) is exactly total bytes. EventType is "e" and ResourceID is empty.
func reduceCE(name string, total int, isErr bool) *canonicalEvent {
	return &canonicalEvent{
		EventType:  "e",
		Summary:    strings.Repeat("x", total-65),
		ResourceID: "",
		isError:    isErr,
		raw:        &event.Event{EventType: name},
	}
}

func sameEvents(got, want []*canonicalEvent) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func Test_reduceEvents(t *testing.T) {
	// The reduce target is 98304 bytes (96 KiB).
	t.Run("under the target is returned as is", func(t *testing.T) {
		in := []*canonicalEvent{reduceCE("a", 1000, false), reduceCE("b", 1000, false)}

		got, reduced := reduceEvents(in)
		if reduced || !sameEvents(got, in) {
			t.Errorf("Wrong match. expect: unchanged, got: %d events reduced=%v", len(got), reduced)
		}
	})

	t.Run("size exactly at the target is not reduced", func(t *testing.T) {
		in := []*canonicalEvent{reduceCE("a", 50000, false), reduceCE("b", 48304, false)}

		got, reduced := reduceEvents(in)
		if reduced || !sameEvents(got, in) {
			t.Errorf("Wrong match. expect: unchanged, got: %d events reduced=%v", len(got), reduced)
		}
	})

	t.Run("one byte over the target drops the trailing non-error event", func(t *testing.T) {
		a := reduceCE("a", 50000, false)
		b := reduceCE("b", 48305, false)

		got, reduced := reduceEvents([]*canonicalEvent{a, b})
		if !reduced || !sameEvents(got, []*canonicalEvent{a}) {
			t.Errorf("Wrong match. expect: only a, reduced, got: %d events reduced=%v", len(got), reduced)
		}
	})

	t.Run("accumulated size exactly at the target keeps the non-error event", func(t *testing.T) {
		n1 := reduceCE("n1", 60000, false)
		n2 := reduceCE("n2", 38304, false)
		e := reduceCE("e", 5000, true)

		got, reduced := reduceEvents([]*canonicalEvent{n1, n2, e})
		if reduced || !sameEvents(got, []*canonicalEvent{n1, n2, e}) {
			t.Errorf("Wrong match. expect: all kept, not reduced, got: %d events reduced=%v", len(got), reduced)
		}
	})

	t.Run("accumulated size one byte over the target drops the non-error event", func(t *testing.T) {
		n1 := reduceCE("n1", 60000, false)
		n2 := reduceCE("n2", 38305, false)
		e := reduceCE("e", 5000, true)

		got, reduced := reduceEvents([]*canonicalEvent{n1, n2, e})
		if !reduced || !sameEvents(got, []*canonicalEvent{n1, e}) {
			t.Errorf("Wrong match. expect: n1,e reduced, got: %d events reduced=%v", len(got), reduced)
		}
	})

	t.Run("error events are kept once and in order, also after the budget is full", func(t *testing.T) {
		e1 := reduceCE("e1", 1000, true)
		n1 := reduceCE("n1", 50000, false)
		e2 := reduceCE("e2", 2000, true)
		n2 := reduceCE("n2", 60000, false)
		e3 := reduceCE("e3", 3000, true)

		got, reduced := reduceEvents([]*canonicalEvent{e1, n1, e2, n2, e3})
		if !reduced || !sameEvents(got, []*canonicalEvent{e1, n1, e2, e3}) {
			t.Errorf("Wrong match. expect: e1,n1,e2,e3 reduced, got: %d events reduced=%v", len(got), reduced)
		}
	})

	t.Run("a small non-error event after a dropped large one is kept", func(t *testing.T) {
		n1 := reduceCE("n1", 50000, false)
		n2 := reduceCE("n2", 60000, false)
		n3 := reduceCE("n3", 1000, false)

		got, reduced := reduceEvents([]*canonicalEvent{n1, n2, n3})
		if !reduced || !sameEvents(got, []*canonicalEvent{n1, n3}) {
			t.Errorf("Wrong match. expect: n1,n3 reduced, got: %d events reduced=%v", len(got), reduced)
		}
	})

	t.Run("over the target with only error events keeps all and reports no reduction", func(t *testing.T) {
		e1 := reduceCE("e1", 60000, true)
		e2 := reduceCE("e2", 60000, true)

		got, reduced := reduceEvents([]*canonicalEvent{e1, e2})
		if reduced || !sameEvents(got, []*canonicalEvent{e1, e2}) {
			t.Errorf("Wrong match. expect: e1,e2 not reduced, got: %d events reduced=%v", len(got), reduced)
		}
	})
}

func Test_summarizeData(t *testing.T) {
	// 232 and 233 characters keep the whole summary at 240 and 241 runes in the
	// "no known key" and "known key" paths below.
	r232 := strings.Repeat("あ", 232)
	r233 := strings.Repeat("あ", 233)

	tests := []struct {
		name string
		data string

		expect string
	}{
		{"empty data", ``, ""},
		{"array is not an object", `[1]`, `[1]`},
		{"null", `null`, `null`},
		{"empty object", `{}`, `{}`},
		{"known keys follow the fixed order", `{"error":"e","status":"s","zzz":1}`, "status=s error=e"},
		{"number and bool values", `{"message":1.5,"error":true}`, "message=1.5 error=true"},
		{"unknown keys only falls back to raw", `{"x":"y"}`, `{"x":"y"}`},

		{"non-object at 240 runes is kept", `"` + strings.Repeat("あ", 238) + `"`, `"` + strings.Repeat("あ", 238) + `"`},
		{"non-object at 241 runes is cut", `"` + strings.Repeat("あ", 239) + `"`, `"` + strings.Repeat("あ", 239) + "..."},
		{"no known key at 240 runes is kept", `{"x":"` + r232 + `"}`, `{"x":"` + r232 + `"}`},
		{"no known key at 241 runes is cut", `{"x":"` + r233 + `"}`, `{"x":"` + r233 + `"` + "..."},
		{"known key at 240 runes is kept", `{"message":"` + r232 + `"}`, "message=" + r232},
		{"known key at 241 runes is cut", `{"message":"` + r233 + `"}`, "message=" + r232 + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeData(&event.Event{Data: []byte(tt.data)})
			if got != tt.expect {
				t.Errorf("Wrong match.\nexpect: %q\ngot: %q", tt.expect, got)
			}
		})
	}
}

func Test_resourceIDOf(t *testing.T) {
	tests := []struct {
		name string
		data string

		expect string
	}{
		{"string id", `{"id":"abc"}`, "abc"},
		{"numeric id", `{"id":12}`, ""},
		{"no id", `{}`, ""},
		{"invalid json", `{bad`, ""},
		{"empty data", ``, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resourceIDOf(&event.Event{Data: []byte(tt.data)})
			if got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_truncateRunes(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		limit int

		expect string
	}{
		{"shorter than the limit", "ab", 3, "ab"},
		{"equal to the limit", "abc", 3, "abc"},
		{"one over the limit", "abcd", 3, "abc..."},
		{"multibyte counts runes", "가나다라", 3, "가나다..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateRunes(tt.in, tt.limit); got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_joinTranscript(t *testing.T) {
	got := joinTranscript([]tmtranscript.Transcript{
		{Direction: tmtranscript.DirectionIn, Message: "hello"},
		{Direction: tmtranscript.DirectionOut, Message: ""},
		{Direction: tmtranscript.DirectionOut, Message: "bye"},
	})
	if expect := "[in] hello\n[out] bye"; got != expect {
		t.Errorf("Wrong match. expect: %q, got: %q", expect, got)
	}

	if got := joinTranscript(nil); got != "" {
		t.Errorf("Wrong match. expect: empty, got: %q", got)
	}
}

func Test_collectTranscripts(t *testing.T) {
	custID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	tr1 := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333331")
	tr2 := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333332")

	listFilters := map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:   custID,
		tmtranscribe.FieldActiveflowID: afID,
		tmtranscribe.FieldDeleted:      false,
	}
	transcriptFilters := func(id uuid.UUID) map[tmtranscript.Field]any {
		return map[tmtranscript.Field]any{
			tmtranscript.FieldTranscribeID: id,
			tmtranscript.FieldDeleted:      false,
		}
	}
	transcribes := []tmtranscribe.Transcribe{
		{Identity: commonidentity.Identity{ID: tr1}},
		{Identity: commonidentity.Identity{ID: tr2}},
	}

	t.Run("nil activeflow makes no call", func(t *testing.T) {
		h, _, _, mc := newStartTestHandler(t)
		defer mc.Finish()

		if got := h.collectTranscripts(context.Background(), custID, uuid.Nil); got != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", got)
		}
	})

	t.Run("list error returns nil", func(t *testing.T) {
		h, reqMock, _, mc := newStartTestHandler(t)
		defer mc.Finish()
		ctx := context.Background()

		reqMock.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), listFilters).Return(nil, errors.New("rpc failure"))

		if got := h.collectTranscripts(ctx, custID, afID); got != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", got)
		}
	})

	t.Run("a failing transcribe is skipped, empty messages are ignored", func(t *testing.T) {
		h, reqMock, _, mc := newStartTestHandler(t)
		defer mc.Finish()
		ctx := context.Background()

		reqMock.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), listFilters).Return(transcribes, nil)
		reqMock.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), transcriptFilters(tr1)).Return(nil, errors.New("rpc failure"))
		reqMock.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), transcriptFilters(tr2)).Return([]tmtranscript.Transcript{
			{Direction: tmtranscript.DirectionIn, Message: "hi"},
			{Direction: tmtranscript.DirectionOut, Message: ""},
		}, nil)

		got := h.collectTranscripts(ctx, custID, afID)
		if expect := []string{"[in] hi"}; !reflect.DeepEqual(got, expect) {
			t.Errorf("Wrong match. expect: %v, got: %v", expect, got)
		}
	})

	t.Run("a transcribe with only empty messages yields nothing", func(t *testing.T) {
		h, reqMock, _, mc := newStartTestHandler(t)
		defer mc.Finish()
		ctx := context.Background()

		reqMock.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), listFilters).Return(transcribes[:1], nil)
		reqMock.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), transcriptFilters(tr1)).Return([]tmtranscript.Transcript{
			{Direction: tmtranscript.DirectionIn, Message: ""},
		}, nil)

		got := h.collectTranscripts(ctx, custID, afID)
		if got == nil || len(got) != 0 {
			t.Errorf("Wrong match. expect: empty non-nil slice, got: %v", got)
		}
	})

	// the joined text is "[in] " (5 runes) plus the message; the per-resource cap is 8000 runes.
	capTests := []struct {
		name   string
		msgLen int

		expect string
	}{
		{"exactly 8000 runes is kept", 7995, "[in] " + strings.Repeat("あ", 7995)},
		{"8001 runes is cut to 8000 plus ellipsis", 7996, "[in] " + strings.Repeat("あ", 7995) + "..."},
	}
	for _, tt := range capTests {
		t.Run(tt.name, func(t *testing.T) {
			h, reqMock, _, mc := newStartTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()

			reqMock.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), listFilters).Return(transcribes[:1], nil)
			reqMock.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), transcriptFilters(tr1)).Return([]tmtranscript.Transcript{
				{Direction: tmtranscript.DirectionIn, Message: strings.Repeat("あ", tt.msgLen)},
			}, nil)

			got := h.collectTranscripts(ctx, custID, afID)
			if len(got) != 1 || got[0] != tt.expect {
				t.Errorf("Wrong match. expect one text of %d runes, got: %d texts", len([]rune(tt.expect)), len(got))
			}
		})
	}
}
