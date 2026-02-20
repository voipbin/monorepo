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

func Test_gcpHandler_Run_exitsOnConnAstDone(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &gcpHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}

	connAstDone := make(chan struct{})
	cfCtx, cfCancel := context.WithCancel(context.Background())
	defer cfCancel()

	streamCtx, streamCancel := context.WithCancel(cfCtx)

	cf := &GCPConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
		},
		Ctx:          cfCtx,
		Cancel:       cfCancel,
		StreamCtx:    streamCtx,
		StreamCancel: streamCancel,
		ConnAstDone:  connAstDone,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
			},
		},
		processDone: make(chan struct{}),
	}

	// runProcess and runKeepalive will publish events
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

	// Verify the session context was cancelled by terminate()
	select {
	case <-cfCtx.Done():
		// Context was cancelled — correct
	default:
		t.Fatal("session context was not cancelled after Run() exited")
	}
}

func Test_gcpHandler_Run_exitsOnCtxDone(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &gcpHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}

	connAstDone := make(chan struct{})
	cfCtx, cfCancel := context.WithCancel(context.Background())

	streamCtx, streamCancel := context.WithCancel(cfCtx)

	cf := &GCPConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
			},
		},
		Ctx:          cfCtx,
		Cancel:       cfCancel,
		StreamCtx:    streamCtx,
		StreamCancel: streamCancel,
		ConnAstDone:  connAstDone,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000002"),
			},
		},
		processDone: make(chan struct{}),
	}

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	done := make(chan error, 1)
	go func() {
		done <- h.Run(cf)
	}()

	// Cancel context directly (simulating SayStop)
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

func Test_gcpHandler_Run_exitsImmediatelyIfConnAstDoneAlreadyClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &gcpHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}

	// Pre-close the channel (simulating WebSocket disconnected before Init/Run)
	connAstDone := make(chan struct{})
	close(connAstDone)

	cfCtx, cfCancel := context.WithCancel(context.Background())
	defer cfCancel()

	streamCtx, streamCancel := context.WithCancel(cfCtx)

	cf := &GCPConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
			},
		},
		Ctx:          cfCtx,
		Cancel:       cfCancel,
		StreamCtx:    streamCtx,
		StreamCancel: streamCancel,
		ConnAstDone:  connAstDone,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000003"),
			},
		},
		processDone: make(chan struct{}),
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

	// Context should be cancelled by terminate()
	select {
	case <-cfCtx.Done():
		// correct
	default:
		t.Fatal("session context was not cancelled")
	}
}
