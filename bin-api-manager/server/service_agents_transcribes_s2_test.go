package server

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_ServiceAgentsTranscribesHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetServiceAgentsTranscribes unauthenticated", method: http.MethodGet, path: "/service_agents/transcribes", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsTranscribes unauthenticated", method: http.MethodPost, path: "/service_agents/transcribes", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsTranscribes bad json", method: http.MethodPost, path: "/service_agents/transcribes", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	}

	s2RunCases(t, tests)
}
