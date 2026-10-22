package server

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_ContactCaseMessagesHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "PostContactCasesIdMessages unauthenticated", method: http.MethodPost, path: "/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467/messages", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostContactCasesIdMessages bad json", method: http.MethodPost, path: "/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467/messages", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	}

	s2RunCases(t, tests)
}
