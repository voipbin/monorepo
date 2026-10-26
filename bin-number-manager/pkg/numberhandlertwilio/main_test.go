package numberhandlertwilio

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/requestexternal"
	"monorepo/bin-number-manager/pkg/requestexternal/models/twilio"
)

const (
	testSID      = "AC00000000000000000000000000000000"
	testTrunkSID = "TK00000000000000000000000000000000"
	testNumSID   = "PN00000000000000000000000000000000"
	testToken    = "twilio-test-token-0000"
	testNumber   = "+15550000001"
)

func newTestHandler(m *requestexternal.MockRequestExternal) *numberHandlerTwilio {
	return &numberHandlerTwilio{
		sid:             testSID,
		trunkSID:        testTrunkSID,
		token:           testToken,
		requestExternal: m,
	}
}

func Test_NewNumberHandler(t *testing.T) {
	h := NewNumberHandler(nil, nil, testSID, testTrunkSID, testToken)

	res, ok := h.(*numberHandlerTwilio)
	if !ok {
		t.Fatalf("Wrong type. got: %T", h)
	}
	if res.sid != testSID || res.trunkSID != testTrunkSID || res.token != testToken {
		t.Errorf("Wrong injected values.")
	}
	if res.requestExternal == nil {
		t.Errorf("Wrong match. expect: request external, got: nil")
	}
}

func Test_NewNumberHandler_trimsTheStoredValues(t *testing.T) {
	h := NewNumberHandler(nil, nil, " "+testSID+"\n", "\t"+testTrunkSID+" ", testToken)

	res := h.(*numberHandlerTwilio)
	if res.sid != testSID || res.trunkSID != testTrunkSID || res.token != testToken {
		t.Errorf("Wrong match. the sid and the trunk_sid must be trimmed and the token kept as it is.")
	}
}

func Test_ValidateSID(t *testing.T) {
	tests := []struct {
		sid    string
		expect bool
	}{
		{testSID, true},
		{"ACabcdefABCDEF0123456789abcdef0123", true},
		{"", false},
		{"AC0000", false},
		{"SK00000000000000000000000000000000", false},
		{testSID + "x", false},
		{"AC0000000000000000000000000000000g", false},
		{"AC00000000000000000000000000000000/../x", false},
	}
	for _, tt := range tests {
		err := ValidateSID(tt.sid)
		if (err == nil) != tt.expect {
			t.Errorf("Wrong match. sid: %q, expect ok: %v, got: %v", tt.sid, tt.expect, err)
		}
		if err != nil && tt.sid != "" && strings.Contains(err.Error(), tt.sid) {
			t.Errorf("The error holds the sid. got: %v", err)
		}
	}
}

func Test_NumberPurchase(t *testing.T) {
	tests := []struct {
		name string

		num      string
		trunkSID string

		responseCall  bool
		responseRes   *twilio.IncomingPhoneNumber
		responseErr   error
		expectRes     *providernumber.ProviderNumber
		expectErrKind error // provider.ErrPurchaseRejected, provider.ErrPurchaseUnconfirmed or nil
	}{
		{
			name:         "success",
			num:          testNumber,
			trunkSID:     testTrunkSID,
			responseCall: true,
			responseRes:  &twilio.IncomingPhoneNumber{SID: testNumSID, PhoneNumber: testNumber, EmergencyStatus: "Inactive"},
			expectRes:    &providernumber.ProviderNumber{ID: testNumSID, Status: number.StatusActive},
		},
		{
			name:         "success with emergency active",
			num:          testNumber,
			trunkSID:     testTrunkSID,
			responseCall: true,
			responseRes:  &twilio.IncomingPhoneNumber{SID: testNumSID, EmergencyStatus: "Active"},
			expectRes:    &providernumber.ProviderNumber{ID: testNumSID, Status: number.StatusActive, EmergencyEnabled: true},
		},
		{
			name:          "answer without a number sid is unconfirmed",
			num:           testNumber,
			trunkSID:      testTrunkSID,
			responseCall:  true,
			responseRes:   &twilio.IncomingPhoneNumber{},
			expectErrKind: provider.ErrPurchaseUnconfirmed,
		},
		{
			name:          "answer with a malformed number sid is unconfirmed",
			num:           testNumber,
			trunkSID:      testTrunkSID,
			responseCall:  true,
			responseRes:   &twilio.IncomingPhoneNumber{SID: "PN123"},
			expectErrKind: provider.ErrPurchaseUnconfirmed,
		},
		{
			name:          "unparsable success answer is unconfirmed",
			num:           testNumber,
			trunkSID:      testTrunkSID,
			responseCall:  true,
			responseErr:   errors.New("could not parse the response"),
			expectErrKind: provider.ErrPurchaseUnconfirmed,
		},
		{
			name:          "transport error is unconfirmed",
			num:           testNumber,
			trunkSID:      testTrunkSID,
			responseCall:  true,
			responseErr:   errors.New("context deadline exceeded"),
			expectErrKind: provider.ErrPurchaseUnconfirmed,
		},
		{name: "http 400 is rejected", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 400, ErrorCode: 21422}, expectErrKind: provider.ErrPurchaseRejected},
		{name: "http 401 is rejected", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 401, ErrorCode: 20003}, expectErrKind: provider.ErrPurchaseRejected},
		{name: "http 403 is rejected", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 403}, expectErrKind: provider.ErrPurchaseRejected},
		{name: "http 404 is rejected", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 404}, expectErrKind: provider.ErrPurchaseRejected},
		{name: "http 422 is rejected", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 422}, expectErrKind: provider.ErrPurchaseRejected},
		{name: "http 408 is unconfirmed", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 408}, expectErrKind: provider.ErrPurchaseUnconfirmed},
		{name: "http 409 is unconfirmed", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 409}, expectErrKind: provider.ErrPurchaseUnconfirmed},
		{name: "http 429 is unconfirmed", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 429, ErrorCode: 20429}, expectErrKind: provider.ErrPurchaseUnconfirmed},
		{name: "http 500 is unconfirmed", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 500}, expectErrKind: provider.ErrPurchaseUnconfirmed},
		{name: "http 503 is unconfirmed", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 503}, expectErrKind: provider.ErrPurchaseUnconfirmed},
		{name: "http 302 is unconfirmed", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 302}, expectErrKind: provider.ErrPurchaseUnconfirmed},
		{name: "http 202 is unconfirmed", num: testNumber, trunkSID: testTrunkSID, responseCall: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 202}, expectErrKind: provider.ErrPurchaseUnconfirmed},
		{name: "no trunk_sid is rejected without a request", num: testNumber, trunkSID: "", expectErrKind: provider.ErrPurchaseRejected},
		{name: "blank trunk_sid is rejected without a request", num: testNumber, trunkSID: "  ", expectErrKind: provider.ErrPurchaseRejected},
		{name: "malformed trunk_sid is rejected without a request", num: testNumber, trunkSID: "TK123", expectErrKind: provider.ErrPurchaseRejected},
		{name: "malformed number is rejected without a request", num: "155****0001", trunkSID: testTrunkSID, expectErrKind: provider.ErrPurchaseRejected},
		{name: "empty number is rejected without a request", num: "", trunkSID: testTrunkSID, expectErrKind: provider.ErrPurchaseRejected},
		{name: "a leading zero is rejected without a request", num: "+01234567", trunkSID: testTrunkSID, expectErrKind: provider.ErrPurchaseRejected},
		{name: "6 digits is too short", num: "+123456", trunkSID: testTrunkSID, expectErrKind: provider.ErrPurchaseRejected},
		{name: "16 digits is too long", num: "+1234567890123456", trunkSID: testTrunkSID, expectErrKind: provider.ErrPurchaseRejected},
		{name: "7 digits is accepted", num: "+1234567", trunkSID: testTrunkSID, responseCall: true, responseRes: &twilio.IncomingPhoneNumber{SID: testNumSID}, expectRes: &providernumber.ProviderNumber{ID: testNumSID, Status: number.StatusActive}},
		{name: "15 digits is accepted", num: "+123456789012345", trunkSID: testTrunkSID, responseCall: true, responseRes: &twilio.IncomingPhoneNumber{SID: testNumSID}, expectRes: &providernumber.ProviderNumber{ID: testNumSID, Status: number.StatusActive}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)
			h := newTestHandler(mockExternal)
			h.trunkSID = tt.trunkSID

			if tt.responseCall {
				mockExternal.EXPECT().TwilioIncomingPhoneNumbersPost(testSID, testToken, tt.num, tt.trunkSID).Return(tt.responseRes, tt.responseErr)
			}

			res, err := h.NumberPurchase(tt.num)

			if tt.expectErrKind != nil {
				if !errors.Is(err, tt.expectErrKind) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErrKind, err)
				}
				other := provider.ErrPurchaseRejected
				if tt.expectErrKind == provider.ErrPurchaseRejected {
					other = provider.ErrPurchaseUnconfirmed
				}
				if errors.Is(err, other) {
					t.Errorf("The error must be one kind only. got: %v", err)
				}
				if res != nil {
					t.Errorf("Wrong result. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberRelease(t *testing.T) {
	releaseErr := &requestexternal.HTTPStatusError{StatusCode: 404, ErrorCode: 20404}

	tests := []struct {
		name string

		refID string

		responseCall bool
		responseErr  error
		expectErr    bool
	}{
		{name: "success", refID: testNumSID, responseCall: true},
		{name: "provider error is returned", refID: testNumSID, responseCall: true, responseErr: releaseErr, expectErr: true},
		{name: "empty reference id makes no request", refID: "", expectErr: true},
		{name: "a telnyx style id makes no request", refID: "1579827332531618841", expectErr: true},
		{name: "a path like id makes no request", refID: "../Accounts/x", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)
			h := newTestHandler(mockExternal)

			if tt.responseCall {
				mockExternal.EXPECT().TwilioIncomingPhoneNumbersIDDelete(testSID, testToken, tt.refID).Return(tt.responseErr)
			}

			err := h.NumberRelease(context.Background(), &number.Number{ProviderReferenceID: tt.refID})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.responseErr != nil && !errors.Is(err, tt.responseErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.responseErr, err)
			}
		})
	}
}

func Test_NumberUpdateTags(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no request is expected: the mock fails the test on any call
	h := newTestHandler(requestexternal.NewMockRequestExternal(mc))

	if err := h.NumberUpdateTags(context.Background(), &number.Number{ProviderReferenceID: testNumSID}, []string{"a", "b"}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_GetAvailableNumbers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)
	h := newTestHandler(mockExternal)

	mockExternal.EXPECT().TwilioAvailableNumberGets(testSID, testToken, "US", uint(2)).Return([]*twilio.AvailableNumber{
		{PhoneNumber: "+15550000001", IsoCountry: "US", Region: "CA", PostalCode: "94000", Capabilities: twilio.Capabilities{Voice: true, SMS: true, MMS: true, Fax: true}},
		{PhoneNumber: "+15550000002", IsoCountry: "US"},
	}, nil)

	res, err := h.GetAvailableNumbers("US", 2)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*availablenumber.AvailableNumber{
		{
			Number: "+15550000001", ProviderName: number.ProviderNameTwilio, Country: "US", Region: "CA", PostalCode: "94000",
			Features: []availablenumber.Feature{availablenumber.FeatureVoice, availablenumber.FeatureSMS, availablenumber.FeatureMMS, availablenumber.FeatureFax},
		},
		{Number: "+15550000002", ProviderName: number.ProviderNameTwilio, Country: "US"},
	}
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}

	mockExternal.EXPECT().TwilioAvailableNumberGets(testSID, testToken, "US", uint(1)).Return(nil, errors.New("search failed"))
	if _, err := h.GetAvailableNumbers("US", 1); err == nil {
		t.Errorf("Wrong match. expect: error")
	}

	for _, country := range []string{"", "U", "USA", "../"} {
		if _, err := h.GetAvailableNumbers(country, 1); err == nil {
			t.Errorf("Wrong match. country %q must be refused without a request", country)
		}
	}
}

// the logs of the handler must never hold the token or the account sid, whatever the outcome
func Test_noCredentialInLogs(t *testing.T) {
	hook := logrustest.NewGlobal()
	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() { logrus.SetLevel(prevLevel) })

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)
	h := newTestHandler(mockExternal)
	ctx := context.Background()

	mockExternal.EXPECT().TwilioIncomingPhoneNumbersPost(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&twilio.IncomingPhoneNumber{SID: testNumSID}, nil)
	_, _ = h.NumberPurchase(testNumber)

	mockExternal.EXPECT().TwilioIncomingPhoneNumbersPost(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &requestexternal.HTTPStatusError{StatusCode: 401, ErrorCode: 20003})
	_, _ = h.NumberPurchase(testNumber)

	mockExternal.EXPECT().TwilioIncomingPhoneNumbersIDDelete(gomock.Any(), gomock.Any(), gomock.Any()).Return(&requestexternal.HTTPStatusError{StatusCode: 500})
	_ = h.NumberRelease(ctx, &number.Number{ProviderReferenceID: testNumSID})

	// the paths that refuse a purchase before any request
	_, _ = h.NumberPurchase("not-a-number")
	h.trunkSID = "bad-" + testTrunkSID
	_, _ = h.NumberPurchase(testNumber)
	h.trunkSID = testTrunkSID

	mockExternal.EXPECT().TwilioAvailableNumberGets(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("failed"))
	_, _ = h.GetAvailableNumbers("US", 1)

	_ = h.NumberUpdateTags(ctx, &number.Number{}, []string{"a"})

	if len(hook.AllEntries()) == 0 {
		t.Fatalf("Wrong match. expect: log entries")
	}

	for _, e := range hook.AllEntries() {
		line := fmt.Sprintf("%s %v", e.Message, e.Data)
		if strings.Contains(line, testToken) || strings.Contains(line, testSID) || strings.Contains(line, testTrunkSID) {
			t.Errorf("A log entry holds a credential: %s", line)
		}
	}

}

func Test_classifyOrderError_sentinels(t *testing.T) {
	rejected := classifyOrderError(&requestexternal.HTTPStatusError{StatusCode: 400})
	unconfirmed := classifyOrderError(errors.New("plain"))

	if !errors.Is(rejected, provider.ErrPurchaseRejected) || errors.Is(rejected, provider.ErrPurchaseUnconfirmed) {
		t.Errorf("Wrong kind. got: %v", rejected)
	}
	if !errors.Is(unconfirmed, provider.ErrPurchaseUnconfirmed) || errors.Is(unconfirmed, provider.ErrPurchaseRejected) {
		t.Errorf("Wrong kind. got: %v", unconfirmed)
	}
}

func Test_NumberPurchase_logMarkers(t *testing.T) {
	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() { logrus.SetLevel(prevLevel) })

	hasMarker := func(hook *logrustest.Hook, marker string) bool {
		for _, e := range hook.AllEntries() {
			if strings.Contains(e.Message, marker) {
				return true
			}
		}
		return false
	}

	tests := []struct {
		name        string
		trunkSID    string
		responseErr error
		call        bool
		noSID       bool
		marker      string
		absent      string
	}{
		{name: "401 is an auth rejection", trunkSID: testTrunkSID, call: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 401}, marker: "PROVIDER_AUTH_REJECTED"},
		{name: "403 is an auth rejection", trunkSID: testTrunkSID, call: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 403}, marker: "PROVIDER_AUTH_REJECTED"},
		{name: "404 is an auth rejection", trunkSID: testTrunkSID, call: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 404}, marker: "PROVIDER_AUTH_REJECTED"},
		{name: "400 is not an auth rejection", trunkSID: testTrunkSID, call: true, responseErr: &requestexternal.HTTPStatusError{StatusCode: 400}, absent: "PROVIDER_AUTH_REJECTED"},
		{name: "an answer without a number sid is unconfirmed", trunkSID: testTrunkSID, call: true, noSID: true, marker: "PURCHASE_UNCONFIRMED"},
		{name: "no trunk_sid is a config error", trunkSID: "", marker: "PROVIDER_CONFIG_INVALID"},
		{name: "a malformed trunk_sid is a config error", trunkSID: "TK1", marker: "PROVIDER_CONFIG_INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logrustest.NewGlobal()

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)
			h := newTestHandler(mockExternal)
			h.trunkSID = tt.trunkSID

			if tt.call && tt.noSID {
				mockExternal.EXPECT().TwilioIncomingPhoneNumbersPost(testSID, testToken, testNumber, tt.trunkSID).Return(&twilio.IncomingPhoneNumber{}, nil)
			} else if tt.call {
				mockExternal.EXPECT().TwilioIncomingPhoneNumbersPost(testSID, testToken, testNumber, tt.trunkSID).Return(nil, tt.responseErr)
			}

			_, _ = h.NumberPurchase(testNumber)

			if tt.marker != "" && !hasMarker(hook, tt.marker) {
				t.Errorf("Wrong match. expect the log marker %s", tt.marker)
			}
			if tt.absent != "" && hasMarker(hook, tt.absent) {
				t.Errorf("Wrong match. the log marker %s must be absent", tt.absent)
			}
		})
	}
}
