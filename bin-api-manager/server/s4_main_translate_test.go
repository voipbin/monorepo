package server

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_translateToVoipbinError_Extra(t *testing.T) {
	tests := []struct {
		name       string
		in         error
		wantStatus cerrors.Status
		wantReason string
		wantMsg    string
	}{
		{
			name:       "nil error becomes INTERNAL",
			in:         nil,
			wantStatus: cerrors.StatusInternal, wantReason: "INTERNAL", wantMsg: "An internal error occurred.",
		},
		{
			name:       "case closed",
			in:         fmt.Errorf("wrap: %w", serviceerrors.ErrCaseClosed),
			wantStatus: cerrors.StatusFailedPrecondition, wantReason: "CASE_CLOSED",
			wantMsg: "This case is closed. Call POST /v1.0/contact_cases/{id}/continue to reopen it before performing this action.",
		},
		{
			name:       "case destination not associated",
			in:         serviceerrors.ErrCaseDestinationNotAssociated,
			wantStatus: cerrors.StatusInvalidArgument, wantReason: "DESTINATION_NOT_ASSOCIATED",
			wantMsg: "destination is not associated with this case",
		},
		{
			name:       "case source not owned",
			in:         fmt.Errorf("wrap: %w", serviceerrors.ErrCaseSourceNotOwned),
			wantStatus: cerrors.StatusInvalidArgument, wantReason: "SOURCE_NOT_OWNED",
			wantMsg: "source is not an active number owned by this customer",
		},
		{
			name:       "context canceled",
			in:         fmt.Errorf("rpc: %w", context.Canceled),
			wantStatus: cerrors.StatusUnavailable, wantReason: "REQUEST_CANCELED", wantMsg: "The request was canceled.",
		},
		{
			name:       "context deadline exceeded",
			in:         fmt.Errorf("rpc: %w", context.DeadlineExceeded),
			wantStatus: cerrors.StatusUnavailable, wantReason: "REQUEST_TIMEOUT", wantMsg: "The request timed out.",
		},
		{
			name:       "unknown error falls to INTERNAL",
			in:         stderrors.New("boom"),
			wantStatus: cerrors.StatusInternal, wantReason: "INTERNAL", wantMsg: "An internal error occurred.",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			out := translateToVoipbinError(tt.in)
			if out == nil {
				t.Fatal("translateToVoipbinError returned nil")
			}
			if out.Status != tt.wantStatus {
				t.Errorf("status = %q want %q", out.Status, tt.wantStatus)
			}
			if out.Reason != tt.wantReason {
				t.Errorf("reason = %q want %q", out.Reason, tt.wantReason)
			}
			if out.Message != tt.wantMsg {
				t.Errorf("message = %q want %q", out.Message, tt.wantMsg)
			}
		})
	}
}

func Test_ConvertFlowManagerAction_TmExecuteFallback(t *testing.T) {
	const id = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		tmExec   *string
		wantTime *time.Time
	}{
		{
			name:     "legacy layout is parsed through the utilhandler fallback",
			tmExec:   strPtr("2024-03-05 10:20:30.123456"),
			wantTime: func() *time.Time { v := time.Date(2024, 3, 5, 10, 20, 30, 123456000, time.UTC); return &v }(),
		},
		{
			name:     "unparseable value leaves TMExecute nil",
			tmExec:   strPtr("not a time"),
			wantTime: nil,
		},
		{
			name:     "empty value leaves TMExecute nil",
			tmExec:   strPtr(""),
			wantTime: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			res := ConvertFlowManagerAction(openapi_server.FlowManagerAction{Id: id, Type: "answer", TmExecute: tt.tmExec})
			if tt.wantTime == nil {
				if res.TMExecute != nil {
					t.Errorf("TMExecute = %v want nil", res.TMExecute)
				}
				return
			}
			if res.TMExecute == nil || !res.TMExecute.Equal(*tt.wantTime) {
				t.Errorf("TMExecute = %v want %v", res.TMExecute, tt.wantTime)
			}
		})
	}
}

func Test_structToFilteredMap_Extra(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
		Num  int    `json:"num"`
	}

	t.Run("struct becomes map", func(t *testing.T) {
		got, err := structToFilteredMap(sample{Name: "x", Num: 3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["name"] != "x" || got["num"] != float64(3) || len(got) != 2 {
			t.Errorf("unexpected map: %v", got)
		}
	})

	t.Run("non object json fails to unmarshal into a map", func(t *testing.T) {
		got, err := structToFilteredMap("just a string")
		if err == nil {
			t.Fatalf("expected an error, got map %v", got)
		}
		if got != nil {
			t.Errorf("expected nil map on error, got %v", got)
		}
	})

	t.Run("unmarshalable value fails to marshal", func(t *testing.T) {
		got, err := structToFilteredMap(make(chan int))
		if err == nil {
			t.Fatalf("expected an error, got map %v", got)
		}
		if got != nil {
			t.Errorf("expected nil map on error, got %v", got)
		}
	})
}
