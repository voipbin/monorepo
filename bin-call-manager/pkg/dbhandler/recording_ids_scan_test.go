package dbhandler

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

// Test_recordingIDsCorruptedFormsFailUnmarshal pins the read-path failure modes
// of VOIP-1305 and is the executable documentation for why
// buildConfbridgeAddRecordingIDs must bind recordingID.String().
//
// Background: before VOIP-1305 the builder bound recordingID.Bytes(), so
// json_array_append stored a 16-byte binary value instead of the 36-char UUID
// text. Depending on the go-sql-driver protocol path, that produced one of two
// corrupted element shapes in the recording_ids JSON column (verified against
// MySQL 8.0 with the real driver):
//
//   - Form A: prepared-statement path (driver default, the likely production
//     path). The raw 16 bytes land inside a JSON STRING element; MySQL escapes
//     control bytes and quotes, non-ASCII bytes stay raw (invalid UTF-8).
//   - Form B: interpolateParams=true text-protocol path (or a _binary SQL
//     literal). MySQL stores an opaque JSON BLOB element that renders as
//     "base64:type15:<payload>".
//
// The model field Confbridge.RecordingIDs is []uuid.UUID (db tag
// "recording_ids,json"), so reads go through the ScanRow copyJSON path in
// bin-common-handler/pkg/databasehandler/mapping.go: the column arrives as
// sql.NullString and is json.Unmarshal-ed into []uuid.UUID. This test feeds
// each stored shape through that same json.Unmarshal step.
//
// Assertions are deliberately limited to err nil / non-nil (plus value
// equality for the clean control): the corrupted forms must fail, the clean
// .String() form must succeed. Error strings are not asserted because the uuid
// library wording differs across versions.
func Test_recordingIDsCorruptedFormsFailUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		raw     []byte
		wantErr bool
		want    []uuid.UUID
	}{
		{
			// Form A, invalid UTF-8: the 16 raw bytes of UUID
			// deadbeef-0011-2233-4455-66778899aabb as stored by the
			// prepared-statement path. MySQL kept the printable ASCII bytes
			// (3DUfw), JSON-escaped the control bytes 0x00 and 0x11 to
			// backslash-u sequences and the quote byte 0x22 to an escaped
			// quote, and left the non-ASCII bytes raw, which makes the
			// document invalid UTF-8. The raw literal below reproduces that
			// stored document byte for byte.
			name:    "form A raw bytes with invalid UTF-8 fails",
			raw:     []byte("[\"\xde\xad\xbe\xef\\u0000\\u0011\\\"3DUfw\x88\x99\xaa\xbb\"]"),
			wantErr: true,
		},
		{
			// Form A, all-ASCII: a UUID whose 16 raw bytes are all printable
			// ASCII (here 0x30..0x66, the bytes of UUID
			// 30313233-3435-3637-3839-616263646566). The prepared-statement
			// path stores them as a plain 16-char JSON string. The document
			// is valid JSON and valid UTF-8, but the element is 16 chars,
			// not the 36-char UUID text form.
			name:    "form A ascii 16-byte string fails",
			raw:     []byte(`["0123456789abcdef"]`),
			wantErr: true,
		},
		{
			// Form B: the interpolateParams / text-protocol path (or a
			// _binary literal) makes MySQL store an opaque JSON BLOB, which
			// the server renders as base64:type15:<payload>. The payload is
			// the base64 of the same 16 raw bytes of
			// deadbeef-0011-2233-4455-66778899aabb.
			name:    "form B base64 opaque blob rendering fails",
			raw:     []byte(`["base64:type15:3q2+7wARIjNEVWZ3iJmquw=="]`),
			wantErr: true,
		},
		{
			// Control: the shape written by binding recordingID.String(),
			// which is what the fixed builder produces and what the backfill
			// migration restores corrupted elements to.
			name:    "clean uuid string form succeeds",
			raw:     []byte(`["deadbeef-0011-2233-4455-66778899aabb"]`),
			wantErr: false,
			want:    []uuid.UUID{uuid.FromStringOrNil("deadbeef-0011-2233-4455-66778899aabb")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []uuid.UUID
			err := json.Unmarshal(tt.raw, &got)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected unmarshal error, got nil. result: %v", got)
				}
				return
			}

			if err != nil {
				t.Errorf("expected success, got err: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unexpected result.\nwant: %v\ngot:  %v", tt.want, got)
			}
		})
	}
}
