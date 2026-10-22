package server

import (
	"net/http"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsContactInteractions_Extra(t *testing.T) {
	contactID := s4UUID(s4ID1)
	addressID := s4UUID(s4ID2)
	base := "/service_agents/contact_interactions"
	items := []*tmpeerevent.PeerEvent{}

	listCase := func(name, query string, size uint64, token, peerType, peerTarget string, cID, aID uuid.UUID) s4Case {
		return s4Case{
			name: name, method: http.MethodGet, path: base + query,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentInteractionList(gomock.Any(), a, size, token, peerType, peerTarget, cID, aID).Return(items, "next-tok", nil)
			},
			wantStatus: 200, wantBody: map[string]any{"result": []any{}, "next_page_token": ""},
		}
	}

	cases := []s4Case{
		{
			name: "unauthenticated", method: http.MethodGet, path: base + "?contact_id=" + s4ID1, noAuth: true,
			wantErrStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED",
		},
		{
			name: "no_filter", method: http.MethodGet, path: base,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_FILTER",
		},
		{
			name: "two_filters_contact_and_address", method: http.MethodGet, path: base + "?contact_id=" + s4ID1 + "&address_id=" + s4ID2,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_FILTER",
		},
		{
			name: "two_filters_peer_and_contact", method: http.MethodGet, path: base + "?peer_type=tel&peer_target=%2B15551230001&contact_id=" + s4ID1,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_FILTER",
		},
		{
			name: "peer_type_alone_counts_as_peer_filter", method: http.MethodGet, path: base + "?peer_type=tel&contact_id=" + s4ID1,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_FILTER",
		},
		{
			name: "non_empty_result_passes_next_token", method: http.MethodGet, path: base + "?contact_id=" + s4ID1,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentInteractionList(gomock.Any(), a, uint64(100), "", "", "", contactID, uuid.Nil).Return(
					[]*tmpeerevent.PeerEvent{{EventType: "call_created", Direction: "incoming"}}, "next-tok", nil)
			},
			wantStatus: 200, wantBody: map[string]any{
				"result": []any{map[string]any{
					"timestamp": "0001-01-01T00:00:00Z", "customer_id": "00000000-0000-0000-0000-000000000000", "publisher": "",
					"event_type": "call_created", "reference_id": "00000000-0000-0000-0000-000000000000", "direction": "incoming",
					"peer": map[string]any{}, "local": map[string]any{}, "data": nil,
				}},
				"next_page_token": "next-tok",
			},
		},
		listCase("size_and_token_passthrough", "?contact_id="+s4ID1+"&page_size=7&page_token=tok-1", 7, "tok-1", "", "", contactID, uuid.Nil),
		listCase("size_zero_falls_back_to_100", "?contact_id="+s4ID1+"&page_size=0", 100, "", "", "", contactID, uuid.Nil),
		listCase("size_over_limit_clamped", "?address_id="+s4ID2+"&page_size=101", 100, "", "", "", uuid.Nil, addressID),
		listCase("peer_filter_passthrough", "?peer_type=tel&peer_target=%2B15551230001", 100, "", "tel", "+15551230001", uuid.Nil, uuid.Nil),
		{
			name: "service_error", method: http.MethodGet, path: base + "?contact_id=" + s4ID1,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentInteractionList(gomock.Any(), a, uint64(100), "", "", "", contactID, uuid.Nil).Return(nil, "", s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
	}
	s4RunCases(t, cases)
}

func Test_GetAggregatedEvents_Extra(t *testing.T) {
	events := []*servicehandler.TimelineEvent{{Timestamp: "2024-03-05T10:20:30Z", EventType: "call_created", Data: map[string]any{"k": "v"}}}
	want := map[string]any{
		"result":          []any{map[string]any{"timestamp": "2024-03-05T10:20:30Z", "event_type": "call_created", "data": map[string]any{"k": "v"}}},
		"next_page_token": "next-1",
	}
	mkCase := func(name, query string, size int, token string) s4Case {
		return s4Case{
			name: name, method: http.MethodGet, path: "/aggregated-events" + query,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AggregatedEventList(gomock.Any(), a, s4UUID(s4ID1), s4UUID(s4ID2), size, token).Return(events, "next-1", nil)
			},
			wantStatus: 200, wantBody: want,
		}
	}
	ids := "?activeflow_id=" + s4ID1 + "&call_id=" + s4ID2
	s4RunCases(t, []s4Case{
		mkCase("page_size_over_1000_clamped_to_100", ids+"&page_size=1001", 100, ""),
		mkCase("page_size_exactly_1000_kept", ids+"&page_size=1000", 1000, ""),
		mkCase("page_size_zero_clamped_to_100", ids+"&page_size=0", 100, ""),
		mkCase("page_token_passthrough", ids+"&page_token=tok-9&page_size=5", 5, "tok-9"),
		mkCase("defaults", ids, 100, ""),
	})
}

func Test_GetTimelinesEvents_Extra(t *testing.T) {
	events := []*servicehandler.TimelineEvent{{Timestamp: "2024-03-05T10:20:30Z", EventType: "flow_started", Data: "x"}}
	want := map[string]any{
		"result":          []any{map[string]any{"timestamp": "2024-03-05T10:20:30Z", "event_type": "flow_started", "data": "x"}},
		"next_page_token": "next-2",
	}
	mkCase := func(name, query string, size int, token string) s4Case {
		return s4Case{
			name: name, method: http.MethodGet, path: "/timelines/flows/" + s4ID1 + "/events" + query,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().TimelineEventList(gomock.Any(), a, "flows", s4UUID(s4ID1), size, token).Return(events, "next-2", nil)
			},
			wantStatus: 200, wantBody: want,
		}
	}
	s4RunCases(t, []s4Case{
		mkCase("page_size_over_1000_clamped_to_100", "?page_size=1001", 100, ""),
		mkCase("page_size_exactly_1000_kept", "?page_size=1000", 1000, ""),
		mkCase("page_size_negative_clamped_to_100", "?page_size=-3", 100, ""),
		mkCase("page_token_passthrough", "?page_size=5&page_token=tok-7", 5, "tok-7"),
	})
}
