package servicehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_SpeakingCreate(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		referenceType tmstreaming.ReferenceType
		referenceID   uuid.UUID
		language      string
		provider      string
		voiceID       string
		direction     tmstreaming.Direction

		responseSpeaking *tmspeaking.Speaking

		expectRes *tmspeaking.WebhookMessage
		expectErr bool
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			referenceType: tmstreaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "voice123",
			direction:     tmstreaming.DirectionIncoming,

			responseSpeaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status: tmspeaking.StatusActive,
			},

			expectRes: &tmspeaking.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status: tmspeaking.StatusActive,
			},
			expectErr: false,
		},
		{
			name: "no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: 0,
			},
			referenceType: tmstreaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "",
			voiceID:       "",
			direction:     tmstreaming.DirectionIncoming,

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			if !tt.expectErr {
				mockReq.EXPECT().TTSV1SpeakingCreate(
					ctx,
					tt.agent.CustomerID,
					tt.referenceType,
					tt.referenceID,
					tt.language,
					tt.provider,
					tt.voiceID,
					tt.direction,
				).Return(tt.responseSpeaking, nil)
			}

			res, err := h.SpeakingCreate(ctx, tt.agent, tt.referenceType, tt.referenceID, tt.language, tt.provider, tt.voiceID, tt.direction)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SpeakingGet(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		speakingID uuid.UUID

		responseSpeaking *tmspeaking.Speaking

		expectRes *tmspeaking.WebhookMessage
		expectErr bool
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),

			responseSpeaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status: tmspeaking.StatusActive,
			},

			expectRes: &tmspeaking.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status: tmspeaking.StatusActive,
			},
			expectErr: false,
		},
		{
			name: "no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),

			responseSpeaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TTSV1SpeakingGet(ctx, tt.speakingID).Return(tt.responseSpeaking, nil)

			res, err := h.SpeakingGet(ctx, tt.agent, tt.speakingID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SpeakingList(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		responseTime      string
		responseSpeakings []*tmspeaking.Speaking

		expectToken   string
		expectFilters map[tmspeaking.Field]any
		expectRes     []*tmspeaking.WebhookMessage
	}{
		{
			name: "normal with token",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			pageToken: "2024-01-01T00:00:00.000000Z",
			pageSize:  10,

			responseSpeakings: []*tmspeaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
				},
			},

			expectToken: "2024-01-01T00:00:00.000000Z",
			expectFilters: map[tmspeaking.Field]any{
				tmspeaking.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				tmspeaking.FieldDeleted:    false,
			},
			expectRes: []*tmspeaking.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
				},
			},
		},
		{
			name: "empty page token defaults",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			pageToken: "",
			pageSize:  100,

			responseTime: "2024-06-01T00:00:00.000000Z",
			responseSpeakings: []*tmspeaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
				},
			},

			expectToken: "2024-06-01T00:00:00.000000Z",
			expectFilters: map[tmspeaking.Field]any{
				tmspeaking.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				tmspeaking.FieldDeleted:    false,
			},
			expectRes: []*tmspeaking.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			if tt.pageToken == "" {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseTime)
			}

			mockReq.EXPECT().TTSV1SpeakingGets(ctx, tt.expectToken, tt.pageSize, tt.expectFilters).Return(tt.responseSpeakings, nil)

			res, err := h.SpeakingList(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SpeakingSay(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		speakingID uuid.UUID
		text       string

		responseSpeakingGet *tmspeaking.Speaking
		responseSpeakingSay *tmspeaking.Speaking

		expectRes *tmspeaking.WebhookMessage
		expectErr bool
	}{
		{
			name: "normal pod-targeted",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
			text:       "hello world",

			responseSpeakingGet: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID:  "tts-pod-1",
				Status: tmspeaking.StatusActive,
			},
			responseSpeakingSay: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID:  "tts-pod-1",
				Status: tmspeaking.StatusActive,
			},

			expectRes: &tmspeaking.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status: tmspeaking.StatusActive,
			},
			expectErr: false,
		},
		{
			name: "no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
			text:       "hello world",

			responseSpeakingGet: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID: "tts-pod-1",
			},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TTSV1SpeakingGet(ctx, tt.speakingID).Return(tt.responseSpeakingGet, nil)

			if !tt.expectErr {
				mockReq.EXPECT().TTSV1SpeakingSay(ctx, tt.responseSpeakingGet.PodID, tt.speakingID, tt.text).Return(tt.responseSpeakingSay, nil)
			}

			res, err := h.SpeakingSay(ctx, tt.agent, tt.speakingID, tt.text)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SpeakingFlush(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		speakingID uuid.UUID

		responseSpeakingGet   *tmspeaking.Speaking
		responseSpeakingFlush *tmspeaking.Speaking

		expectRes *tmspeaking.WebhookMessage
		expectErr bool
	}{
		{
			name: "normal pod-targeted",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),

			responseSpeakingGet: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID:  "tts-pod-1",
				Status: tmspeaking.StatusActive,
			},
			responseSpeakingFlush: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID:  "tts-pod-1",
				Status: tmspeaking.StatusActive,
			},

			expectRes: &tmspeaking.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status: tmspeaking.StatusActive,
			},
			expectErr: false,
		},
		{
			name: "no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),

			responseSpeakingGet: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID: "tts-pod-1",
			},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TTSV1SpeakingGet(ctx, tt.speakingID).Return(tt.responseSpeakingGet, nil)

			if !tt.expectErr {
				mockReq.EXPECT().TTSV1SpeakingFlush(ctx, tt.responseSpeakingGet.PodID, tt.speakingID).Return(tt.responseSpeakingFlush, nil)
			}

			res, err := h.SpeakingFlush(ctx, tt.agent, tt.speakingID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SpeakingStop(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		speakingID uuid.UUID

		responseSpeakingGet  *tmspeaking.Speaking
		responseSpeakingStop *tmspeaking.Speaking

		expectRes *tmspeaking.WebhookMessage
		expectErr bool
	}{
		{
			name: "normal pod-targeted",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),

			responseSpeakingGet: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID:  "tts-pod-1",
				Status: tmspeaking.StatusActive,
			},
			responseSpeakingStop: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID:  "tts-pod-1",
				Status: tmspeaking.StatusStopped,
			},

			expectRes: &tmspeaking.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status: tmspeaking.StatusStopped,
			},
			expectErr: false,
		},
		{
			name: "no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),

			responseSpeakingGet: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID: "tts-pod-1",
			},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TTSV1SpeakingGet(ctx, tt.speakingID).Return(tt.responseSpeakingGet, nil)

			if !tt.expectErr {
				mockReq.EXPECT().TTSV1SpeakingStop(ctx, tt.responseSpeakingGet.PodID, tt.speakingID).Return(tt.responseSpeakingStop, nil)
			}

			res, err := h.SpeakingStop(ctx, tt.agent, tt.speakingID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SpeakingDelete(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		speakingID uuid.UUID

		responseSpeakingGet    *tmspeaking.Speaking
		responseSpeakingStop   *tmspeaking.Speaking
		responseSpeakingDelete *tmspeaking.Speaking

		expectRes *tmspeaking.WebhookMessage
		expectErr bool
	}{
		{
			name: "normal stops active session first",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),

			responseSpeakingGet: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID:  "tts-pod-1",
				Status: tmspeaking.StatusActive,
			},
			responseSpeakingStop: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status: tmspeaking.StatusStopped,
			},
			responseSpeakingDelete: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: func() *time.Time { t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
			},

			expectRes: &tmspeaking.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: func() *time.Time { t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
			},
			expectErr: false,
		},
		{
			name: "already stopped skips stop",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			speakingID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),

			responseSpeakingGet: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PodID:  "tts-pod-1",
				Status: tmspeaking.StatusStopped,
			},
			responseSpeakingDelete: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: func() *time.Time { t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
			},

			expectRes: &tmspeaking.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: func() *time.Time { t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			// speakingGet
			mockReq.EXPECT().TTSV1SpeakingGet(ctx, tt.speakingID).Return(tt.responseSpeakingGet, nil)

			// Stop if active or initiating
			if tt.responseSpeakingGet.Status == tmspeaking.StatusActive || tt.responseSpeakingGet.Status == tmspeaking.StatusInitiating {
				mockReq.EXPECT().TTSV1SpeakingStop(ctx, tt.responseSpeakingGet.PodID, tt.speakingID).Return(tt.responseSpeakingStop, nil)
			}

			// Delete
			mockReq.EXPECT().TTSV1SpeakingDelete(ctx, tt.speakingID).Return(tt.responseSpeakingDelete, nil)

			res, err := h.SpeakingDelete(ctx, tt.agent, tt.speakingID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
