package summaryhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
)

var (
	g4CustomerID = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000001")
	g4RefID      = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000002")
	g4SummaryID  = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000003")
	g4FlowID     = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000004")
	g4OnEndID    = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000005")
	g4ConfbrID   = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000006")
	g4TrID       = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000007")
	g4ActiveID   = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000008")
)

type g4Mocks struct {
	h      *summaryHandler
	db     *dbhandler.MockDBHandler
	util   *utilhandler.MockUtilHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler
	openai *engine_openai_handler.MockEngineOpenaiHandler
}

func g4New(mc *gomock.Controller) *g4Mocks {
	m := &g4Mocks{
		db:     dbhandler.NewMockDBHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		openai: engine_openai_handler.NewMockEngineOpenaiHandler(mc),
	}
	m.h = &summaryHandler{
		utilHandler:         m.util,
		db:                  m.db,
		reqHandler:          m.req,
		notifyHandler:       m.notify,
		engineOpenaiHandler: m.openai,
		model:               "test-model",
		reasoningEffort:     "none",
	}
	return m
}

func g4Summary(refType summary.ReferenceType) *summary.Summary {
	return &summary.Summary{
		Identity:      commonidentity.Identity{ID: g4SummaryID, CustomerID: g4CustomerID},
		ReferenceType: refType,
		ReferenceID:   g4RefID,
		Language:      "en-US",
		Status:        summary.StatusProgressing,
	}
}

func g4Resp(content string) *openai.ChatCompletionResponse {
	return &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: content}}},
	}
}

// ---- event.go ----

func Test_EventCMCallHangup_NoSummary(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", map[summary.Field]any{
		summary.FieldDeleted:     false,
		summary.FieldReferenceID: g4RefID,
	}).Return([]*summary.Summary{}, nil)

	m.h.EventCMCallHangup(context.Background(), &cmcall.Call{Identity: commonidentity.Identity{ID: g4RefID}})
}

func Test_EventCMCallHangup_ListError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, errors.New("db down"))

	m.h.EventCMCallHangup(context.Background(), &cmcall.Call{Identity: commonidentity.Identity{ID: g4RefID}})
}

func Test_EventCMCallHangup_ProcessesCallSummary(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	// The lookup runs once in the event handler and once in the call content process.
	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).
		Return([]*summary.Summary{g4Summary(summary.ReferenceTypeCall)}, nil).Times(2)
	m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), map[tmtranscribe.Field]any{
		tmtranscribe.FieldDeleted:     false,
		tmtranscribe.FieldCustomerID:  cmcustomer.IDAIManager.String(),
		tmtranscribe.FieldReferenceID: g4RefID.String(),
	}).Return(nil, errors.New("rpc failed"))

	m.h.EventCMCallHangup(context.Background(), &cmcall.Call{Identity: commonidentity.Identity{ID: g4RefID}})
}

func Test_EventCMConferenceUpdated_ListError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, errors.New("db down"))

	m.h.EventCMConferenceUpdated(context.Background(), &cfconference.Conference{Identity: commonidentity.Identity{ID: g4RefID}})
}

// ---- content.go ----

func Test_ContentProcess_UnsupportedReferenceType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	// No mock expectation: an unsupported type must not reach any backend.
	m.h.ContentProcess(context.Background(), g4Summary(summary.ReferenceTypeRecording))
}

func Test_ContentProcess_ConferenceDispatch(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).
		Return([]*summary.Summary{g4Summary(summary.ReferenceTypeConference)}, nil)
	m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(nil, errors.New("rpc failed"))

	m.h.ContentProcess(context.Background(), g4Summary(summary.ReferenceTypeConference))
}

// g4Pipeline drives contentProcessReferenceType{Call,Conference} through each failure stage.
func Test_contentProcessReferenceTypeCall_stages(t *testing.T) {
	rpcErr := errors.New("backend failed")
	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantErr string
	}{
		{
			name: "summary lookup fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not get the summary",
		},
		{
			name: "no transcribe found",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*summary.Summary{g4Summary(summary.ReferenceTypeCall)}, nil)
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]tmtranscribe.Transcribe{}, nil)
			},
			wantErr: "could not get the transcripts",
		},
		{
			name: "transcript list fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*summary.Summary{g4Summary(summary.ReferenceTypeCall)}, nil)
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).
					Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID}}}, nil)
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), map[tmtranscript.Field]any{
					tmtranscript.FieldDeleted:      false,
					tmtranscript.FieldTranscribeID: g4TrID.String(),
				}).Return(nil, rpcErr)
			},
			wantErr: "could not get the transcripts",
		},
		{
			name: "content generation fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*summary.Summary{g4Summary(summary.ReferenceTypeCall)}, nil)
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).
					Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID}}}, nil)
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not send the request",
		},
		{
			name: "status update fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*summary.Summary{g4Summary(summary.ReferenceTypeCall)}, nil)
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).
					Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID}}}, nil)
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("summary text"), nil)
				m.db.EXPECT().SummaryUpdateStatusDoneIfNotDone(gomock.Any(), g4SummaryID, "summary text").Return(int64(0), rpcErr)
			},
			wantErr: "could not update the status",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			err := m.h.contentProcessReferenceTypeCall(context.Background(), g4RefID)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func Test_contentProcessReferenceTypeCall_onEndFlowFailureIsNotFatal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	sm := g4Summary(summary.ReferenceTypeCall)
	sm.OnEndFlowID = g4OnEndID
	done := g4Summary(summary.ReferenceTypeCall)
	done.OnEndFlowID = g4OnEndID
	done.Status = summary.StatusDone
	done.Content = "summary text"

	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*summary.Summary{sm}, nil)
	m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).
		Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID}}}, nil)
	m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
	m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("summary text"), nil)
	m.db.EXPECT().SummaryUpdateStatusDoneIfNotDone(gomock.Any(), g4SummaryID, "summary text").Return(int64(1), nil)
	m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(done, nil)
	m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), g4CustomerID, summary.EventTypeUpdated, done)
	m.req.EXPECT().FlowV1ActiveflowCreate(gomock.Any(), uuid.Nil, g4CustomerID, g4OnEndID, fmactiveflow.ReferenceTypeAI, g4SummaryID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).
		Return(nil, errors.New("flow down"))

	if err := m.h.contentProcessReferenceTypeCall(context.Background(), g4RefID); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func Test_contentProcessReferenceTypeConference_stages(t *testing.T) {
	rpcErr := errors.New("backend failed")
	sm := func() []*summary.Summary { return []*summary.Summary{g4Summary(summary.ReferenceTypeConference)} }
	cf := &cfconference.Conference{Identity: commonidentity.Identity{ID: g4RefID}, ConfbridgeID: g4ConfbrID}

	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantErr string
	}{
		{
			name: "summary lookup fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not get the summary",
		},
		{
			name: "conference get fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(sm(), nil)
				m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(nil, rpcErr)
			},
			wantErr: "could not get the conference data",
		},
		{
			name: "transcripts lookup uses the confbridge id and fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(sm(), nil)
				m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(cf, nil)
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), map[tmtranscribe.Field]any{
					tmtranscribe.FieldDeleted:     false,
					tmtranscribe.FieldCustomerID:  cmcustomer.IDAIManager.String(),
					tmtranscribe.FieldReferenceID: g4ConfbrID.String(),
				}).Return(nil, rpcErr)
			},
			wantErr: "could not get the transcripts",
		},
		{
			name: "content generation fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(sm(), nil)
				m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(cf, nil)
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).
					Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID}}}, nil)
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not send the request",
		},
		{
			name: "status update fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(sm(), nil)
				m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(cf, nil)
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).
					Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID}}}, nil)
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("conf summary"), nil)
				m.db.EXPECT().SummaryUpdateStatusDoneIfNotDone(gomock.Any(), g4SummaryID, "conf summary").Return(int64(0), rpcErr)
			},
			wantErr: "could not update the status",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			err := m.h.contentProcessReferenceTypeConference(context.Background(), g4RefID)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func Test_contentProcessReferenceTypeConference_onEndFlowFailureIsNotFatal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	sm := g4Summary(summary.ReferenceTypeConference)
	sm.OnEndFlowID = g4OnEndID
	done := g4Summary(summary.ReferenceTypeConference)
	done.OnEndFlowID = g4OnEndID
	done.Status = summary.StatusDone

	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*summary.Summary{sm}, nil)
	m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(&cfconference.Conference{ConfbridgeID: g4ConfbrID}, nil)
	m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).
		Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID}}}, nil)
	m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
	m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("conf summary"), nil)
	m.db.EXPECT().SummaryUpdateStatusDoneIfNotDone(gomock.Any(), g4SummaryID, "conf summary").Return(int64(1), nil)
	m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(done, nil)
	m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), g4CustomerID, summary.EventTypeUpdated, done)
	m.req.EXPECT().FlowV1ActiveflowCreate(gomock.Any(), uuid.Nil, g4CustomerID, g4OnEndID, fmactiveflow.ReferenceTypeAI, g4SummaryID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).
		Return(nil, errors.New("flow down"))

	if err := m.h.contentProcessReferenceTypeConference(context.Background(), g4RefID); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func Test_contentGetTranscripts_transcriptListError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	rpcErr := errors.New("backend failed")
	m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).
		Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID}}}, nil)
	m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(nil, rpcErr)

	res, err := m.h.contentGetTranscripts(context.Background(), g4RefID)
	if res != nil || !errors.Is(err, rpcErr) {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_contentGetTranscripts_noTranscribe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]tmtranscribe.Transcribe{}, nil)

	res, err := m.h.contentGetTranscripts(context.Background(), g4RefID)
	if res != nil || err == nil || !strings.Contains(err.Error(), "could not find the transcribe data") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_contentGet_variableGetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	rpcErr := errors.New("flow down")
	m.req.EXPECT().FlowV1VariableGet(gomock.Any(), g4FlowID).Return(nil, rpcErr)

	res, err := m.h.contentGet(context.Background(), g4FlowID, summary.ReferenceTypeCall, nil, "en-US")
	if res != "" || !errors.Is(err, rpcErr) {
		t.Errorf("unexpected result: %q, err: %v", res, err)
	}
}

func Test_contentGet_variablesAreSentToTheModel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.req.EXPECT().FlowV1VariableGet(gomock.Any(), g4FlowID).Return(&fmvariable.Variable{
		ID:        g4FlowID,
		Variables: map[string]string{"k": "v-value"},
	}, nil)
	m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
			if len(req.Messages) != 2 || !strings.Contains(req.Messages[1].Content, `"k":"v-value"`) {
				t.Errorf("variables missing from the request: %+v", req.Messages)
			}
			return g4Resp("ok"), nil
		})

	res, err := m.h.contentGet(context.Background(), g4FlowID, summary.ReferenceTypeCall, nil, "en-US")
	if err != nil || res != "ok" {
		t.Errorf("unexpected result: %q, err: %v", res, err)
	}
}

func Test_contentGet_emptyChoices(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(&openai.ChatCompletionResponse{}, nil)

	res, err := m.h.contentGet(context.Background(), uuid.Nil, summary.ReferenceTypeCall, nil, "en-US")
	if err != nil || res != "" {
		t.Errorf("unexpected result: %q, err: %v", res, err)
	}
}

func Test_verifyOutputLanguage_branches(t *testing.T) {
	prose := "- The customer asked for a refund and the agent explained the whole process."
	tests := []struct {
		name    string
		lang    string
		content string
		setup   func(m *g4Mocks)
		want    bool
	}{
		{name: "english target skips the detector", lang: "en-US", content: prose, setup: func(*g4Mocks) {}, want: true},
		{name: "too little prose skips the detector", lang: "ko-KR", content: "Call Type:\n- None", setup: func(*g4Mocks) {}, want: true},
		{
			name: "detector error is fail-open", lang: "ko-KR", content: prose,
			setup: func(m *g4Mocks) {
				m.openai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).Return(nil, errors.New("timeout"))
			},
			want: true,
		},
		{
			name: "empty detector choices is fail-open", lang: "ko-KR", content: prose,
			setup: func(m *g4Mocks) {
				m.openai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).Return(&openai.ChatCompletionResponse{}, nil)
			},
			want: true,
		},
		{
			name: "yes answer", lang: "ko-KR", content: prose,
			setup: func(m *g4Mocks) {
				m.openai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).Return(g4Resp("  Yes."), nil)
			},
			want: true,
		},
		{
			name: "no answer", lang: "ko-KR", content: prose,
			setup: func(m *g4Mocks) {
				m.openai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).Return(g4Resp("NO, it is English"), nil)
			},
			want: false,
		},
		{
			name: "unexpected answer is fail-open", lang: "ko-KR", content: prose,
			setup: func(m *g4Mocks) {
				m.openai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).Return(g4Resp("maybe"), nil)
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			if got := m.h.verifyOutputLanguage(context.Background(), tt.content, tt.lang); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_verifyOutputLanguage_sampleIsTruncatedOnRuneBoundary(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	long := "- " + strings.Repeat("가", languageVerifySampleLen+500)
	m.openai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
			body := req.Messages[0].Content
			if got := strings.Count(body, "가"); got != languageVerifySampleLen-2 {
				t.Errorf("sample rune count: got %d, want %d", got, languageVerifySampleLen-2)
			}
			return g4Resp("yes"), nil
		})

	if !m.h.verifyOutputLanguage(context.Background(), long, "ko-KR") {
		t.Errorf("expected pass")
	}
}

// ---- db.go ----

func Test_Create_errors(t *testing.T) {
	rpcErr := errors.New("backend failed")
	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantErr string
	}{
		{
			name: "insert fails",
			setup: func(m *g4Mocks) {
				m.util.EXPECT().UUIDCreate().Return(g4SummaryID)
				m.db.EXPECT().SummaryCreate(gomock.Any(), gomock.Any()).Return(rpcErr)
			},
			wantErr: "could not create summary",
		},
		{
			name: "reload fails",
			setup: func(m *g4Mocks) {
				m.util.EXPECT().UUIDCreate().Return(g4SummaryID)
				m.db.EXPECT().SummaryCreate(gomock.Any(), gomock.Any()).Return(nil)
				m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(nil, rpcErr)
			},
			wantErr: "could not get created data",
		},
		{
			name: "variable set fails",
			setup: func(m *g4Mocks) {
				m.util.EXPECT().UUIDCreate().Return(g4SummaryID)
				m.db.EXPECT().SummaryCreate(gomock.Any(), gomock.Any()).Return(nil)
				sm := g4Summary(summary.ReferenceTypeCall)
				sm.ActiveflowID = g4FlowID
				m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(sm, nil)
				m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), g4FlowID, gomock.Any()).Return(rpcErr)
			},
			wantErr: "could not set the variable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			res, err := m.h.Create(context.Background(), g4CustomerID, g4FlowID, uuid.Nil, summary.ReferenceTypeCall, g4RefID, summary.StatusProgressing, "en-US", "")
			if res != nil || !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

func Test_Get_otherError_isWrapped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	rpcErr := errors.New("db down")
	m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(nil, rpcErr)

	res, err := m.h.Get(context.Background(), g4SummaryID)
	if res != nil || !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), "could not get data") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_GetByReferenceID_listError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	rpcErr := errors.New("db down")
	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, rpcErr)

	res, err := m.h.GetByReferenceID(context.Background(), g4RefID)
	if res != nil || !errors.Is(err, rpcErr) {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_List_listError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	rpcErr := errors.New("db down")
	m.db.EXPECT().SummaryList(gomock.Any(), uint64(5), "tok", gomock.Any()).Return(nil, rpcErr)

	res, err := m.h.List(context.Background(), 5, "tok", nil)
	if res != nil || !errors.Is(err, rpcErr) {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_GetByCustomerIDAndReferenceIDAndLanguage_listError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	rpcErr := errors.New("db down")
	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1000), "", map[summary.Field]any{
		summary.FieldDeleted:     false,
		summary.FieldCustomerID:  g4CustomerID,
		summary.FieldReferenceID: g4RefID,
		summary.FieldLanguage:    "ko-KR",
	}).Return(nil, rpcErr)

	res, err := m.h.GetByCustomerIDAndReferenceIDAndLanguage(context.Background(), g4CustomerID, g4RefID, "ko-KR")
	if res != nil || !errors.Is(err, rpcErr) {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Delete_errors(t *testing.T) {
	rpcErr := errors.New("db down")

	t.Run("delete fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().SummaryDelete(gomock.Any(), g4SummaryID).Return(rpcErr)

		res, err := m.h.Delete(context.Background(), g4SummaryID)
		if res != nil || !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), "could not delete the summary") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("reload fails and no webhook is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().SummaryDelete(gomock.Any(), g4SummaryID).Return(nil)
		m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(nil, rpcErr)

		res, err := m.h.Delete(context.Background(), g4SummaryID)
		if res != nil || !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), "could not get updated summary") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

func Test_UpdateContentLanguage_errors(t *testing.T) {
	rpcErr := errors.New("db down")

	t.Run("update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().SummaryUpdate(gomock.Any(), g4SummaryID, map[summary.Field]any{
			summary.FieldContent:  "c",
			summary.FieldLanguage: "ko-KR",
			summary.FieldStatus:   summary.StatusDone,
		}).Return(rpcErr)

		res, err := m.h.UpdateContentLanguage(context.Background(), g4SummaryID, "c", "ko-KR")
		if res != nil || !errors.Is(err, rpcErr) {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("reload fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().SummaryUpdate(gomock.Any(), g4SummaryID, gomock.Any()).Return(nil)
		m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(nil, rpcErr)

		res, err := m.h.UpdateContentLanguage(context.Background(), g4SummaryID, "c", "ko-KR")
		if res != nil || !errors.Is(err, rpcErr) {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

func Test_UpdateStatusDone_errors(t *testing.T) {
	rpcErr := errors.New("db down")

	t.Run("conditional update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().SummaryUpdateStatusDoneIfNotDone(gomock.Any(), g4SummaryID, "c").Return(int64(0), rpcErr)

		res, err := m.h.UpdateStatusDone(context.Background(), g4SummaryID, "c")
		if res != nil || !errors.Is(err, rpcErr) || errors.Is(err, ErrSummaryAlreadyDone) {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("reload fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().SummaryUpdateStatusDoneIfNotDone(gomock.Any(), g4SummaryID, "c").Return(int64(1), nil)
		m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(nil, rpcErr)

		res, err := m.h.UpdateStatusDone(context.Background(), g4SummaryID, "c")
		if res != nil || !errors.Is(err, rpcErr) {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

// ---- variable.go / service.go / regenerate.go ----

func Test_variableSet_requestError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	rpcErr := errors.New("flow down")
	m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), g4FlowID, gomock.Any()).Return(rpcErr)

	err := m.h.variableSet(context.Background(), g4FlowID, g4Summary(summary.ReferenceTypeCall))
	if !errors.Is(err, rpcErr) {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func Test_variableSet_nilActiveflowSkipsRequest(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	if err := m.h.variableSet(context.Background(), uuid.Nil, g4Summary(summary.ReferenceTypeCall)); err != nil {
		t.Errorf("unexpected err: %v", err)
	}
}

func Test_ServiceStart_startError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1000), "", gomock.Any()).Return(nil, errors.New("db down"))
	m.req.EXPECT().CallV1CallGet(gomock.Any(), g4RefID).Return(nil, errors.New("rpc failed"))

	res, err := m.h.ServiceStart(context.Background(), g4CustomerID, g4FlowID, uuid.Nil, summary.ReferenceTypeCall, g4RefID, "en-US")
	if res != nil || err == nil || !strings.Contains(err.Error(), "Could not start the summary") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Regenerate_errors(t *testing.T) {
	rpcErr := errors.New("backend failed")
	existing := func() *summary.Summary {
		sm := g4Summary(summary.ReferenceTypeRecording)
		sm.Status = summary.StatusDone
		sm.Content = "old"
		return sm
	}
	expectReuse := func(m *g4Mocks) {
		m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]tmtranscribe.Transcribe{
			{Identity: commonidentity.Identity{ID: g4TrID, CustomerID: g4CustomerID}},
		}, nil)
		m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{
			{Identity: commonidentity.Identity{ID: g4ActiveID}},
		}, nil)
	}

	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantErr string
	}{
		{
			name: "transcript source fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(existing(), nil)
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not get the transcripts",
		},
		{
			name: "persisting the new content fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(existing(), nil)
				expectReuse(m)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("new content"), nil)
				m.db.EXPECT().SummaryUpdate(gomock.Any(), g4SummaryID, gomock.Any()).Return(rpcErr)
			},
			wantErr: "could not update the summary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			res, err := m.h.Regenerate(context.Background(), g4SummaryID, "en-US")
			if res != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

// ---- start.go ----

func Test_Start_existingSummaryIsReturned(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	existing := g4Summary(summary.ReferenceTypeCall)
	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1000), "", map[summary.Field]any{
		summary.FieldDeleted:     false,
		summary.FieldCustomerID:  g4CustomerID,
		summary.FieldReferenceID: g4RefID,
		summary.FieldLanguage:    "en-US",
	}).Return([]*summary.Summary{existing}, nil)

	res, err := m.h.Start(context.Background(), g4CustomerID, uuid.Nil, uuid.Nil, summary.ReferenceTypeCall, g4RefID, "")
	if err != nil || res != existing {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Start_unsupportedReferenceType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().SummaryList(gomock.Any(), uint64(1000), "", gomock.Any()).Return([]*summary.Summary{}, nil)

	res, err := m.h.Start(context.Background(), g4CustomerID, uuid.Nil, uuid.Nil, summary.ReferenceType("bogus"), g4RefID, "en-US")
	if res != nil || err == nil || !strings.Contains(err.Error(), "unsupported reference type: bogus") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Start_dispatchesByReferenceType(t *testing.T) {
	rpcErr := errors.New("backend failed")
	tests := []struct {
		name    string
		refType summary.ReferenceType
		expect  func(m *g4Mocks)
		wantErr string
	}{
		{
			name:    "call",
			refType: summary.ReferenceTypeCall,
			expect: func(m *g4Mocks) {
				m.req.EXPECT().CallV1CallGet(gomock.Any(), g4RefID).Return(nil, rpcErr)
			},
			wantErr: "could not get the call data",
		},
		{
			name:    "conference",
			refType: summary.ReferenceTypeConference,
			expect: func(m *g4Mocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(nil, rpcErr)
			},
			wantErr: "could not get the conference data",
		},
		{
			name:    "transcribe",
			refType: summary.ReferenceTypeTranscribe,
			expect: func(m *g4Mocks) {
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not get the transcribe data",
		},
		{
			name:    "recording",
			refType: summary.ReferenceTypeRecording,
			expect: func(m *g4Mocks) {
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not get the transcripts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)

			m.db.EXPECT().SummaryList(gomock.Any(), uint64(1000), "", gomock.Any()).Return(nil, rpcErr)
			tt.expect(m)

			res, err := m.h.Start(context.Background(), g4CustomerID, uuid.Nil, uuid.Nil, tt.refType, g4RefID, "en-US")
			if res != nil || !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

func Test_startReferenceTypeCall_errors(t *testing.T) {
	rpcErr := errors.New("backend failed")
	callOK := &cmcall.Call{Identity: commonidentity.Identity{ID: g4RefID}, ActiveflowID: g4ActiveID}
	expectStart := func(m *g4Mocks, activeflowID uuid.UUID, ret error) {
		var tr *tmtranscribe.Transcribe
		if ret == nil {
			tr = &tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: g4TrID}}
		}
		m.req.EXPECT().TranscribeV1TranscribeStart(gomock.Any(), uuid.Nil, cmcustomer.IDAIManager, activeflowID, uuid.Nil,
			tmtranscribe.ReferenceTypeCall, g4RefID, "en-US", tmtranscribe.DirectionBoth, tmtranscribe.ProviderEmpty, 5000).Return(tr, ret)
	}

	tests := []struct {
		name         string
		activeflowID uuid.UUID
		setup        func(m *g4Mocks)
		wantErr      string
	}{
		{
			name:  "call get fails",
			setup: func(m *g4Mocks) { m.req.EXPECT().CallV1CallGet(gomock.Any(), g4RefID).Return(nil, rpcErr) },

			wantErr: "could not get the call data",
		},
		{
			name: "call already hung up",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().CallV1CallGet(gomock.Any(), g4RefID).Return(&cmcall.Call{Status: cmcall.StatusHangup}, nil)
			},
			wantErr: "already been hung up",
		},
		{
			name:         "transcribe start fails using the explicit activeflow",
			activeflowID: g4FlowID,
			setup: func(m *g4Mocks) {
				m.req.EXPECT().CallV1CallGet(gomock.Any(), g4RefID).Return(callOK, nil)
				expectStart(m, g4FlowID, rpcErr)
			},
			wantErr: "could not start the transcribe",
		},
		{
			name: "nil activeflow falls back to the call's activeflow, then create fails",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().CallV1CallGet(gomock.Any(), g4RefID).Return(callOK, nil)
				expectStart(m, g4ActiveID, nil)
				m.util.EXPECT().UUIDCreate().Return(g4SummaryID)
				m.db.EXPECT().SummaryCreate(gomock.Any(), gomock.Any()).Return(rpcErr)
			},
			wantErr: "could not create the summary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			res, err := m.h.startReferenceTypeCall(context.Background(), g4CustomerID, tt.activeflowID, uuid.Nil, g4RefID, "en-US")
			if res != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

func Test_startReferenceTypeConference_errors(t *testing.T) {
	rpcErr := errors.New("backend failed")
	cfOK := &cfconference.Conference{Identity: commonidentity.Identity{ID: g4RefID}, Status: cfconference.StatusProgressing, ConfbridgeID: g4ConfbrID}
	expectStart := func(m *g4Mocks, ret error) {
		var tr *tmtranscribe.Transcribe
		if ret == nil {
			tr = &tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: g4TrID}}
		}
		m.req.EXPECT().TranscribeV1TranscribeStart(gomock.Any(), uuid.Nil, cmcustomer.IDAIManager, g4FlowID, uuid.Nil,
			tmtranscribe.ReferenceTypeConfbridge, g4ConfbrID, "en-US", tmtranscribe.DirectionIn, tmtranscribe.ProviderEmpty, 5000).Return(tr, ret)
	}

	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantErr string
	}{
		{
			name:    "conference get fails",
			setup:   func(m *g4Mocks) { m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(nil, rpcErr) },
			wantErr: "could not get the conference data",
		},
		{
			name: "conference not progressing",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).
					Return(&cfconference.Conference{Status: cfconference.StatusTerminated}, nil)
			},
			wantErr: "not progressing",
		},
		{
			name: "transcribe start fails",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(cfOK, nil)
				expectStart(m, rpcErr)
			},
			wantErr: "could not start the transcribe",
		},
		{
			name: "create fails",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(gomock.Any(), g4RefID).Return(cfOK, nil)
				expectStart(m, nil)
				m.util.EXPECT().UUIDCreate().Return(g4SummaryID)
				m.db.EXPECT().SummaryCreate(gomock.Any(), gomock.Any()).Return(rpcErr)
			},
			wantErr: "could not create the summary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			res, err := m.h.startReferenceTypeConference(context.Background(), g4CustomerID, g4FlowID, uuid.Nil, g4RefID, "en-US")
			if res != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

// g4ExpectCreateDone registers the Create path for a done summary without an activeflow.
func g4ExpectCreateDone(m *g4Mocks, refType summary.ReferenceType, onEnd uuid.UUID, content string) *summary.Summary {
	created := g4Summary(refType)
	created.Status = summary.StatusDone
	created.Content = content
	created.OnEndFlowID = onEnd
	m.util.EXPECT().UUIDCreate().Return(g4SummaryID)
	m.db.EXPECT().SummaryCreate(gomock.Any(), gomock.Any()).Return(nil)
	m.db.EXPECT().SummaryGet(gomock.Any(), g4SummaryID).Return(created, nil)
	m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), g4CustomerID, summary.EventTypeCreated, created)
	return created
}

func Test_startReferenceTypeTranscribe_errors(t *testing.T) {
	rpcErr := errors.New("backend failed")
	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantErr string
	}{
		{
			name: "content generation fails",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not send the request",
		},
		{
			name: "create fails",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("text"), nil)
				m.util.EXPECT().UUIDCreate().Return(g4SummaryID)
				m.db.EXPECT().SummaryCreate(gomock.Any(), gomock.Any()).Return(rpcErr)
			},
			wantErr: "could not create the summary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			res, err := m.h.startReferenceTypeTranscribe(context.Background(), g4CustomerID, uuid.Nil, uuid.Nil, g4RefID, "en-US")
			if res != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

func Test_startReferenceTypeTranscribe_onEndFlowFailureIsNotFatal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)
	m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("text"), nil)
	created := g4ExpectCreateDone(m, summary.ReferenceTypeTranscribe, g4OnEndID, "text")
	m.req.EXPECT().FlowV1ActiveflowCreate(gomock.Any(), uuid.Nil, g4CustomerID, g4OnEndID, fmactiveflow.ReferenceTypeAI, g4SummaryID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).
		Return(nil, errors.New("flow down"))

	res, err := m.h.startReferenceTypeTranscribe(context.Background(), g4CustomerID, uuid.Nil, g4OnEndID, g4RefID, "en-US")
	if err != nil || res != created {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_startReferenceTypeRecording_errors(t *testing.T) {
	rpcErr := errors.New("backend failed")
	reuseTranscribe := []tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID, CustomerID: g4CustomerID}}}
	oneTranscript := []tmtranscript.Transcript{{Identity: commonidentity.Identity{ID: g4ActiveID}}}

	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantErr string
	}{
		{
			name: "content generation fails",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), gomock.Any()).Return(reuseTranscribe, nil)
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(oneTranscript, nil)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not send the request",
		},
		{
			name: "create fails",
			setup: func(m *g4Mocks) {
				m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), gomock.Any()).Return(reuseTranscribe, nil)
				m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(oneTranscript, nil)
				m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("text"), nil)
				m.util.EXPECT().UUIDCreate().Return(g4SummaryID)
				m.db.EXPECT().SummaryCreate(gomock.Any(), gomock.Any()).Return(rpcErr)
			},
			wantErr: "could not create the summary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			res, err := m.h.startReferenceTypeRecording(context.Background(), g4CustomerID, uuid.Nil, uuid.Nil, g4RefID, "en-US")
			if res != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

func Test_startReferenceTypeRecording_onEndFlowFailureIsNotFatal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), gomock.Any()).
		Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: g4TrID, CustomerID: g4CustomerID}}}, nil)
	m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), gomock.Any()).
		Return([]tmtranscript.Transcript{{Identity: commonidentity.Identity{ID: g4ActiveID}}}, nil)
	m.openai.EXPECT().Send(gomock.Any(), gomock.Any()).Return(g4Resp("text"), nil)
	created := g4ExpectCreateDone(m, summary.ReferenceTypeRecording, g4OnEndID, "text")
	m.req.EXPECT().FlowV1ActiveflowCreate(gomock.Any(), uuid.Nil, g4CustomerID, g4OnEndID, fmactiveflow.ReferenceTypeAI, g4SummaryID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).
		Return(nil, errors.New("flow down"))

	res, err := m.h.startReferenceTypeRecording(context.Background(), g4CustomerID, uuid.Nil, g4OnEndID, g4RefID, "en-US")
	if err != nil || res != created {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_getRecordingTranscripts_paths(t *testing.T) {
	rpcErr := errors.New("backend failed")
	userTr := uuid.FromStringOrNil("10000000-0000-4000-8000-0000000000a1")
	aiTr := uuid.FromStringOrNil("10000000-0000-4000-8000-0000000000a2")
	newTr := uuid.FromStringOrNil("10000000-0000-4000-8000-0000000000a3")
	oneTranscript := []tmtranscript.Transcript{{Identity: commonidentity.Identity{ID: g4ActiveID}}}

	listCandidates := func(m *g4Mocks, cands []tmtranscribe.Transcribe) {
		m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), gomock.Any()).Return(cands, nil)
	}
	transcriptsFor := func(m *g4Mocks, id uuid.UUID, ts []tmtranscript.Transcript, err error) {
		m.req.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(1000), map[tmtranscript.Field]any{
			tmtranscript.FieldDeleted:      false,
			tmtranscript.FieldTranscribeID: id.String(),
		}).Return(ts, err)
	}
	startNew := func(m *g4Mocks, ret error) {
		var tr *tmtranscribe.Transcribe
		if ret == nil {
			tr = &tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: newTr}}
		}
		m.req.EXPECT().TranscribeV1TranscribeStart(gomock.Any(), uuid.Nil, cmcustomer.IDAIManager, g4FlowID, uuid.Nil,
			tmtranscribe.ReferenceTypeRecording, g4RefID, defaultSTTFallbackLanguage, tmtranscribe.DirectionBoth, tmtranscribe.ProviderEmpty, 300000).Return(tr, ret)
	}

	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantLen int
		wantErr string
	}{
		{
			name: "candidate transcript lookup fails",
			setup: func(m *g4Mocks) {
				listCandidates(m, []tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: userTr, CustomerID: g4CustomerID}}})
				transcriptsFor(m, userTr, nil, rpcErr)
			},
			wantErr: "for the candidate transcribe",
		},
		{
			name: "summary-created candidates are skipped, then a new transcribe is started",
			setup: func(m *g4Mocks) {
				listCandidates(m, []tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: aiTr, CustomerID: cmcustomer.IDAIManager}}})
				startNew(m, nil)
				transcriptsFor(m, newTr, oneTranscript, nil)
			},
			wantLen: 1,
		},
		{
			name: "candidate without transcripts falls through to a new transcribe",
			setup: func(m *g4Mocks) {
				listCandidates(m, []tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: userTr, CustomerID: g4CustomerID}}})
				transcriptsFor(m, userTr, []tmtranscript.Transcript{}, nil)
				startNew(m, nil)
				transcriptsFor(m, newTr, oneTranscript, nil)
			},
			wantLen: 1,
		},
		{
			name: "new transcribe start fails",
			setup: func(m *g4Mocks) {
				listCandidates(m, nil)
				startNew(m, rpcErr)
			},
			wantErr: "could not start the transcribe",
		},
		{
			name: "new transcript lookup fails",
			setup: func(m *g4Mocks) {
				listCandidates(m, nil)
				startNew(m, nil)
				transcriptsFor(m, newTr, nil, rpcErr)
			},
			wantErr: "could not get the transcribe data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			res, err := m.h.getRecordingTranscripts(context.Background(), g4FlowID, g4RefID)
			if tt.wantErr != "" {
				if res != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("unexpected result: %v, err: %v", res, err)
				}
				return
			}
			if err != nil || len(res) != tt.wantLen {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

func Test_startOnEndFlow_errors(t *testing.T) {
	rpcErr := errors.New("flow down")
	sm := func() *summary.Summary {
		s := g4Summary(summary.ReferenceTypeCall)
		s.OnEndFlowID = g4OnEndID
		s.ActiveflowID = g4FlowID
		return s
	}
	af := &fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: g4ActiveID}}

	t.Run("no on-end flow is a no-op", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)

		if err := m.h.startOnEndFlow(context.Background(), g4Summary(summary.ReferenceTypeCall)); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("activeflow create fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.req.EXPECT().FlowV1ActiveflowCreate(gomock.Any(), uuid.Nil, g4CustomerID, g4OnEndID, fmactiveflow.ReferenceTypeAI, g4SummaryID, g4FlowID, nil, "", fmactiveflow.WebhookMethodNone).
			Return(nil, rpcErr)

		err := m.h.startOnEndFlow(context.Background(), sm())
		if !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), "could not create the activeflow") {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("variable set failure does not stop execution", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.req.EXPECT().FlowV1ActiveflowCreate(gomock.Any(), uuid.Nil, g4CustomerID, g4OnEndID, fmactiveflow.ReferenceTypeAI, g4SummaryID, g4FlowID, nil, "", fmactiveflow.WebhookMethodNone).
			Return(af, nil)
		m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), g4ActiveID, gomock.Any()).Return(rpcErr)
		m.req.EXPECT().FlowV1ActiveflowExecute(gomock.Any(), g4ActiveID).Return(nil)

		if err := m.h.startOnEndFlow(context.Background(), sm()); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("execute fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.req.EXPECT().FlowV1ActiveflowCreate(gomock.Any(), uuid.Nil, g4CustomerID, g4OnEndID, fmactiveflow.ReferenceTypeAI, g4SummaryID, g4FlowID, nil, "", fmactiveflow.WebhookMethodNone).
			Return(af, nil)
		m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), g4ActiveID, gomock.Any()).Return(nil)
		m.req.EXPECT().FlowV1ActiveflowExecute(gomock.Any(), g4ActiveID).Return(rpcErr)

		err := m.h.startOnEndFlow(context.Background(), sm())
		if !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), "could not execute the activeflow") {
			t.Errorf("unexpected err: %v", err)
		}
	})
}
