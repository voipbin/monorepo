package streaminghandler

import (
	"context"
	"testing"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Start(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		transcribeID  uuid.UUID
		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     transcript.Direction
		provider      transcribe.Provider

		responseUUID          uuid.UUID
		responseExternalMedia *cmexternalmedia.ExternalMedia
	}{
		{
			name: "normal - verifies ExternalMediaStart params and cleanup on WebSocket failure",

			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			transcribeID:  uuid.FromStringOrNil("e210a336-e9df-11ef-b5e9-bbbc7edb0445"),
			referenceType: transcribe.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			direction:     transcript.DirectionIn,
			provider:      transcribe.ProviderEmpty,

			responseUUID: uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID:       uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
				MediaURI: "ws://127.0.0.1:0/invalid", // invalid URI so websocketConnect fails
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &streamingHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				mapStreaming:  make(map[uuid.UUID]*streaming.Streaming),
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStarted, gomock.Any())

			// Verify ExternalMediaStart is called with WebSocket parameters
			mockReq.EXPECT().CallV1ExternalMediaStart(
				ctx,
				tt.responseUUID,
				cmexternalmedia.ReferenceType(tt.referenceType),
				tt.referenceID,
				"INCOMING",            // WebSocket: service dials out
				defaultEncapsulation,  // "none"
				defaultTransport,      // "websocket"
				"",                    // transportData
				defaultConnectionType, // "server"
				defaultFormat,         // "slin"
				cmexternalmedia.Direction(tt.direction),
				cmexternalmedia.DirectionNone,
			).Return(tt.responseExternalMedia, nil)

			// WebSocket connect will fail → expect cleanup of orphaned external media and streaming record
			mockReq.EXPECT().CallV1ExternalMediaStop(ctx, tt.responseExternalMedia.ID).Return(tt.responseExternalMedia, nil)
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStopped, gomock.Any())

			// Start should return error because websocketConnect fails
			_, err := h.Start(ctx, tt.customerID, tt.transcribeID, tt.referenceType, tt.referenceID, tt.language, tt.direction, tt.provider)
			if err == nil {
				t.Error("Expected error from Start (WebSocket connect should fail), got nil")
			}

			// Verify streaming record was cleaned up from the map
			if _, errGet := h.Get(ctx, tt.responseUUID); errGet == nil {
				t.Error("Expected streaming record to be deleted after WebSocket connect failure")
			}
		})
	}
}

func Test_getProviderFunc(t *testing.T) {
	tests := []struct {
		name      string
		gcpClient bool
		awsClient bool
		provider  STTProvider
		expectNil bool
	}{
		{
			name:      "GCP initialized, request GCP",
			gcpClient: true,
			awsClient: false,
			provider:  STTProviderGCP,
			expectNil: false,
		},
		{
			name:      "GCP not initialized, request GCP",
			gcpClient: false,
			awsClient: true,
			provider:  STTProviderGCP,
			expectNil: true,
		},
		{
			name:      "AWS initialized, request AWS",
			gcpClient: false,
			awsClient: true,
			provider:  STTProviderAWS,
			expectNil: false,
		},
		{
			name:      "AWS not initialized, request AWS",
			gcpClient: true,
			awsClient: false,
			provider:  STTProviderAWS,
			expectNil: true,
		},
		{
			name:      "both initialized, request GCP",
			gcpClient: true,
			awsClient: true,
			provider:  STTProviderGCP,
			expectNil: false,
		},
		{
			name:      "both initialized, request AWS",
			gcpClient: true,
			awsClient: true,
			provider:  STTProviderAWS,
			expectNil: false,
		},
		{
			name:      "neither initialized, request GCP",
			gcpClient: false,
			awsClient: false,
			provider:  STTProviderGCP,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &streamingHandler{}
			if tt.gcpClient {
				h.gcpClient = &speech.Client{}
			}
			if tt.awsClient {
				h.awsClient = &transcribestreaming.Client{}
			}

			fn := h.getProviderFunc(tt.provider)
			if tt.expectNil && fn != nil {
				t.Errorf("Expected nil function, got non-nil")
			}
			if !tt.expectNil && fn == nil {
				t.Errorf("Expected non-nil function, got nil")
			}
		})
	}
}

func Test_runSTT_handlerSelection(t *testing.T) {
	// Test the handler list building logic of runSTT.
	// We verify which providers get selected by checking the list that would be built.
	// This tests the selection algorithm without needing real STT connections.

	tests := []struct {
		name           string
		provider       transcribe.Provider
		gcpInitialized bool
		awsInitialized bool
		expectGCPFirst bool // expect GCP handler to appear first
		expectAWSFirst bool // expect AWS handler to appear first
		expectCount    int  // expected number of handlers
	}{
		{
			name:           "empty provider, both initialized - default order GCP first",
			provider:       "",
			gcpInitialized: true,
			awsInitialized: true,
			expectGCPFirst: true,
			expectCount:    2,
		},
		{
			name:           "empty provider, only AWS initialized",
			provider:       "",
			gcpInitialized: false,
			awsInitialized: true,
			expectAWSFirst: true,
			expectCount:    1,
		},
		{
			name:           "empty provider, only GCP initialized",
			provider:       "",
			gcpInitialized: true,
			awsInitialized: false,
			expectGCPFirst: true,
			expectCount:    1,
		},
		{
			name:           "provider gcp, both initialized - GCP first then AWS fallback",
			provider:       "gcp",
			gcpInitialized: true,
			awsInitialized: true,
			expectGCPFirst: true,
			expectCount:    2,
		},
		{
			name:           "provider aws, both initialized - AWS first then GCP fallback",
			provider:       "aws",
			gcpInitialized: true,
			awsInitialized: true,
			expectAWSFirst: true,
			expectCount:    2,
		},
		{
			name:           "provider gcp, GCP not initialized - fallback to AWS only",
			provider:       "gcp",
			gcpInitialized: false,
			awsInitialized: true,
			expectAWSFirst: true,
			expectCount:    1,
		},
		{
			name:           "provider aws, AWS not initialized - fallback to GCP only",
			provider:       "aws",
			gcpInitialized: true,
			awsInitialized: false,
			expectGCPFirst: true,
			expectCount:    1,
		},
		{
			name:           "invalid provider azure, both initialized - default order",
			provider:       "azure",
			gcpInitialized: true,
			awsInitialized: true,
			expectGCPFirst: true,
			expectCount:    2,
		},
		{
			name:           "provider GCP uppercase, both initialized - GCP first",
			provider:       "GCP",
			gcpInitialized: true,
			awsInitialized: true,
			expectGCPFirst: true,
			expectCount:    2,
		},
		{
			name:           "neither initialized - zero handlers",
			provider:       "",
			gcpInitialized: false,
			awsInitialized: false,
			expectCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &streamingHandler{}
			if tt.gcpInitialized {
				h.gcpClient = &speech.Client{}
			}
			if tt.awsInitialized {
				h.awsClient = &transcribestreaming.Client{}
			}

			// Build handler list using same logic as runSTT
			handlers := []struct {
				provider STTProvider
			}{}

			provider := tt.provider
			if provider != "" {
				sttProvider, err := validateProvider(string(provider))
				if err == nil {
					if h.getProviderFunc(sttProvider) != nil {
						handlers = append(handlers, struct{ provider STTProvider }{sttProvider})
					}
				}
			}

			for _, p := range defaultProviderOrder {
				if h.getProviderFunc(p) == nil {
					continue
				}
				if len(handlers) > 0 && provider != "" {
					requested, reqErr := validateProvider(string(provider))
					if reqErr == nil && p == requested {
						continue
					}
				}
				handlers = append(handlers, struct{ provider STTProvider }{p})
			}

			if len(handlers) != tt.expectCount {
				t.Errorf("Expected %d handlers, got %d", tt.expectCount, len(handlers))
				return
			}

			if tt.expectCount > 0 {
				if tt.expectGCPFirst && handlers[0].provider != STTProviderGCP {
					t.Errorf("Expected GCP first, got %s", handlers[0].provider)
				}
				if tt.expectAWSFirst && handlers[0].provider != STTProviderAWS {
					t.Errorf("Expected AWS first, got %s", handlers[0].provider)
				}
			}
		})
	}
}
