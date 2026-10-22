package server

import (
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_GetServiceAgentsWs_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/service_agents/ws", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	})
}
