package server

import (
	"net/http"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"

	"go.uber.org/mock/gomock"
)

func Test_GetSpeakings_Matrix(t *testing.T) {
	s4RunCases(t, s4ListCases(s4ListEP{
		name: "GetSpeakings",
		path: "/speakings",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().SpeakingList(gomock.Any(), a, size, token)
		},
		mk: s4MkList[tmspeaking.WebhookMessage],
	}))
}

func Test_PostSpeakings_Matrix(t *testing.T) {
	validBody := `{"reference_type":"call","reference_id":"` + s4ID1 + `","language":"en-US","provider":"elevenlabs","voice_id":"voice-1","direction":"both"}`
	ret := s4Obj[tmspeaking.WebhookMessage](s4ID2, nil)

	cases := s4ItemCases(s4ItemEP{
		name: "PostSpeakings", method: http.MethodPost, path: "/speakings", body: validBody,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().SpeakingCreate(gomock.Any(), a, tmstreaming.ReferenceTypeCall, s4UUID(s4ID1), "en-US", "elevenlabs", "voice-1", tmstreaming.DirectionBoth)
		},
		ret: ret, okStatus: 201,
	})

	cases = append(cases,
		s4Case{
			name: "PostSpeakings/invalid_reference_type", method: http.MethodPost, path: "/speakings",
			body:          `{"reference_type":"bogus","reference_id":"` + s4ID1 + `"}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		s4Case{
			name: "PostSpeakings/invalid_reference_id", method: http.MethodPost, path: "/speakings",
			body:          `{"reference_type":"call","reference_id":"not-a-uuid"}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		s4Case{
			name: "PostSpeakings/invalid_direction", method: http.MethodPost, path: "/speakings",
			body:          `{"reference_type":"call","reference_id":"` + s4ID1 + `","direction":"sideways"}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		s4Case{
			name: "PostSpeakings/optional_fields_default_to_empty", method: http.MethodPost, path: "/speakings",
			body: `{"reference_type":"confbridge","reference_id":"` + s4ID1 + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().SpeakingCreate(gomock.Any(), a, tmstreaming.ReferenceTypeConfbridge, s4UUID(s4ID1), "", "", "", tmstreaming.DirectionNone).Return(ret, nil)
			},
			wantStatus: 201, wantBody: ret,
		},
	)
	s4RunCases(t, cases)
}

func Test_SpeakingsID_Matrix(t *testing.T) {
	ret := s4Obj[tmspeaking.WebhookMessage](s4ID1, nil)
	bad := "/speakings/not-a-uuid"

	var cases []s4Case
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteSpeakingsId", method: http.MethodDelete, path: "/speakings/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().SpeakingDelete(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetSpeakingsId", method: http.MethodGet, path: "/speakings/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().SpeakingGet(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostSpeakingsIdFlush", method: http.MethodPost, path: "/speakings/" + s4ID1 + "/flush",
		badIDPath: bad + "/flush", badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().SpeakingFlush(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostSpeakingsIdStop", method: http.MethodPost, path: "/speakings/" + s4ID1 + "/stop",
		badIDPath: bad + "/stop", badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().SpeakingStop(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostSpeakingsIdSay", method: http.MethodPost, path: "/speakings/" + s4ID1 + "/say",
		body:      `{"text":"hello there"}`,
		badIDPath: bad + "/say", badIDReason: "INVALID_ID",
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().SpeakingSay(gomock.Any(), a, s4UUID(s4ID1), "hello there")
		},
		ret: ret, okStatus: 200,
	})...)

	longText := make([]byte, 5001)
	for i := range longText {
		longText[i] = 'a'
	}
	maxText := string(longText[:5000])
	cases = append(cases,
		s4Case{
			name: "PostSpeakingsIdSay/empty_text", method: http.MethodPost, path: "/speakings/" + s4ID1 + "/say",
			body:          `{"text":""}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		s4Case{
			name: "PostSpeakingsIdSay/text_over_5000", method: http.MethodPost, path: "/speakings/" + s4ID1 + "/say",
			body:          `{"text":"` + string(longText) + `"}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		s4Case{
			name: "PostSpeakingsIdSay/text_exactly_5000_accepted", method: http.MethodPost, path: "/speakings/" + s4ID1 + "/say",
			body: `{"text":"` + maxText + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().SpeakingSay(gomock.Any(), a, s4UUID(s4ID1), maxText).Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
	)
	s4RunCases(t, cases)
}
