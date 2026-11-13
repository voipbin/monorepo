package listenhandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	hmhook "monorepo/bin-hook-manager/models/hook"

	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/internal/logtest"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/paddlehandler"
)

const (
	testHookSignature = "ts=1700000000;h1=0123456789abcdef"
	testHookBody      = `{"event_id":"evt_sig_01","event_type":"transaction.payment_failed","data":{}}`
)

func buildSignedHookData(t *testing.T, body, signature string) []byte {
	t.Helper()

	data, err := json.Marshal(hmhook.Hook{
		ReceivedURI:       "hook.example.com/billing/paddle",
		ReceivedData:      []byte(body),
		ReceivedSignature: signature,
	})
	if err != nil {
		t.Fatalf("could not marshal hook: %v", err)
	}

	return data
}

func Test_processV1HooksPaddlePost_signature(t *testing.T) {
	tests := []struct {
		name string

		verifyErr error
		body      string

		expectVerify bool
		expectStatus int
	}{
		{name: "valid signature continues to the event", body: testHookBody, expectVerify: true, expectStatus: 200},
		{name: "rejected signature is 401 and the event is not processed", verifyErr: fmt.Errorf("%w: mismatch", paddlehandler.ErrSignatureInvalid), body: testHookBody, expectVerify: true, expectStatus: 401},
		{name: "unreadable key is 500 and the event is not processed", verifyErr: fmt.Errorf("%w: boom", paddlehandler.ErrKeyUnavailable), body: testHookBody, expectVerify: true, expectStatus: 500},
		{name: "any other error is 500", verifyErr: errors.New("unexpected"), body: testHookBody, expectVerify: true, expectStatus: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			// the account handler is a strict mock: any call of it fails the test
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockPaddle := paddlehandler.NewMockPaddleHandler(mc)

			h := &listenHandler{sockHandler: mockSock, accountHandler: mockAccount, paddleHandler: mockPaddle}

			// the verification gets exactly the signature header and the raw body of the hook
			mockPaddle.EXPECT().VerifyWebhook(gomock.Any(), testHookSignature, []byte(tt.body)).Return(tt.verifyErr)

			res, err := h.processRequest(&sock.Request{
				URI:    "/v1/hooks/paddle",
				Method: sock.RequestMethodPost,
				Data:   buildSignedHookData(t, tt.body, testHookSignature),
			})
			if err != nil || res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %v %v", tt.expectStatus, res, err)
			}
		})
	}
}

// A hook that can not be unmarshaled is a 400 before any signature check.
func Test_processV1HooksPaddlePost_signatureNotCheckedForBrokenHook(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// the paddle handler is a strict mock: a call of VerifyWebhook fails the test
	h := &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		accountHandler: accounthandler.NewMockAccountHandler(mc),
		paddleHandler:  paddlehandler.NewMockPaddleHandler(mc),
	}

	res, err := h.processRequest(&sock.Request{URI: "/v1/hooks/paddle", Method: sock.RequestMethodPost, Data: []byte(`{broken`)})
	if err != nil || res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: 400, got: %v %v", res, err)
	}
}

// The log lines that the signature check writes hold neither the signature nor the body.
func Test_processV1HooksPaddlePost_signatureIsNotLogged(t *testing.T) {
	const marker = "marker-signature-value-0123"

	tests := []struct {
		name      string
		verifyErr error
	}{
		{name: "rejected", verifyErr: fmt.Errorf("%w: mismatch", paddlehandler.ErrSignatureInvalid)},
		{name: "unreadable key", verifyErr: fmt.Errorf("%w: boom", paddlehandler.ErrKeyUnavailable)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPaddle := paddlehandler.NewMockPaddleHandler(mc)
			h := &listenHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				accountHandler: accounthandler.NewMockAccountHandler(mc),
				paddleHandler:  mockPaddle,
			}

			signature := "ts=1700000000;h1=" + marker
			mockPaddle.EXPECT().VerifyWebhook(gomock.Any(), signature, gomock.Any()).Return(tt.verifyErr)

			data := buildSignedHookData(t, testHookBody, signature)
			hook := hmhook.Hook{}
			if err := json.Unmarshal(data, &hook); err != nil {
				t.Fatal(err)
			}

			logs := logtest.New(t)
			// call the handler itself: processRequest writes its own entry with the whole request
			// (a known line that this change does not add to), so this test looks at the lines of
			// the check only
			res, err := h.processV1HooksPaddlePost(t.Context(), &sock.Request{URI: "/v1/hooks/paddle", Method: sock.RequestMethodPost, Data: data})
			if err != nil || res == nil || res.StatusCode == 200 {
				t.Fatalf("Wrong match. got: %v %v", res, err)
			}

			dump := logs.Dump()
			if logs.Len() == 0 {
				t.Errorf("Wrong match. expect: a log line for the rejection, got: none")
			}
			if strings.Contains(dump, marker) || strings.Contains(dump, "evt_sig_01") {
				t.Errorf("Wrong match. a log line holds the signature or the body: %s", dump)
			}
		})
	}
}
