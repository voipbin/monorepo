package participant

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	created := time.Date(2026, 7, 8, 9, 10, 11, 0, time.UTC)

	tests := []struct {
		name string
		in   *Participant

		expectAI   string
		expectCall string
		expectTM   *time.Time
	}{
		{
			name: "all fields",
			in: &Participant{
				AIID:     uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				AIcallID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				TMCreate: &created,
			},
			expectAI:   "11111111-1111-1111-1111-111111111111",
			expectCall: "22222222-2222-2222-2222-222222222222",
			expectTM:   &created,
		},
		{
			name:       "zero value",
			in:         &Participant{},
			expectAI:   "00000000-0000-0000-0000-000000000000",
			expectCall: "00000000-0000-0000-0000-000000000000",
			expectTM:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.ConvertWebhookMessage()
			if got.AIID.String() != tt.expectAI {
				t.Errorf("AIID = %s, expect: %s", got.AIID, tt.expectAI)
			}
			if got.AIcallID.String() != tt.expectCall {
				t.Errorf("AIcallID = %s, expect: %s", got.AIcallID, tt.expectCall)
			}
			if (got.TMCreate == nil) != (tt.expectTM == nil) {
				t.Fatalf("TMCreate = %v, expect: %v", got.TMCreate, tt.expectTM)
			}
			if tt.expectTM != nil && !got.TMCreate.Equal(*tt.expectTM) {
				t.Errorf("TMCreate = %v, expect: %v", got.TMCreate, tt.expectTM)
			}
		})
	}
}
