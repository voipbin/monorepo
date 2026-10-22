package server

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_TranscriptsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetTranscripts unauthenticated", method: http.MethodGet, path: "/transcripts?transcribe_id=3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
