package server

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

func Test_TimelinesSipHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetTimelinesCallsCallIdSipAnalysis unauthenticated", method: http.MethodGet, path: "/timelines/calls/3bc539bc-c68b-11ec-b41f-0776699e7467/sip-analysis", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetTimelinesCallsCallIdPcap unauthenticated", method: http.MethodGet, path: "/timelines/calls/3bc539bc-c68b-11ec-b41f-0776699e7467/pcap", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
