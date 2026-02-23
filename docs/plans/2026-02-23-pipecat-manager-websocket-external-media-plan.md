# Pipecat Manager WebSocket External Media Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace pipecat-manager's Audiosocket TCP external media with WebSocket external media, eliminating sample rate conversion and simplifying the connection model.

**Architecture:** Asterisk acts as WebSocket server, pipecat-manager dials `em.MediaURI` to connect. Audio format is slin16 (16kHz, 16-bit signed linear PCM) end-to-end, matching Pipecat Python's native 16kHz. The internal Go↔Python WebSocket connections (input/output) are unchanged.

**Tech Stack:** Go, gorilla/websocket, gomock, zaf/resample (retained for safety), protobuf

**Design doc:** `docs/plans/2026-02-23-pipecat-manager-websocket-external-media-design.md`

---

### Task 1: Update Session Model

**Files:**
- Modify: `bin-pipecat-manager/models/pipecatcall/session.go`

**Step 1: Update the Session struct**

Replace `net.Conn` fields with WebSocket equivalents:

```go
package pipecatcall

import (
	"context"
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

type Session struct {
	identity.Identity // copied from pipecatcall

	PipecatcallReferenceType ReferenceType `json:"reference_type,omitempty"` // copied from pipecatcall
	PipecatcallReferenceID   uuid.UUID     `json:"reference_id,omitempty"`   // copied from pipecatcall

	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	// Runner info
	RunnerWebsocketChan chan *SessionFrame `json:"-"`

	// asterisk info
	AsteriskStreamingID uuid.UUID       `json:"-"`
	ConnAst             *websocket.Conn `json:"-"`
	ConnAstDone         chan struct{}    `json:"-"`

	// llm
	LLMKey     string `json:"-"`
	LLMBotText string `json:"-"`
}
```

**Step 2: Verify the change compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go build ./...`

Expected: Compile errors in files that reference `AsteriskConn net.Conn` — this is expected and will be fixed in subsequent tasks.

---

### Task 2: Add Asterisk WebSocket Helper Functions

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/websocket.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/websocket_test.go`

**Step 1: Write tests for the new WebSocket helpers**

Add to `websocket_test.go`:

```go
func Test_websocketAsteriskWrite(t *testing.T) {
	tests := []struct {
		name string

		data      []byte
		frameSize int

		expectErr     bool
		expectWrites  int
	}{
		{
			name:         "single frame",
			data:         make([]byte, 640),
			frameSize:    640,
			expectErr:    false,
			expectWrites: 1,
		},
		{
			name:         "multiple frames",
			data:         make([]byte, 1280),
			frameSize:    640,
			expectErr:    false,
			expectWrites: 2,
		},
		{
			name:         "data smaller than frame size",
			data:         make([]byte, 320),
			frameSize:    640,
			expectErr:    false,
			expectWrites: 1,
		},
		{
			name:         "empty data",
			data:         []byte{},
			frameSize:    640,
			expectErr:    false,
			expectWrites: 0,
		},
		{
			name:      "invalid frame size",
			data:      make([]byte, 640),
			frameSize: 0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockWS := NewMockWebsocketHandler(mc)

			if tt.expectWrites > 0 {
				mockWS.EXPECT().WriteMessage(gomock.Any(), websocket.BinaryMessage, gomock.Any()).Times(tt.expectWrites).Return(nil)
			}

			conn := &websocket.Conn{} // placeholder, WriteMessage is mocked

			h := &pipecatcallHandler{
				websocketHandler: mockWS,
			}

			err := h.websocketAsteriskWrite(context.Background(), conn, tt.data, tt.frameSize)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_websocketAsteriskWrite_contextCancelled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockWS := NewMockWebsocketHandler(mc)

	// First write succeeds, then context is cancelled before second frame
	mockWS.EXPECT().WriteMessage(gomock.Any(), websocket.BinaryMessage, gomock.Any()).Return(nil).Times(1)

	conn := &websocket.Conn{}

	h := &pipecatcallHandler{
		websocketHandler: mockWS,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// With cancelled context and multi-frame data, should return context error
	err := h.websocketAsteriskWrite(ctx, conn, make([]byte, 1280), 640)
	if err == nil {
		t.Errorf("expected context error but got nil")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go test ./pkg/pipecatcallhandler/... -run Test_websocketAsterisk -v`

Expected: FAIL — `websocketAsteriskWrite` not defined.

**Step 3: Add the WebSocket helper functions to websocket.go**

Add `websocketAsteriskConnect`, `websocketAsteriskWrite`, and `runWebSocketAsteriskRead` to `websocket.go`. Also add the new constants. The `WebsocketHandler` interface gets a `DialContext` method added.

```go
// Add to the WebsocketHandler interface:
type WebsocketHandler interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
	ReadMessage(conn *websocket.Conn) (int, []byte, error)
	WriteMessage(conn *websocket.Conn, messageType int, data []byte) error
	DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

// Add to websocketHandler:
func (h *websocketHandler) DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	dialer := websocket.Dialer{
		Subprotocols: []string{websocketAsteriskSubprotocol},
	}
	return dialer.DialContext(ctx, urlStr, requestHeader)
}
```

Then the Asterisk-specific functions are methods on `pipecatcallHandler` using the mockable `WebsocketHandler`:

```go
const (
	websocketAsteriskSubprotocol = "media"
	websocketAsteriskWriteDelay  = 20 * time.Millisecond
	websocketAsteriskFrameSize   = 640 // 16000 Hz * 2 bytes * 20ms
)

// websocketAsteriskConnect dials the Asterisk chan_websocket endpoint and waits for MEDIA_START.
func (h *pipecatcallHandler) websocketAsteriskConnect(ctx context.Context, mediaURI string) (*websocket.Conn, error) {
	conn, _, err := h.websocketHandler.DialContext(ctx, mediaURI, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not dial WebSocket. media_uri: %s", mediaURI)
	}

	if errDeadline := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); errDeadline != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(errDeadline, "could not set read deadline")
	}

	msgType, _, err := h.websocketHandler.ReadMessage(conn)
	if err != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(err, "could not read MEDIA_START message")
	}

	if errDeadline := conn.SetReadDeadline(time.Time{}); errDeadline != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(errDeadline, "could not clear read deadline")
	}

	if msgType != websocket.TextMessage {
		_ = conn.Close()
		return nil, errors.Errorf("expected text message for MEDIA_START, got type %d", msgType)
	}

	return conn, nil
}

// websocketAsteriskWrite fragments and sends raw audio data over a WebSocket connection
// as binary frames with 20ms pacing.
func (h *pipecatcallHandler) websocketAsteriskWrite(ctx context.Context, conn *websocket.Conn, data []byte, frameSize int) error {
	if len(data) == 0 {
		return nil
	}
	if frameSize <= 0 {
		return fmt.Errorf("frameSize must be positive, got %d", frameSize)
	}

	ticker := time.NewTicker(websocketAsteriskWriteDelay)
	defer ticker.Stop()

	offset := 0
	payloadLen := len(data)

	for offset < payloadLen {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fragmentLen := min(frameSize, payloadLen-offset)
		fragment := data[offset : offset+fragmentLen]

		if err := h.websocketHandler.WriteMessage(conn, websocket.BinaryMessage, fragment); err != nil {
			return errors.Wrapf(err, "failed to write WebSocket binary frame")
		}

		offset += fragmentLen

		if offset >= payloadLen {
			break
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// runWebSocketAsteriskRead reads from the WebSocket connection to handle ping/pong and close frames.
// Closes doneCh when the connection is closed or encounters an error.
func runWebSocketAsteriskRead(conn *websocket.Conn, doneCh chan struct{}) {
	log := logrus.WithField("func", "runWebSocketAsteriskRead")
	defer close(doneCh)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Debugf("Asterisk WebSocket closed normally: %v", err)
			} else {
				log.Errorf("Asterisk WebSocket read error: %v", err)
			}
			return
		}
	}
}
```

**Step 4: Regenerate mocks**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go generate ./pkg/pipecatcallhandler/...`

**Step 5: Run tests to verify they pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go test ./pkg/pipecatcallhandler/... -run Test_websocketAsterisk -v`

Expected: PASS

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git add bin-pipecat-manager/models/pipecatcall/session.go bin-pipecat-manager/pkg/pipecatcallhandler/websocket.go bin-pipecat-manager/pkg/pipecatcallhandler/websocket_test.go bin-pipecat-manager/pkg/pipecatcallhandler/mock_websocket.go
git commit -m "NOJIRA-pipecat-manager-websocket-external-media

- bin-pipecat-manager: Update Session model to use *websocket.Conn instead of net.Conn
- bin-pipecat-manager: Add Asterisk WebSocket connect, write, and read lifecycle helpers
- bin-pipecat-manager: Add DialContext to WebsocketHandler interface
- bin-pipecat-manager: Add tests for WebSocket write with fragmentation and pacing"
```

---

### Task 3: Update Constants and Remove listenAddress

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/main.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/main_test.go`
- Modify: `bin-pipecat-manager/cmd/pipecat-manager/main.go`
- Modify: `bin-pipecat-manager/cmd/pipecat-control/main.go`

**Step 1: Update constants in main.go**

Replace the external media defaults:

```go
const (
	defaultEncapsulation  = string(cmexternalmedia.EncapsulationNone)
	defaultTransport      = string(cmexternalmedia.TransportWebsocket)
	defaultConnectionType = "server"
	defaultFormat         = "slin16"
)
```

Remove `listenAddress` from the struct and constructor:

```go
type pipecatcallHandler struct {
	utilHandler    utilhandler.UtilHandler
	requestHandler requesthandler.RequestHandler
	notifyHandler  notifyhandler.NotifyHandler
	db             dbhandler.DBHandler
	toolHandler    toolhandler.ToolHandler

	pythonRunner        PythonRunner
	audiosocketHandler  AudiosocketHandler
	websocketHandler    WebsocketHandler
	pipecatframeHandler PipecatframeHandler

	hostID string

	mapPipecatcallSession map[uuid.UUID]*pipecatcall.Session
	muPipecatcallSession  sync.Mutex
}

func NewPipecatcallHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	dbHandler dbhandler.DBHandler,
	toolHandler toolhandler.ToolHandler,
	hostID string,
) PipecatcallHandler {
	return &pipecatcallHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		requestHandler: reqHandler,
		notifyHandler:  notifyHandler,
		db:             dbHandler,
		toolHandler:    toolHandler,

		pythonRunner:        NewPythonRunner(),
		audiosocketHandler:  NewAudiosocketHandler(),
		websocketHandler:    NewWebsocketHandler(),
		pipecatframeHandler: NewPipecatframeHandler(),

		hostID: hostID,

		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}
}
```

Remove `Run()` from the `PipecatcallHandler` interface (it's no longer needed — no TCP listener).

**Step 2: Update cmd/pipecat-manager/main.go**

- Remove the `listenAddress` variable and the `listenIP` → `listenAddress` construction (lines 115-119)
- Update `NewPipecatcallHandler` call to remove `listenAddress` parameter (keep `listenIP` as `hostID`)
- Remove the `runStreaming` function and its call (the TCP listener goroutine is no longer needed)

```go
// In run() function:
pipecatcallHandler := pipecatcallhandler.NewPipecatcallHandler(requestHandler, notifyHandler, dbHandler, toolHandler, listenIP)
// Remove: listenAddress variable
// Remove: runStreaming call and function
```

**Step 3: Update cmd/pipecat-control/main.go**

Remove the `listenAddress` parameter from `NewPipecatcallHandler`:

```go
return pipecatcallhandler.NewPipecatcallHandler(reqHandler, notifyHandler, db, toolHandler, "cli-host"), nil
```

**Step 4: Verify compilation**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go build ./...`

Expected: May have errors in files referencing `h.listenAddress` or `AsteriskConn` — those will be fixed in subsequent tasks.

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git add bin-pipecat-manager/pkg/pipecatcallhandler/main.go bin-pipecat-manager/cmd/pipecat-manager/main.go bin-pipecat-manager/cmd/pipecat-control/main.go
git commit -m "NOJIRA-pipecat-manager-websocket-external-media

- bin-pipecat-manager: Update external media defaults to websocket/none/server/slin16
- bin-pipecat-manager: Remove listenAddress from handler and constructor
- bin-pipecat-manager: Remove Run() TCP listener from interface
- bin-pipecat-manager: Remove runStreaming goroutine from daemon"
```

---

### Task 4: Update Session Management

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/session.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/session_test.go`

**Step 1: Write updated tests for SessionCreate and SessionStop**

Update `session_test.go`:
- `TestSessionCreate`: Remove `asteriskStreamingID` parameter since the session no longer takes `net.Conn`. The new signature takes `*websocket.Conn` and streaming ID.
- `TestSessionStop`: Update to verify `ConnAst.Close()` is called on a `*websocket.Conn` (can be nil for sessions without Asterisk).
- Remove `TestSessionsetAsteriskInfo` (the `SessionsetAsteriskInfo` function is removed since connection is set at creation time).

**Step 2: Update session.go**

Update `SessionCreate` signature to accept `*websocket.Conn` and `chan struct{}`:

```go
func (h *pipecatcallHandler) SessionCreate(
	pc *pipecatcall.Pipecatcall,
	asteriskStreamingID uuid.UUID,
	connAst *websocket.Conn,
	connAstDone chan struct{},
	llmKey string,
) (*pipecatcall.Session, error) {
	ctx, cancel := context.WithCancel(context.Background())
	res := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         pc.ID,
			CustomerID: pc.CustomerID,
		},

		PipecatcallReferenceType: pc.ReferenceType,
		PipecatcallReferenceID:   pc.ReferenceID,

		Ctx:    ctx,
		Cancel: cancel,

		RunnerWebsocketChan: make(chan *pipecatcall.SessionFrame, defaultRunnerWebsocketChanBufferSize),

		AsteriskStreamingID: asteriskStreamingID,
		ConnAst:             connAst,
		ConnAstDone:         connAstDone,

		LLMKey: llmKey,
	}
	// ... rest unchanged
}
```

Update `SessionStop` to close `*websocket.Conn`:

```go
func (h *pipecatcallHandler) SessionStop(id uuid.UUID) {
	// ...
	if pc.ConnAst != nil {
		if errClose := pc.ConnAst.Close(); errClose != nil {
			log.Errorf("Could not close the asterisk connection. err: %v", errClose)
		}
	}
	// ... rest unchanged
}
```

Remove `SessionsetAsteriskInfo` (no longer needed — connection set at creation).

**Step 3: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go test ./pkg/pipecatcallhandler/... -run TestSession -v`

Expected: PASS

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git add bin-pipecat-manager/pkg/pipecatcallhandler/session.go bin-pipecat-manager/pkg/pipecatcallhandler/session_test.go
git commit -m "NOJIRA-pipecat-manager-websocket-external-media

- bin-pipecat-manager: Update SessionCreate to accept *websocket.Conn and done channel
- bin-pipecat-manager: Update SessionStop to close WebSocket connection
- bin-pipecat-manager: Remove SessionsetAsteriskInfo helper"
```

---

### Task 5: Update External Media Start (start.go)

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/start.go`

**Step 1: Update startReferenceTypeCall**

After creating external media, dial the WebSocket, create session, and start goroutines:

```go
func (h *pipecatcallHandler) startReferenceTypeCall(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startReferenceTypeCall",
		"pipecatcall_id": pc.ID,
	})

	c, err := h.requestHandler.CallV1CallGet(ctx, pc.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get call info")
	}
	log.WithField("call", c).Debugf("Retrieved call info. call_id: %s", c.ID)

	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		pc.ID,
		cmexternalmedia.ReferenceTypeCall,
		c.ID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"",
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	)
	if err != nil {
		return errors.Wrapf(err, "could not create external media")
	}
	log.WithField("external_media", em).Debugf("Created external media. external_media_id: %s, media_uri: %s", em.ID, em.MediaURI)

	// Connect to Asterisk via WebSocket
	conn, err := h.websocketAsteriskConnect(ctx, em.MediaURI)
	if err != nil {
		log.Errorf("Could not connect WebSocket to Asterisk. err: %v", err)
		if _, errStop := h.requestHandler.CallV1ExternalMediaStop(ctx, em.ID); errStop != nil {
			log.Errorf("Could not stop orphaned external media. err: %v", errStop)
		}
		return errors.Wrapf(err, "could not connect to asterisk websocket")
	}
	log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

	connAstDone := make(chan struct{})

	// Create session
	llmKey := h.runGetLLMKey(ctx, pc)
	se, err := h.SessionCreate(pc, pc.ID, conn, connAstDone, llmKey)
	if err != nil {
		_ = conn.Close()
		return errors.Wrapf(err, "could not create pipecatcall session")
	}

	// Start WebSocket read lifecycle goroutine
	go runWebSocketAsteriskRead(conn, connAstDone)

	// Start pipecat runner
	go func() {
		defer se.Cancel()
		h.RunnerStart(pc, se)
	}()

	// Start media handler (reads audio from Asterisk WebSocket, sends to Python)
	go func() {
		defer se.Cancel()
		h.runAsteriskReceivedMediaHandle(se)
	}()

	// Monitor lifecycle — when context or WebSocket dies, terminate
	go func() {
		select {
		case <-se.Ctx.Done():
		case <-connAstDone:
		}
		log.Debugf("Asterisk connection or context done, terminating. pipecatcall_id: %s", pc.ID)
		h.terminate(context.Background(), pc)
	}()

	return nil
}
```

**Step 2: Apply the same pattern to startReferenceTypeAIcall**

For the `amaicall.ReferenceTypeCall` case, apply the same WebSocket connect + session creation pattern. The `default` case (non-call reference types) stays mostly the same but uses `nil` for WebSocket connection and done channel.

```go
case amaicall.ReferenceTypeCall:
	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		pc.ID,
		cmexternalmedia.ReferenceTypeCall,
		c.ReferenceID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"",
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	)
	if err != nil {
		return errors.Wrapf(err, "could not create external media")
	}
	log.WithField("external_media", em).Debugf("Created external media. external_media_id: %s, media_uri: %s", em.ID, em.MediaURI)

	conn, err := h.websocketAsteriskConnect(ctx, em.MediaURI)
	if err != nil {
		log.Errorf("Could not connect WebSocket to Asterisk. err: %v", err)
		if _, errStop := h.requestHandler.CallV1ExternalMediaStop(ctx, em.ID); errStop != nil {
			log.Errorf("Could not stop orphaned external media. err: %v", errStop)
		}
		return errors.Wrapf(err, "could not connect to asterisk websocket")
	}

	connAstDone := make(chan struct{})
	llmKey := h.runGetLLMKey(ctx, pc)
	se, err := h.SessionCreate(pc, pc.ID, conn, connAstDone, llmKey)
	if err != nil {
		_ = conn.Close()
		return errors.Wrapf(err, "could not create pipecatcall session")
	}

	go runWebSocketAsteriskRead(conn, connAstDone)

	go func() {
		defer se.Cancel()
		h.RunnerStart(pc, se)
	}()

	go func() {
		defer se.Cancel()
		h.runAsteriskReceivedMediaHandle(se)
	}()

	go func() {
		select {
		case <-se.Ctx.Done():
		case <-connAstDone:
		}
		h.terminate(context.Background(), pc)
	}()

	return nil

default:
	llmKey := h.runGetLLMKey(ctx, pc)
	se, err := h.SessionCreate(pc, uuid.Nil, nil, nil, llmKey)
	if err != nil {
		return errors.Wrapf(err, "could not create pipecatcall session")
	}

	go h.RunnerStart(pc, se)
	return nil
```

**Step 3: Verify compilation**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go build ./...`

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git add bin-pipecat-manager/pkg/pipecatcallhandler/start.go
git commit -m "NOJIRA-pipecat-manager-websocket-external-media

- bin-pipecat-manager: Update external media start to use websocket transport
- bin-pipecat-manager: Dial Asterisk MediaURI after external media creation
- bin-pipecat-manager: Create session with WebSocket connection in start flow
- bin-pipecat-manager: Add lifecycle monitoring for Asterisk WebSocket connection"
```

---

### Task 6: Rewrite Audio Read (run.go)

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go`

**Step 1: Write tests for the new audio read function**

```go
func Test_runAsteriskReceivedMediaHandle(t *testing.T) {
	tests := []struct {
		name string

		readMessages []struct {
			msgType int
			data    []byte
			err     error
		}

		expectAudioFrames int
	}{
		{
			name: "receives binary audio frames",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: fmt.Errorf("connection closed")},
			},
			expectAudioFrames: 2,
		},
		{
			name: "skips non-binary messages",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.TextMessage, data: []byte("text"), err: nil},
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: fmt.Errorf("connection closed")},
			},
			expectAudioFrames: 1,
		},
	}
	// ... test implementation using mocked WebsocketHandler
}
```

**Step 2: Rewrite run.go**

Remove the TCP listener (`Run()`, `runStart()`, `runAsteriskKeepAlive()`, `retryWithBackoff()`).

Rewrite `runAsteriskReceivedMediaHandle()` to read from WebSocket instead of Audiosocket:

```go
package pipecatcallhandler

import (
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	defaultMediaSampleRate = 16000
	defaultMediaNumChannel = 1
)

func (h *pipecatcallHandler) runAsteriskReceivedMediaHandle(se *pipecatcall.Session) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runAsteriskReceivedMediaHandle",
		"pipecatcall_id": se.ID,
	})

	if se.ConnAst == nil {
		log.Debugf("No Asterisk WebSocket connection, skipping media handle.")
		return
	}

	packetID := uint64(0)
	for {
		if se.Ctx.Err() != nil {
			log.Debugf("Context has finished. pipecatcall_id: %s", se.ID)
			return
		}

		msgType, data, err := h.websocketHandler.ReadMessage(se.ConnAst)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Debugf("Asterisk WebSocket closed normally.")
			} else {
				log.Infof("Asterisk WebSocket read error: %v", err)
			}
			return
		}

		if msgType != websocket.BinaryMessage {
			continue
		}

		if len(data) == 0 {
			continue
		}

		if errSend := h.pipecatframeHandler.SendAudio(se, packetID, data); errSend != nil {
			log.Errorf("Could not send audio frame. err: %v", errSend)
		}

		packetID++
	}
}
```

Also keep `runGetLLMKey` in this file (it's unrelated to audio transport).

**Step 3: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go test ./pkg/pipecatcallhandler/... -run Test_runAsterisk -v`

Expected: PASS

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git add bin-pipecat-manager/pkg/pipecatcallhandler/run.go bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go
git commit -m "NOJIRA-pipecat-manager-websocket-external-media

- bin-pipecat-manager: Remove TCP listener, keep-alive, and Audiosocket read loop
- bin-pipecat-manager: Rewrite audio read to use WebSocket binary frames at 16kHz
- bin-pipecat-manager: Update tests for WebSocket-based audio reading"
```

---

### Task 7: Update Audio Write (runner.go)

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_test.go`

**Step 1: Write tests for updated runnerWebsocketHandleAudio**

```go
func Test_runnerWebsocketHandleAudio(t *testing.T) {
	tests := []struct {
		name string

		se          *pipecatcall.Session
		sampleRate  int
		numChannels int
		data        []byte

		responseDataSamples []byte
		expectWriteErr      error
		expectErr           bool
	}{
		{
			name: "16kHz mono audio - no conversion needed",
			se: &pipecatcall.Session{
				ConnAst: &websocket.Conn{}, // placeholder, write is mocked
				Ctx:     context.Background(),
			},
			sampleRate:  16000,
			numChannels: 1,
			data:        make([]byte, 640),
			expectErr:   false,
		},
		{
			name: "non-16kHz audio - resampled to 16kHz",
			se: &pipecatcall.Session{
				ConnAst: &websocket.Conn{},
				Ctx:     context.Background(),
			},
			sampleRate:          24000,
			numChannels:         1,
			data:                make([]byte, 960),
			responseDataSamples: make([]byte, 640),
			expectErr:           false,
		},
		{
			name: "stereo audio - rejected",
			se: &pipecatcall.Session{
				ConnAst: &websocket.Conn{},
				Ctx:     context.Background(),
			},
			sampleRate:  16000,
			numChannels: 2,
			data:        make([]byte, 1280),
			expectErr:   true,
		},
	}
	// ... test implementation
}
```

**Step 2: Update runnerWebsocketHandleAudio in runner.go**

Replace the Audiosocket write with WebSocket write:

```go
func (h *pipecatcallHandler) runnerWebsocketHandleAudio(se *pipecatcall.Session, sampleRate int, numChannels int, data []byte) error {
	if numChannels != 1 {
		return errors.Errorf("only mono audio is supported. num_channels: %d", numChannels)
	}

	audioData := data
	if sampleRate != defaultMediaSampleRate {
		var err error
		audioData, err = h.audiosocketHandler.GetDataSamples(sampleRate, data)
		if err != nil {
			return errors.Wrapf(err, "could not resample audio data")
		}
	}

	if errWrite := h.websocketAsteriskWrite(se.Ctx, se.ConnAst, audioData, websocketAsteriskFrameSize); errWrite != nil {
		return errors.Wrapf(errWrite, "could not write audio data to asterisk websocket")
	}

	return nil
}
```

**Step 3: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go test ./pkg/pipecatcallhandler/... -run Test_runnerWebsocketHandleAudio -v`

Expected: PASS

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner_test.go
git commit -m "NOJIRA-pipecat-manager-websocket-external-media

- bin-pipecat-manager: Update audio write to use WebSocket binary frames
- bin-pipecat-manager: Skip resampling when Python sends 16kHz audio
- bin-pipecat-manager: Add safety net resampling for non-16kHz audio from Python"
```

---

### Task 8: Clean Up Audiosocket Code

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/audiosocket.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/audiosocket_test.go`

**Step 1: Remove unused Audiosocket functions**

Remove from the interface and implementation:
- `GetStreamingID`
- `GetNextMedia`
- `Upsample8kTo16k`
- `WrapDataPCM16Bit`
- `Write`

Keep only `GetDataSamples` (safety net for resampling).

Updated interface:

```go
type AudiosocketHandler interface {
	GetDataSamples(inputRate int, data []byte) ([]byte, error)
}
```

**Step 2: Remove unused Audiosocket tests**

Remove from `audiosocket_test.go`:
- `Test_audiosocketUpsample8kTo16k`
- `Test_audiosocketWrapDataPCM16Bit`

Keep `Test_audiosocketGetDataSamples`.

**Step 3: Remove unused imports**

Remove `net`, `github.com/CyCoreSystems/audiosocket`, `github.com/gofrs/uuid`, `context` from audiosocket.go if they are no longer used (only `resample` and `fmt` may remain). Also remove `defaultAudiosocketFormatSLIN` and `defaultAudiosocketMaxFragmentSize` constants.

**Step 4: Regenerate mocks**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go generate ./pkg/pipecatcallhandler/...`

**Step 5: Run all tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go test ./pkg/pipecatcallhandler/... -v`

Expected: PASS

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git add bin-pipecat-manager/pkg/pipecatcallhandler/audiosocket.go bin-pipecat-manager/pkg/pipecatcallhandler/audiosocket_test.go bin-pipecat-manager/pkg/pipecatcallhandler/mock_audiosocket.go
git commit -m "NOJIRA-pipecat-manager-websocket-external-media

- bin-pipecat-manager: Remove Audiosocket protocol functions no longer needed
- bin-pipecat-manager: Keep GetDataSamples as safety net for non-16kHz resampling
- bin-pipecat-manager: Remove Audiosocket-specific tests"
```

---

### Task 9: Remove DummyConn Test Helper and Fix Remaining Tests

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/pipecatframe_test.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go`

**Step 1: Remove DummyConn from pipecatframe_test.go**

The `DummyConn` struct implements `net.Conn` for test purposes. Since we no longer use `net.Conn` for Asterisk connections, remove it. If `run_test.go` still references it (for keepalive test), that test should also be removed since `runAsteriskKeepAlive` is gone.

**Step 2: Update run_test.go**

Remove `Test_runKeepAlive` — the Audiosocket keepalive is gone. Add tests for the new `runAsteriskReceivedMediaHandle` if not already done in Task 6.

**Step 3: Run all tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go test ./... -v`

Expected: PASS

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git add bin-pipecat-manager/pkg/pipecatcallhandler/pipecatframe_test.go bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go
git commit -m "NOJIRA-pipecat-manager-websocket-external-media

- bin-pipecat-manager: Remove DummyConn test helper (net.Conn no longer used)
- bin-pipecat-manager: Remove Audiosocket keepalive test"
```

---

### Task 10: Full Verification and Final Commit

**Files:** All changed files

**Step 1: Run full verification workflow**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass with no errors.

**Step 2: Fix any lint or test issues**

Address any remaining compilation errors, unused imports, or lint warnings.

**Step 3: Verify all tests pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media/bin-pipecat-manager && go test ./... -count=1`

Expected: PASS

**Step 4: Review the diff**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media && git diff --stat HEAD~9`

Verify the changes match the design document.

**Step 5: Squash or create final commit if needed**

If there were fix-up commits, consider squashing. Otherwise, push and create PR.

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-pipecat-manager-websocket-external-media
git push -u origin NOJIRA-pipecat-manager-websocket-external-media
```
