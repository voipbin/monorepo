package pipecatcallhandler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/internal/logtest"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

func Test_SessionStop_logsAndCleansUp(t *testing.T) {
	tests := []struct {
		name string

		droppedFrames int64
		stopErr       error

		expectLogs    []string
		notExpectLogs []string
	}{
		{
			name:          "dropped frames are reported",
			droppedFrames: 3,
			expectLogs:    []string{"Session had 3 dropped audio frames."},
			notExpectLogs: []string{"Could not stop the pipecatcall in python runner"},
		},
		{
			name:          "python runner stop fails",
			stopErr:       fmt.Errorf("runner unreachable"),
			expectLogs:    []string{"Could not stop the pipecatcall in python runner", "runner unreachable"},
			notExpectLogs: []string{"dropped audio frames"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			logs := logtest.New(t)
			mockPython := NewMockPythonRunner(mc)
			h := &pipecatcallHandler{
				pythonRunner:          mockPython,
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
				muTerminated:          sync.Mutex{},
				terminatedPublished:   make(map[uuid.UUID]struct{}),
			}

			id := uuid.FromStringOrNil("a1b2c3d4-6001-4000-8000-000000000001")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			se := &pipecatcall.Session{
				Identity: commonidentity.Identity{ID: id},
				Ctx:      ctx,
				Cancel:   cancel,
			}
			se.DroppedFrames.Store(tt.droppedFrames)
			h.mapPipecatcallSession[id] = se

			mockPython.EXPECT().Stop(gomock.Any(), id).Return(tt.stopErr)

			h.SessionStop(id)

			if _, ok := h.mapPipecatcallSession[id]; ok {
				t.Errorf("the session must be deleted even when the runner stop fails")
			}
			if ctx.Err() == nil {
				t.Errorf("the session context must be cancelled")
			}
			dump := logs.Dump()
			for _, expect := range tt.expectLogs {
				if !strings.Contains(dump, expect) {
					t.Errorf("Wrong log. expect contains: %q", expect)
				}
			}
			for _, notExpect := range tt.notExpectLogs {
				if strings.Contains(dump, notExpect) {
					t.Errorf("Wrong log. must not contain: %q", notExpect)
				}
			}
		})
	}
}
