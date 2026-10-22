package server

import (
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_PostFlowBuilderChat_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/flow_builder/chat", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "POST", path: "/flow_builder/chat", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}
