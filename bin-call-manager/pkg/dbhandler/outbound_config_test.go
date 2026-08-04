package dbhandler

import (
	"context"
	"database/sql/driver"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

// Tests for pkg/dbhandler/outbound_config.go (VOIP-1078 plan Phase 7).
//
// This file is new: outbound_config.go had ZERO test coverage before this
// migration, which is why the plan makes tests a required Phase 7 deliverable
// alongside the new scripts/database_scripts_test/table_outbound_configs.sql
// fixture.

func newOutboundConfigTestHandler(t *testing.T) (*handler, *utilhandler.MockUtilHandler, *gomock.Controller) {
	t.Helper()

	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	return &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}, mockUtil, mc
}

// ---------------------------------------------------------------------------
// R-C: UUID columns must be written as raw 16 bytes, never a 36-char string
// ---------------------------------------------------------------------------

func Test_OutboundConfigCreate_uuidColumnsStoredAs16Bytes(t *testing.T) {
	h, mockUtil, mc := newOutboundConfigTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	id := uuid.FromStringOrNil("c1000000-0000-11f0-9e2b-0242ac110002")
	customerID := uuid.FromStringOrNil("c1100000-0000-11f0-9e2b-0242ac110002")
	numberID := uuid.FromStringOrNil("c1200000-0000-11f0-9e2b-0242ac110002")
	ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

	mockUtil.EXPECT().TimeNow().Return(ts)
	if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{
		ID:                            id,
		CustomerID:                    customerID,
		DefaultOutgoingSourceNumberID: numberID,
	}); err != nil {
		t.Fatalf("could not create outbound config. err: %v", err)
	}

	// The columns are BINARY(16) in MySQL. Asserting "not NULL" would pass even
	// for a corrupted 36-char write, so assert the byte length explicitly.
	for _, column := range []string{"id", "customer_id", "default_outgoing_source_number_id"} {
		got := selectRawColumn(t, "SELECT "+column+" FROM call_outbound_configs WHERE id = ?", id.Bytes())
		if len(got) != 16 {
			t.Errorf("%s must be stored as raw 16 bytes, not a 36-char string. expect len: 16, got len: %d (%q)", column, len(got), got)
		}
	}
}

func Test_OutboundConfigUpdate_uuidStoredAs16Bytes(t *testing.T) {
	h, mockUtil, mc := newOutboundConfigTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	id := uuid.FromStringOrNil("c2000000-0000-11f0-9e2b-0242ac110002")
	customerID := uuid.FromStringOrNil("c2100000-0000-11f0-9e2b-0242ac110002")
	newNumberID := uuid.FromStringOrNil("c2200000-0000-11f0-9e2b-0242ac110002")
	ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

	mockUtil.EXPECT().TimeNow().Return(ts)
	if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{ID: id, CustomerID: customerID}); err != nil {
		t.Fatalf("could not create outbound config. err: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(ts)
	if _, err := h.OutboundConfigUpdate(ctx, id, &outboundconfig.UpdateRequest{
		DefaultOutgoingSourceNumberID: &newNumberID,
	}); err != nil {
		t.Fatalf("could not update outbound config. err: %v", err)
	}

	got := selectRawColumn(t, "SELECT default_outgoing_source_number_id FROM call_outbound_configs WHERE id = ?", id.Bytes())
	if len(got) != 16 {
		t.Errorf("default_outgoing_source_number_id must be stored as raw 16 bytes. expect len: 16, got len: %d (%q)", len(got), got)
	}
	if got != string(newNumberID.Bytes()) {
		t.Errorf("stored uuid bytes do not round-trip. expect: %q, got: %q", string(newNumberID.Bytes()), got)
	}
}

// Test_outboundConfigUpdate_pointerUUIDCorruptionMechanism pins the failure mode
// plan §5 R-C describes, so the dereference in OutboundConfigUpdate is not
// "simplified" away later.
//
// A *uuid.UUID placed in the update map is NOT matched by processMapValues's
// uuid.UUID type switch, and falls through convertValueForDB unchanged (Ptr ->
// Elem is an Array, which matches none of the Slice/Map/Struct branches). The
// 36-char string appears one layer deeper: gofrs/uuid implements driver.Valuer
// with a VALUE receiver, which *uuid.UUID also satisfies via method-set
// promotion, so database/sql itself calls Value() and gets 36 chars — which
// then hits a BINARY(16) column. That is silent truncation under permissive
// MySQL mode and a "Data too long for column" write error under strict mode.
func Test_outboundConfigUpdate_pointerUUIDCorruptionMechanism(t *testing.T) {
	numberID := uuid.FromStringOrNil("c3000000-0000-11f0-9e2b-0242ac110002")

	// The WRONG way: a raw *uuid.UUID.
	wrong, err := commondatabasehandler.PrepareFields(map[outboundconfig.Field]any{
		outboundconfig.FieldDefaultOutgoingSourceNumberID: &numberID,
	})
	if err != nil {
		t.Fatalf("prepare fields failed. err: %v", err)
	}

	got := wrong[string(outboundconfig.FieldDefaultOutgoingSourceNumberID)]
	ptr, ok := got.(*uuid.UUID)
	if !ok {
		t.Fatalf("expected the pointer to pass through unconverted, got %T", got)
	}

	// This is the corruption: what the driver would actually send.
	v, err := driver.Valuer(ptr).Value()
	if err != nil {
		t.Fatalf("Value() failed. err: %v", err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("expected a string from Value(), got %T", v)
	}
	if len(s) != 36 {
		t.Fatalf("expected the 36-char form that overflows BINARY(16), got len %d", len(s))
	}

	// The RIGHT way: a dereferenced value-typed uuid.UUID, which is what
	// OutboundConfigUpdate puts in the map.
	right, err := commondatabasehandler.PrepareFields(map[outboundconfig.Field]any{
		outboundconfig.FieldDefaultOutgoingSourceNumberID: numberID,
	})
	if err != nil {
		t.Fatalf("prepare fields failed. err: %v", err)
	}

	b, ok := right[string(outboundconfig.FieldDefaultOutgoingSourceNumberID)].([]byte)
	if !ok {
		t.Fatalf("expected []byte for a dereferenced uuid, got %T", right[string(outboundconfig.FieldDefaultOutgoingSourceNumberID)])
	}
	if len(b) != 16 {
		t.Errorf("dereferenced uuid must convert to 16 bytes. got len: %d", len(b))
	}
}

// ---------------------------------------------------------------------------
// R-A / B-2: destination_whitelist NULL / empty / non-empty, per call site
// ---------------------------------------------------------------------------

func Test_OutboundConfigCreate_destinationWhitelistThreeWay(t *testing.T) {
	tests := []struct {
		name      string
		whitelist []string
		wantRaw   string
		wantRead  []string
	}{
		{
			// R-A write table: Create normalizes nil -> []. Load-bearing, since
			// destination_whitelist is JSON NOT NULL in MySQL and the struct
			// path would otherwise write SQL NULL for a nil slice.
			name:      "nil normalizes to empty array",
			whitelist: nil,
			wantRaw:   "[]",
			wantRead:  []string{},
		},
		{
			name:      "empty stays empty array",
			whitelist: []string{},
			wantRaw:   "[]",
			wantRead:  []string{},
		},
		{
			name:      "non-empty round-trips",
			whitelist: []string{"us", "kr"},
			wantRaw:   `["us","kr"]`,
			wantRead:  []string{"us", "kr"},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockUtil, mc := newOutboundConfigTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()

			id := uuid.FromStringOrNil("c400000" + string(rune('0'+i)) + "-0000-11f0-9e2b-0242ac110002")
			customerID := uuid.FromStringOrNil("c410000" + string(rune('0'+i)) + "-0000-11f0-9e2b-0242ac110002")
			ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

			mockUtil.EXPECT().TimeNow().Return(ts)
			if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{
				ID:                   id,
				CustomerID:           customerID,
				DestinationWhitelist: tt.whitelist,
			}); err != nil {
				t.Fatalf("could not create outbound config. err: %v", err)
			}

			if got := selectRawColumn(t, "SELECT destination_whitelist FROM call_outbound_configs WHERE id = ?", id.Bytes()); got != tt.wantRaw {
				t.Errorf("wrong stored destination_whitelist. expect: %s, got: %s", tt.wantRaw, got)
			}

			res, err := h.OutboundConfigGetByID(ctx, id)
			if err != nil {
				t.Fatalf("could not get outbound config. err: %v", err)
			}
			if !reflect.DeepEqual(res.DestinationWhitelist, tt.wantRead) {
				t.Errorf("wrong read-back destination_whitelist. expect: %#v, got: %#v", tt.wantRead, res.DestinationWhitelist)
			}
		})
	}
}

func Test_OutboundConfigUpdate_destinationWhitelistThreeWay(t *testing.T) {
	nilSlice := []string(nil)
	emptySlice := []string{}
	fullSlice := []string{"us"}

	tests := []struct {
		name      string
		whitelist *[]string
		wantRaw   string
		wantRead  []string
	}{
		{
			// R-A write table: Update does NOT normalize, unlike Create. A
			// pointer to a nil slice writes the JSON literal "null" — the
			// pre-migration behavior, preserved deliberately. This is exactly
			// why the R-A table is keyed by (field x call site).
			name:      "pointer to nil slice writes JSON null",
			whitelist: &nilSlice,
			wantRaw:   "null",
			wantRead:  nil,
		},
		{
			name:      "pointer to empty slice writes empty array",
			whitelist: &emptySlice,
			wantRaw:   "[]",
			wantRead:  []string{},
		},
		{
			name:      "pointer to non-empty slice round-trips",
			whitelist: &fullSlice,
			wantRaw:   `["us"]`,
			wantRead:  []string{"us"},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockUtil, mc := newOutboundConfigTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()

			id := uuid.FromStringOrNil("c500000" + string(rune('0'+i)) + "-0000-11f0-9e2b-0242ac110002")
			customerID := uuid.FromStringOrNil("c510000" + string(rune('0'+i)) + "-0000-11f0-9e2b-0242ac110002")
			ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

			mockUtil.EXPECT().TimeNow().Return(ts)
			if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{
				ID:                   id,
				CustomerID:           customerID,
				DestinationWhitelist: []string{"initial"},
			}); err != nil {
				t.Fatalf("could not create outbound config. err: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(ts)
			res, err := h.OutboundConfigUpdate(ctx, id, &outboundconfig.UpdateRequest{
				DestinationWhitelist: tt.whitelist,
			})
			if err != nil {
				t.Fatalf("could not update outbound config. err: %v", err)
			}

			if got := selectRawColumn(t, "SELECT destination_whitelist FROM call_outbound_configs WHERE id = ?", id.Bytes()); got != tt.wantRaw {
				t.Errorf("wrong stored destination_whitelist. expect: %s, got: %s", tt.wantRaw, got)
			}
			if !reflect.DeepEqual(res.DestinationWhitelist, tt.wantRead) {
				t.Errorf("wrong read-back destination_whitelist. expect: %#v, got: %#v", tt.wantRead, res.DestinationWhitelist)
			}
		})
	}
}

// Test_OutboundConfig_destinationWhitelistNullRead covers the intentional
// read-side behavior change (plan §5 R-A / B-2): the previous hand-written scan
// called json.Unmarshal unconditionally and therefore ERRORED on a NULL or
// empty destination_whitelist. ScanRow's copyJSON skips NULL/empty and leaves
// the field nil instead, which is not an error.
func Test_OutboundConfig_destinationWhitelistNullRead(t *testing.T) {
	h, _, mc := newOutboundConfigTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	id := uuid.FromStringOrNil("c6000000-0000-11f0-9e2b-0242ac110002")
	customerID := uuid.FromStringOrNil("c6100000-0000-11f0-9e2b-0242ac110002")

	// Insert a row with a NULL destination_whitelist directly — no dbhandler
	// write path produces this, which is the point.
	if _, err := dbTest.Exec(
		"INSERT INTO call_outbound_configs(id, customer_id, name, detail, destination_whitelist, codecs, default_outgoing_source_number_id, tm_create) VALUES(?, ?, '', '', NULL, '', ?, ?)",
		id.Bytes(), customerID.Bytes(), uuid.Nil.Bytes(), "2026-08-04 01:02:03.000000",
	); err != nil {
		t.Fatalf("could not insert row. err: %v", err)
	}

	res, err := h.OutboundConfigGetByID(ctx, id)
	if err != nil {
		t.Fatalf("a NULL destination_whitelist must no longer be a read error. err: %v", err)
	}
	if res == nil {
		t.Fatalf("expected a record, got nil")
	}
	if res.DestinationWhitelist != nil {
		t.Errorf("a NULL destination_whitelist must read back as nil. got: %#v", res.DestinationWhitelist)
	}
}

// ---------------------------------------------------------------------------
// R-C: timestamps, now that all three functions use the injectable clock
// ---------------------------------------------------------------------------

func Test_OutboundConfigCreate_timestamps(t *testing.T) {
	h, mockUtil, mc := newOutboundConfigTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	id := uuid.FromStringOrNil("c7000000-0000-11f0-9e2b-0242ac110002")
	customerID := uuid.FromStringOrNil("c7100000-0000-11f0-9e2b-0242ac110002")
	ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

	mockUtil.EXPECT().TimeNow().Return(ts)
	if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{ID: id, CustomerID: customerID}); err != nil {
		t.Fatalf("could not create outbound config. err: %v", err)
	}

	res, err := h.OutboundConfigGetByID(ctx, id)
	if err != nil {
		t.Fatalf("could not get outbound config. err: %v", err)
	}

	// outbound_config sets BOTH tm_create and tm_update to now at Create, unlike
	// recording.go which leaves TMUpdate nil. Exact-match assertions are only
	// possible because OutboundConfigCreate now uses h.utilHandler.TimeNow()
	// instead of an inline time.Now().
	if !reflect.DeepEqual(res.TMCreate, ts) {
		t.Errorf("wrong tm_create. expect: %v, got: %v", ts, res.TMCreate)
	}
	if !reflect.DeepEqual(res.TMUpdate, ts) {
		t.Errorf("wrong tm_update. expect: %v, got: %v", ts, res.TMUpdate)
	}
	if res.TMDelete != nil {
		t.Errorf("tm_delete must be nil on create. got: %v", res.TMDelete)
	}
}

func Test_OutboundConfigUpdate_timestamps(t *testing.T) {
	h, mockUtil, mc := newOutboundConfigTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	id := uuid.FromStringOrNil("c8000000-0000-11f0-9e2b-0242ac110002")
	customerID := uuid.FromStringOrNil("c8100000-0000-11f0-9e2b-0242ac110002")
	createTS := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")
	updateTS := testhelper.TimePtr("2026-08-05T09:10:11.000000Z")

	mockUtil.EXPECT().TimeNow().Return(createTS)
	if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{ID: id, CustomerID: customerID}); err != nil {
		t.Fatalf("could not create outbound config. err: %v", err)
	}

	name := "updated"
	mockUtil.EXPECT().TimeNow().Return(updateTS)
	res, err := h.OutboundConfigUpdate(ctx, id, &outboundconfig.UpdateRequest{Name: &name})
	if err != nil {
		t.Fatalf("could not update outbound config. err: %v", err)
	}

	if !reflect.DeepEqual(res.TMCreate, createTS) {
		t.Errorf("tm_create must not change on update. expect: %v, got: %v", createTS, res.TMCreate)
	}
	if !reflect.DeepEqual(res.TMUpdate, updateTS) {
		t.Errorf("wrong tm_update. expect: %v, got: %v", updateTS, res.TMUpdate)
	}
	if res.Name != name {
		t.Errorf("wrong name. expect: %s, got: %s", name, res.Name)
	}
}

// Test_OutboundConfigDelete_setsTMDelete asserts tm_delete with a DIRECT SQL
// query rather than a dbhandler read method.
//
// This is required, not a style choice (plan v12 finding): both
// OutboundConfigGetByID and OutboundConfigList filter `tm_delete IS NULL`, so a
// soft-deleted row is unreachable through every dbhandler read API. (This is
// unlike recording.go's RecordingGet, which has no such filter — which is why
// the GetByID-based assertion pattern works there but not here.)
func Test_OutboundConfigDelete_setsTMDelete(t *testing.T) {
	h, mockUtil, mc := newOutboundConfigTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	id := uuid.FromStringOrNil("c9000000-0000-11f0-9e2b-0242ac110002")
	customerID := uuid.FromStringOrNil("c9100000-0000-11f0-9e2b-0242ac110002")
	createTS := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")
	deleteTS := testhelper.TimePtr("2026-08-06T07:08:09.000000Z")

	mockUtil.EXPECT().TimeNow().Return(createTS)
	if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{ID: id, CustomerID: customerID}); err != nil {
		t.Fatalf("could not create outbound config. err: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(deleteTS)
	if err := h.OutboundConfigDelete(ctx, id); err != nil {
		t.Fatalf("could not delete outbound config. err: %v", err)
	}

	got := selectRawColumn(t, "SELECT tm_delete FROM call_outbound_configs WHERE id = ?", id.Bytes())
	if got == "" {
		t.Fatalf("tm_delete must be set after delete, got empty/NULL")
	}

	// And confirm the soft-deleted row really is unreachable through the read
	// APIs, which is what makes the direct query necessary above.
	res, err := h.OutboundConfigGetByID(ctx, id)
	if err != nil {
		t.Fatalf("could not run get. err: %v", err)
	}
	if res != nil {
		t.Errorf("a soft-deleted row must not be returned by OutboundConfigGetByID. got: %v", res)
	}
}

// Test_OutboundConfigGetByCustomerID covers the customer lookup path, including
// that the ,uuid read conversion round-trips through the BINARY(16) fixture.
func Test_OutboundConfigGetByCustomerID(t *testing.T) {
	h, mockUtil, mc := newOutboundConfigTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	id := uuid.FromStringOrNil("ca000000-0000-11f0-9e2b-0242ac110002")
	customerID := uuid.FromStringOrNil("ca100000-0000-11f0-9e2b-0242ac110002")
	ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

	mockUtil.EXPECT().TimeNow().Return(ts)
	if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{
		ID:         id,
		CustomerID: customerID,
		Codecs:     "pcmu,pcma",
	}); err != nil {
		t.Fatalf("could not create outbound config. err: %v", err)
	}

	res, err := h.OutboundConfigGetByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("could not get outbound config. err: %v", err)
	}
	if res == nil {
		t.Fatalf("expected a record, got nil")
	}
	if res.ID != id {
		t.Errorf("wrong id round-trip. expect: %v, got: %v", id, res.ID)
	}
	if res.CustomerID != customerID {
		t.Errorf("wrong customer_id round-trip. expect: %v, got: %v", customerID, res.CustomerID)
	}
	if res.Codecs != "pcmu,pcma" {
		t.Errorf("wrong codecs. expect: pcmu,pcma, got: %s", res.Codecs)
	}
}

func Test_OutboundConfigList(t *testing.T) {
	h, mockUtil, mc := newOutboundConfigTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	id := uuid.FromStringOrNil("cb000000-0000-11f0-9e2b-0242ac110002")
	customerID := uuid.FromStringOrNil("cb100000-0000-11f0-9e2b-0242ac110002")
	ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

	mockUtil.EXPECT().TimeNow().Return(ts)
	if err := h.OutboundConfigCreate(ctx, &outboundconfig.OutboundConfig{ID: id, CustomerID: customerID}); err != nil {
		t.Fatalf("could not create outbound config. err: %v", err)
	}

	res, err := h.OutboundConfigList(ctx, customerID, 10, "")
	if err != nil {
		t.Fatalf("could not list outbound configs. err: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("wrong result count. expect: 1, got: %d", len(res))
	}
	if res[0].ID != id {
		t.Errorf("wrong id. expect: %v, got: %v", id, res[0].ID)
	}
}
