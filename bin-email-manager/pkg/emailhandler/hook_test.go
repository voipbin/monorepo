package emailhandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Hook(t *testing.T) {

	tests := []struct {
		name string

		uri  string
		data []byte

		expectIDs      []uuid.UUID
		expectStatuses []email.Status
	}{
		{
			name: "normal",

			uri:  "hook.voipbin.net/v1.0/emails/sendgrid",
			data: []byte(`[{"email":"pchero21@gmail.com","event":"delivered","ip":"149.72.123.24","voipbin_message_id":"b90a8049-b235-443e-b17e-07a5d2e59a79","response":"250 2.0.0 OK  1741882953 af79cd13be357-7c573d1a920si196001785a.586 - gsmtp","sg_event_id":"ZGVsaXZlcmVkLTAtNTEyMzAwODMtc0NtMHBmRFBSanktSHloSEJfNVR1dy0w","sg_message_id":"sCm0pfDPRjy-HyhHB_5Tuw.recvd-69cbc48b94-n2vpp-1-67D30649-1.0","smtp-id":"<sCm0pfDPRjy-HyhHB_5Tuw@geopod-ismtpd-8>","timestamp":1741882953,"tls":1},{"email":"pchero21@gmail.com","event":"processed","voipbin_message_id":"b90a8049-b235-443e-b17e-07a5d2e59a79","send_at":0,"sg_event_id":"cHJvY2Vzc2VkLTUxMjMwMDgzLXNDbTBwZkRQUmp5LUh5aEhCXzVUdXctMA","sg_message_id":"sCm0pfDPRjy-HyhHB_5Tuw.recvd-69cbc48b94-n2vpp-1-67D30649-1.0","smtp-id":"<sCm0pfDPRjy-HyhHB_5Tuw@geopod-ismtpd-8>","timestamp":1741882953}]`),

			expectIDs: []uuid.UUID{
				uuid.FromStringOrNil("b90a8049-b235-443e-b17e-07a5d2e59a79"),
				uuid.FromStringOrNil("b90a8049-b235-443e-b17e-07a5d2e59a79"),
			},
			expectStatuses: []email.Status{
				email.StatusProcessed,
				email.StatusDelivered,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}
			ctx := context.Background()

			for i := range tt.expectIDs {
				mockDB.EXPECT().EmailUpdateStatus(ctx, tt.expectIDs[i], tt.expectStatuses[i]).Return(nil)
				mockDB.EXPECT().EmailGet(ctx, tt.expectIDs[i]).Return(&email.Email{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), email.EventTypeUpdated, gomock.Any())
			}

			if err := h.Hook(ctx, tt.uri, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hookSendgrid(t *testing.T) {

	tests := []struct {
		name string

		data []byte

		expectIDs      []uuid.UUID
		expectStatuses []email.Status
	}{
		{
			name: "normal",

			data: []byte(`[{"email":"pchero21@gmail.com","event":"delivered","ip":"149.72.123.24","voipbin_message_id":"b90a8049-b235-443e-b17e-07a5d2e59a79","response":"250 2.0.0 OK  1741882953 af79cd13be357-7c573d1a920si196001785a.586 - gsmtp","sg_event_id":"ZGVsaXZlcmVkLTAtNTEyMzAwODMtc0NtMHBmRFBSanktSHloSEJfNVR1dy0w","sg_message_id":"sCm0pfDPRjy-HyhHB_5Tuw.recvd-69cbc48b94-n2vpp-1-67D30649-1.0","smtp-id":"<sCm0pfDPRjy-HyhHB_5Tuw@geopod-ismtpd-8>","timestamp":1741882953,"tls":1},{"email":"pchero21@gmail.com","event":"processed","voipbin_message_id":"b90a8049-b235-443e-b17e-07a5d2e59a79","send_at":0,"sg_event_id":"cHJvY2Vzc2VkLTUxMjMwMDgzLXNDbTBwZkRQUmp5LUh5aEhCXzVUdXctMA","sg_message_id":"sCm0pfDPRjy-HyhHB_5Tuw.recvd-69cbc48b94-n2vpp-1-67D30649-1.0","smtp-id":"<sCm0pfDPRjy-HyhHB_5Tuw@geopod-ismtpd-8>","timestamp":1741882953}]`),

			expectIDs: []uuid.UUID{
				uuid.FromStringOrNil("b90a8049-b235-443e-b17e-07a5d2e59a79"),
				uuid.FromStringOrNil("b90a8049-b235-443e-b17e-07a5d2e59a79"),
			},
			expectStatuses: []email.Status{
				email.StatusProcessed,
				email.StatusDelivered,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}
			ctx := context.Background()

			for i := range tt.expectIDs {
				mockDB.EXPECT().EmailUpdateStatus(ctx, tt.expectIDs[i], tt.expectStatuses[i]).Return(nil)
				mockDB.EXPECT().EmailGet(ctx, tt.expectIDs[i]).Return(&email.Email{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), email.EventTypeUpdated, gomock.Any())
			}

			if err := h.hookSendgrid(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

const testMailgunEmailID = "5e8f0f0a-8888-4888-8888-000000000001"

const (
	testMailgunSigningKey = "synthetic-signing-key"
	testMailgunTimestamp  = "1770146431"
	testMailgunToken      = "synthetic-token-0123456789"
)

// testMailgunNow is 69 seconds after testMailgunTimestamp.
func testMailgunNow() time.Time { return time.Unix(1770146500, 0) }

// mailgunTestSign returns the hex HMAC-SHA256 of timestamp+token.
func mailgunTestSign(key string, timestamp string, token string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(timestamp + token))
	return hex.EncodeToString(mac.Sum(nil))
}

// mailgunTestSignature returns a valid "signature" object (JSON) for the test key.
func mailgunTestSignature() string {
	return mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, mailgunTestSign(testMailgunSigningKey, testMailgunTimestamp, testMailgunToken))
}

// mailgunTestSignatureWith builds the "signature" object from raw JSON timestamp, token and signature.
func mailgunTestSignatureWith(timestampJSON string, token string, signature string) string {
	return `{"timestamp":` + timestampJSON + `,"token":"` + token + `","signature":"` + signature + `"}`
}

// mailgunTestPayloadSigned builds a webhook body with fixed, synthetic values
// and the given "signature" object (JSON, empty for none).
func mailgunTestPayloadSigned(event string, severity string, vars string, wrapped bool, signature string) []byte {
	ev := `"event":"` + event + `","id":"synthetic-event-id","timestamp":1770146431.65,"recipient":"user@example.com","message":{"headers":{"message-id":"synthetic-message-id@example.com"}},"user-variables":` + vars
	if severity != "" {
		ev += `,"severity":"` + severity + `","reason":"generic"`
	}
	sig := ""
	if signature != "" {
		sig = `"signature":` + signature
	}
	if wrapped {
		if sig != "" {
			sig += ","
		}
		return []byte(`{` + sig + `"event-data":{` + ev + `}}`)
	}
	if sig != "" {
		sig += ","
	}
	return []byte(`{` + sig + ev + `}`)
}

// mailgunTestPayload builds a webhook body signed with the test key.
func mailgunTestPayload(event string, severity string, vars string, wrapped bool) []byte {
	return mailgunTestPayloadSigned(event, severity, vars, wrapped, mailgunTestSignature())
}

func Test_hookMailgun_status(t *testing.T) {
	id := uuid.FromStringOrNil(testMailgunEmailID)
	vars := `{"voipbin_message_id":"` + testMailgunEmailID + `"}`

	tests := []struct {
		name string
		data []byte

		expectStatus email.Status
		expectFrom   []email.Status
	}{
		{"accepted", mailgunTestPayload("accepted", "", vars, true), email.StatusProcessed, []email.Status{email.StatusNone, email.StatusInitiated}},
		{"delivered", mailgunTestPayload("delivered", "", vars, true), email.StatusDelivered, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered}},
		{"failed_temporary", mailgunTestPayload("failed", "temporary", vars, true), email.StatusDeffered, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed}},
		{"failed_permanent", mailgunTestPayload("failed", "permanent", vars, true), email.StatusBounce, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered, email.StatusDelivered}},
		{"opened", mailgunTestPayload("opened", "", vars, true), email.StatusOpen, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered, email.StatusDelivered, email.StatusBounce, email.StatusDropped}},
		{"clicked", mailgunTestPayload("clicked", "", vars, true), email.StatusClick, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered, email.StatusDelivered, email.StatusBounce, email.StatusDropped, email.StatusOpen}},
		{"unsubscribed", mailgunTestPayload("unsubscribed", "", vars, true), email.StatusUnsubscribe, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered, email.StatusDelivered, email.StatusBounce, email.StatusDropped, email.StatusOpen, email.StatusClick}},
		{"complained", mailgunTestPayload("complained", "", vars, true), email.StatusSpamreport, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered, email.StatusDelivered, email.StatusBounce, email.StatusDropped, email.StatusOpen, email.StatusClick, email.StatusUnsubscribe}},
		{"flat_body_without_event_data", mailgunTestPayload("delivered", "", vars, false), email.StatusDelivered, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &emailHandler{
				db:                       mockDB,
				mailgunWebhookSigningKey: staticSecret(testMailgunSigningKey),
				timeNow:                  testMailgunNow,
				notifyHandler:            mockNotify,
			}
			ctx := context.Background()
			customerID := uuid.FromStringOrNil("5e8f0f0a-8888-4888-8888-0000000000c1")
			updated := &email.Email{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: tt.expectStatus}

			mockDB.EXPECT().EmailUpdateStatusFrom(ctx, id, tt.expectStatus, tt.expectFrom).Return(true, nil)
			mockDB.EXPECT().EmailGet(ctx, id).Return(updated, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, email.EventTypeUpdated, updated)

			if err := h.hookMailgun(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hookMailgun_noUpdate(t *testing.T) {
	id := uuid.FromStringOrNil(testMailgunEmailID)
	vars := `{"voipbin_message_id":"` + testMailgunEmailID + `"}`

	tests := []struct {
		name string
		data []byte

		// the status update is attempted and reports no change
		expectAttempt bool
	}{
		{"stale_or_duplicate_event", mailgunTestPayload("accepted", "", vars, true), true},
		{"event_without_status", mailgunTestPayload("rejected", "", vars, true), false},
		{"failed_with_unknown_severity", mailgunTestPayload("failed", "", vars, true), false},
		{"user_variables_empty_array", mailgunTestPayload("delivered", "", `[]`, true), false},
		{"user_variables_missing_id", mailgunTestPayload("delivered", "", `{"other":"x"}`, true), false},
		{"user_variables_invalid_id", mailgunTestPayload("delivered", "", `{"voipbin_message_id":"not-a-uuid"}`, true), false},
		{"user_variables_not_a_string", mailgunTestPayload("delivered", "", `{"voipbin_message_id":123}`, true), false},
		{"user_variables_absent", []byte(`{"signature":` + mailgunTestSignature() + `,"event":"delivered"}`), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// strict mocks: no notify, no get
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &emailHandler{
				db:                       mockDB,
				mailgunWebhookSigningKey: staticSecret(testMailgunSigningKey),
				timeNow:                  testMailgunNow,
				notifyHandler:            notifyhandler.NewMockNotifyHandler(mc),
			}
			ctx := context.Background()

			if tt.expectAttempt {
				mockDB.EXPECT().EmailUpdateStatusFrom(ctx, id, email.StatusProcessed, gomock.Any()).Return(false, nil)
			}

			if err := h.hookMailgun(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hookMailgun_errors(t *testing.T) {
	id := uuid.FromStringOrNil(testMailgunEmailID)
	vars := `{"voipbin_message_id":"` + testMailgunEmailID + `"}`
	data := mailgunTestPayload("delivered", "", vars, true)

	tests := []struct {
		name  string
		data  []byte
		setup func(db *dbhandler.MockDBHandler)
	}{
		{"invalid_json", []byte(`not json`), func(db *dbhandler.MockDBHandler) {}},
		{"no_event", []byte(`{"signature":` + mailgunTestSignature() + `}`), func(db *dbhandler.MockDBHandler) {}},
		{"update_fails", data, func(db *dbhandler.MockDBHandler) {
			db.EXPECT().EmailUpdateStatusFrom(gomock.Any(), id, email.StatusDelivered, gomock.Any()).Return(false, errors.New("db down"))
		}},
		{"get_fails_after_update", data, func(db *dbhandler.MockDBHandler) {
			db.EXPECT().EmailUpdateStatusFrom(gomock.Any(), id, email.StatusDelivered, gomock.Any()).Return(true, nil)
			db.EXPECT().EmailGet(gomock.Any(), id).Return(nil, errors.New("select failed"))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &emailHandler{
				db:                       mockDB,
				mailgunWebhookSigningKey: staticSecret(testMailgunSigningKey),
				timeNow:                  testMailgunNow,
				notifyHandler:            notifyhandler.NewMockNotifyHandler(mc),
			}
			tt.setup(mockDB)

			if err := h.hookMailgun(context.Background(), tt.data); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_statusesBefore(t *testing.T) {
	tests := []struct {
		status email.Status
		expect []email.Status
	}{
		{email.StatusProcessed, []email.Status{email.StatusNone, email.StatusInitiated}},
		{email.StatusBounce, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered, email.StatusDelivered}},
		{email.StatusDropped, []email.Status{email.StatusNone, email.StatusInitiated, email.StatusProcessed, email.StatusDeffered, email.StatusDelivered}},
		{email.StatusInitiated, []email.Status{}},
		{email.Status("unknown"), nil},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if res := statusesBefore(tt.status); !reflect.DeepEqual(res, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func Test_Hook_UnknownURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{
			name: "fails_on_unknown_hook_uri",
			uri:  "hook.voipbin.net/v1.0/emails/unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			err := h.Hook(ctx, tt.uri, []byte{})
			if err == nil {
				t.Errorf("Expected error for unknown URI")
			}
		})
	}
}

func Test_hookSendgrid_InvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "fails_on_invalid_json",
			data: []byte(`invalid json`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			err := h.hookSendgrid(ctx, tt.data)
			if err == nil {
				t.Errorf("Expected error for invalid JSON")
			}
		})
	}
}
