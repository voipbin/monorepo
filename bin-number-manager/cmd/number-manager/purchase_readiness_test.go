package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/pkg/providerhandler"
)

func Test_logPurchaseReadiness(t *testing.T) {
	tests := []struct {
		name string

		candidates []*providerhandler.Candidate
		err        error

		expectLevel string
		expectText  string
	}{
		{
			name:        "no usable provider is a warning",
			candidates:  nil,
			expectLevel: "level=warning",
			expectText:  "No provider is usable for a number purchase",
		},
		{
			name:        "a usable provider is info",
			candidates:  []*providerhandler.Candidate{{Name: "telnyx"}},
			expectLevel: "level=info",
			expectText:  "count: 1",
		},
		{
			name:        "a read error is logged and does not stop the start",
			err:         errors.New("db down"),
			expectLevel: "level=error",
			expectText:  "Could not check the providers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockProvider := providerhandler.NewMockProviderHandler(mc)
			mockProvider.EXPECT().PurchaseCandidates(gomock.Any()).Return(tt.candidates, tt.err)

			var buf bytes.Buffer
			orig := logrus.StandardLogger().Out
			logrus.SetOutput(&buf)
			defer logrus.SetOutput(orig)

			logPurchaseReadiness(context.Background(), mockProvider)

			out := buf.String()
			if !strings.Contains(out, tt.expectLevel) || !strings.Contains(out, tt.expectText) {
				t.Errorf("Wrong match. expect: %s and %q, got: %s", tt.expectLevel, tt.expectText, out)
			}
		})
	}
}
