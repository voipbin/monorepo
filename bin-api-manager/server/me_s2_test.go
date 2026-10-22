package server

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_MeHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetMe unauthenticated", method: http.MethodGet, path: "/me", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
