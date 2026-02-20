package streaminghandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/message"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_awsHandler_Run_exitsOnConnAstDone(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &awsHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}

	connAstDone := make(chan struct{})
	cfCtx, cfCancel := context.WithCancel(context.Background())
	defer cfCancel()

	cf := &AWSConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
			},
		},
		Ctx:         cfCtx,
		Cancel:      cfCancel,
		ConnAstDone: connAstDone,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
			},
		},
		audioCh: make(chan []byte, defaultAWSAudioChBuffer),
	}

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	done := make(chan error, 1)
	go func() {
		done <- h.Run(cf)
	}()

	// Close ConnAstDone to simulate Asterisk WebSocket disconnect
	close(connAstDone)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not exit after ConnAstDone was closed")
	}

	// Verify the context was cancelled by Run()
	select {
	case <-cfCtx.Done():
		// Context was cancelled — correct
	default:
		t.Fatal("context was not cancelled after Run() exited via ConnAstDone")
	}
}

func Test_awsHandler_Run_exitsOnCtxDone(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &awsHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}

	connAstDone := make(chan struct{})
	cfCtx, cfCancel := context.WithCancel(context.Background())

	cf := &AWSConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000002"),
			},
		},
		Ctx:         cfCtx,
		Cancel:      cfCancel,
		ConnAstDone: connAstDone,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000002"),
			},
		},
		audioCh: make(chan []byte, defaultAWSAudioChBuffer),
	}

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	done := make(chan error, 1)
	go func() {
		done <- h.Run(cf)
	}()

	// Cancel context directly
	cfCancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not exit after context cancellation")
	}
}

func Test_awsHandler_Run_exitsImmediatelyIfConnAstDoneAlreadyClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &awsHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}

	// Pre-close the channel
	connAstDone := make(chan struct{})
	close(connAstDone)

	cfCtx, cfCancel := context.WithCancel(context.Background())
	defer cfCancel()

	cf := &AWSConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000003"),
			},
		},
		Ctx:         cfCtx,
		Cancel:      cfCancel,
		ConnAstDone: connAstDone,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000003"),
			},
		},
		audioCh: make(chan []byte, defaultAWSAudioChBuffer),
	}

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	done := make(chan error, 1)
	go func() {
		done <- h.Run(cf)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not exit immediately when ConnAstDone was already closed")
	}

	// Context should be cancelled
	select {
	case <-cfCtx.Done():
		// correct
	default:
		t.Fatal("context was not cancelled")
	}
}
