package emailhandler

import (
	"context"
	"strconv"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"go.uber.org/mock/gomock"
)

func Test_hookMailgun_signature_rejected(t *testing.T) {
	vars := `{"voipbin_message_id":"` + testMailgunEmailID + `"}`
	goodSig := mailgunTestSign(testMailgunSigningKey, testMailgunTimestamp, testMailgunToken)
	otherSig := mailgunTestSign("other-signing-key", testMailgunTimestamp, testMailgunToken)

	now := testMailgunNow().Unix()
	tsOld := strconv.FormatInt(now-int64(mailgunSignatureMaxAge/time.Second)-1, 10)
	tsFuture := strconv.FormatInt(now+int64(mailgunSignatureMaxSkew/time.Second)+1, 10)

	tests := []struct {
		name    string
		key     string
		wrapped bool
		sig     string
	}{
		{"wrong_signature", testMailgunSigningKey, true, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, otherSig)},
		{"wrong_signature_flat", testMailgunSigningKey, false, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, otherSig)},
		{"signature_missing", testMailgunSigningKey, true, ""},
		{"signature_missing_flat", testMailgunSigningKey, false, ""},
		{"signature_empty_value", testMailgunSigningKey, true, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, "")},
		{"signature_not_hex", testMailgunSigningKey, true, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, "zz"+goodSig[2:])},
		{"signature_truncated", testMailgunSigningKey, true, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, goodSig[:10])},
		{"token_empty", testMailgunSigningKey, true, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, "", mailgunTestSign(testMailgunSigningKey, testMailgunTimestamp, ""))},
		{"token_changed", testMailgunSigningKey, true, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, "another-token", goodSig)},
		{"timestamp_changed", testMailgunSigningKey, true, mailgunTestSignatureWith(`"1770146432"`, testMailgunToken, goodSig)},
		{"timestamp_missing", testMailgunSigningKey, true, `{"token":"` + testMailgunToken + `","signature":"` + goodSig + `"}`},
		{"timestamp_not_a_number", testMailgunSigningKey, true, mailgunTestSignatureWith(`"abc"`, testMailgunToken, mailgunTestSign(testMailgunSigningKey, "abc", testMailgunToken))},
		{"timestamp_fraction", testMailgunSigningKey, true, mailgunTestSignatureWith(`"1770146431.5"`, testMailgunToken, mailgunTestSign(testMailgunSigningKey, "1770146431.5", testMailgunToken))},
		{"timestamp_object", testMailgunSigningKey, true, mailgunTestSignatureWith(`{}`, testMailgunToken, goodSig)},
		{"timestamp_too_old", testMailgunSigningKey, true, mailgunTestSignatureWith(`"`+tsOld+`"`, testMailgunToken, mailgunTestSign(testMailgunSigningKey, tsOld, testMailgunToken))},
		{"timestamp_too_far_in_future", testMailgunSigningKey, true, mailgunTestSignatureWith(`"`+tsFuture+`"`, testMailgunToken, mailgunTestSign(testMailgunSigningKey, tsFuture, testMailgunToken))},
		{"signature_not_an_object", testMailgunSigningKey, true, `"text"`},
		{"empty_key_with_signature_of_empty_key", "", true, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, mailgunTestSign("", testMailgunTimestamp, testMailgunToken))},
		{"empty_key_with_valid_signature", "", true, mailgunTestSignature()},
		{"empty_key_flat", "", false, mailgunTestSignature()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// strict mocks: any db call or webhook publish fails the test.
			h := &emailHandler{
				db:                       dbhandler.NewMockDBHandler(mc),
				notifyHandler:            notifyhandler.NewMockNotifyHandler(mc),
				mailgunWebhookSigningKey: tt.key,
				timeNow:                  testMailgunNow,
			}

			data := mailgunTestPayloadSigned("delivered", "", vars, tt.wrapped, tt.sig)
			if err := h.hookMailgun(context.Background(), data); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_hookMailgun_signature_accepted(t *testing.T) {
	vars := `{"voipbin_message_id":"` + testMailgunEmailID + `"}`

	now := testMailgunNow().Unix()
	tsEdgeOld := strconv.FormatInt(now-int64(mailgunSignatureMaxAge/time.Second), 10)
	tsEdgeFuture := strconv.FormatInt(now+int64(mailgunSignatureMaxSkew/time.Second), 10)

	tests := []struct {
		name    string
		wrapped bool
		sig     string
	}{
		{"wrapped", true, mailgunTestSignature()},
		{"flat", false, mailgunTestSignature()},
		{"timestamp_as_number", true, mailgunTestSignatureWith(testMailgunTimestamp, testMailgunToken, mailgunTestSign(testMailgunSigningKey, testMailgunTimestamp, testMailgunToken))},
		{"uppercase_hex", true, mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, upper(mailgunTestSign(testMailgunSigningKey, testMailgunTimestamp, testMailgunToken)))},
		{"timestamp_at_oldest_edge", true, mailgunTestSignatureWith(`"`+tsEdgeOld+`"`, testMailgunToken, mailgunTestSign(testMailgunSigningKey, tsEdgeOld, testMailgunToken))},
		{"timestamp_at_future_edge", true, mailgunTestSignatureWith(`"`+tsEdgeFuture+`"`, testMailgunToken, mailgunTestSign(testMailgunSigningKey, tsEdgeFuture, testMailgunToken))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &emailHandler{
				db:                       mockDB,
				notifyHandler:            notifyhandler.NewMockNotifyHandler(mc),
				mailgunWebhookSigningKey: testMailgunSigningKey,
				timeNow:                  testMailgunNow,
			}

			// the status update is reached (and reports no change), so the signature passed.
			mockDB.EXPECT().EmailUpdateStatusFrom(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)

			data := mailgunTestPayloadSigned("accepted", "", vars, tt.wrapped, tt.sig)
			if err := h.hookMailgun(context.Background(), data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hookMailgun_signature_defaultClock(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// timeNow is nil: the real clock is used, so the fixed old timestamp is rejected.
	h := &emailHandler{
		db:                       dbhandler.NewMockDBHandler(mc),
		notifyHandler:            notifyhandler.NewMockNotifyHandler(mc),
		mailgunWebhookSigningKey: testMailgunSigningKey,
	}

	data := mailgunTestPayload("delivered", "", `{"voipbin_message_id":"`+testMailgunEmailID+`"}`, true)
	if err := h.hookMailgun(context.Background(), data); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Hook_mailgun_rejected(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &emailHandler{
		db:                       dbhandler.NewMockDBHandler(mc),
		notifyHandler:            notifyhandler.NewMockNotifyHandler(mc),
		mailgunWebhookSigningKey: testMailgunSigningKey,
		timeNow:                  testMailgunNow,
	}

	data := mailgunTestPayloadSigned("delivered", "", `{"voipbin_message_id":"`+testMailgunEmailID+`"}`, true, "")
	if err := h.Hook(context.Background(), "hook.voipbin.net/v1.0/emails/mailgun", data); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func upper(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'f' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}
