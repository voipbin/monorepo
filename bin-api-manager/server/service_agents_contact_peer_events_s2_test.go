package server

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_ServiceAgentsContactPeerEventsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetServiceAgentsContactPeerEvents unauthenticated", method: http.MethodGet, path: "/service_agents/contact_peer_events", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
