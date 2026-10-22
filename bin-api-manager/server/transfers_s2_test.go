package server

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_TransfersHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "PostTransfers unauthenticated", method: http.MethodPost, path: "/transfers", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostTransfers bad json", method: http.MethodPost, path: "/transfers", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	}

	s2RunCases(t, tests)
}
