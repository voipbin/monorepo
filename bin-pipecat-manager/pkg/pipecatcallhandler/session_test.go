package pipecatcallhandler

import (
	"net"
	"sync"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestSessionCreate(t *testing.T) {
	tests := []struct {
		name string

		pc                  *pipecatcall.Pipecatcall
		asteriskStreamingID uuid.UUID
		llmKey              string

		existingSession *pipecatcall.Session

		expectErr bool
	}{
		{
			name: "success",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
			},
			asteriskStreamingID: uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
			llmKey:              "test-llm-key",

			existingSession: nil,

			expectErr: false,
		},
		{
			name: "session already exists",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},
			},
			asteriskStreamingID: uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
			llmKey:              "test-llm-key",

			existingSession: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				},
			},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &pipecatcallHandler{
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
			}

			if tt.existingSession != nil {
				h.mapPipecatcallSession[tt.existingSession.ID] = tt.existingSession
			}

			result, err := h.SessionCreate(tt.pc, tt.asteriskStreamingID, nil, tt.llmKey)

			if tt.expectErr {
				if err == nil {
					t.Errorf("SessionCreate() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("SessionCreate() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("SessionCreate() returned nil result")
				return
			}

			if result.ID != tt.pc.ID {
				t.Errorf("SessionCreate() ID = %v, want %v", result.ID, tt.pc.ID)
			}

			if result.CustomerID != tt.pc.CustomerID {
				t.Errorf("SessionCreate() CustomerID = %v, want %v", result.CustomerID, tt.pc.CustomerID)
			}

			if result.LLMKey != tt.llmKey {
				t.Errorf("SessionCreate() LLMKey = %v, want %v", result.LLMKey, tt.llmKey)
			}

			if result.AsteriskStreamingID != tt.asteriskStreamingID {
				t.Errorf("SessionCreate() AsteriskStreamingID = %v, want %v", result.AsteriskStreamingID, tt.asteriskStreamingID)
			}

			if result.Ctx == nil {
				t.Errorf("SessionCreate() Ctx is nil")
			}

			if result.Cancel == nil {
				t.Errorf("SessionCreate() Cancel is nil")
			}

			if result.RunnerWebsocketChan == nil {
				t.Errorf("SessionCreate() RunnerWebsocketChan is nil")
			}
		})
	}
}

func TestSessionGet(t *testing.T) {
	tests := []struct {
		name string

		id              uuid.UUID
		existingSession *pipecatcall.Session

		expectErr bool
	}{
		{
			name: "success",

			id: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			existingSession: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				},
			},

			expectErr: false,
		},
		{
			name: "session not found",

			id:              uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			existingSession: nil,

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &pipecatcallHandler{
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
			}

			if tt.existingSession != nil {
				h.mapPipecatcallSession[tt.existingSession.ID] = tt.existingSession
			}

			result, err := h.SessionGet(tt.id)

			if tt.expectErr {
				if err == nil {
					t.Errorf("SessionGet() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("SessionGet() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("SessionGet() returned nil result")
				return
			}

			if result.ID != tt.id {
				t.Errorf("SessionGet() ID = %v, want %v", result.ID, tt.id)
			}
		})
	}
}

func TestSessionsetAsteriskInfo(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &pipecatcallHandler{}

	pc := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
		},
	}

	streamingID := uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d")

	// Create a mock connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			defer conn.Close()
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	h.SessionsetAsteriskInfo(pc, streamingID, conn)

	if pc.AsteriskStreamingID != streamingID {
		t.Errorf("SessionsetAsteriskInfo() AsteriskStreamingID = %v, want %v", pc.AsteriskStreamingID, streamingID)
	}

	if pc.AsteriskConn != conn {
		t.Errorf("SessionsetAsteriskInfo() AsteriskConn mismatch")
	}
}

func TestSessionDelete(t *testing.T) {
	id := uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8")

	h := &pipecatcallHandler{
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	// Add a session
	h.mapPipecatcallSession[id] = &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID: id,
		},
	}

	// Verify it exists
	if _, ok := h.mapPipecatcallSession[id]; !ok {
		t.Errorf("Session should exist before delete")
	}

	// Delete it
	h.sessionDelete(id)

	// Verify it's gone
	if _, ok := h.mapPipecatcallSession[id]; ok {
		t.Errorf("Session should not exist after delete")
	}
}

func TestSessionStop(t *testing.T) {
	tests := []struct {
		name string

		id              uuid.UUID
		existingSession *pipecatcall.Session
		pythonRunnerErr error
	}{
		{
			name: "success with connection",

			id: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			existingSession: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				},
			},
			pythonRunnerErr: nil,
		},
		{
			name: "session not found",

			id:              uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			existingSession: nil,
			pythonRunnerErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPythonRunner := NewMockPythonRunner(mc)

			h := &pipecatcallHandler{
				pythonRunner:          mockPythonRunner,
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
			}

			if tt.existingSession != nil {
				h.mapPipecatcallSession[tt.existingSession.ID] = tt.existingSession
				mockPythonRunner.EXPECT().Stop(gomock.Any(), tt.id).Return(tt.pythonRunnerErr)
			}

			h.SessionStop(tt.id)

			// Verify session was deleted if it existed
			if tt.existingSession != nil {
				if _, ok := h.mapPipecatcallSession[tt.id]; ok {
					t.Errorf("SessionStop() session should be deleted")
				}
			}
		})
	}
}
