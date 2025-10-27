package pipecatcall

import (
	"monorepo/bin-common-handler/models/identity"
	"net"
	"net/http"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

type Pipecatcall struct {
	identity.Identity

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	LLM      LLM              `json:"-"`
	STT      STT              `json:"-"`
	TTS      TTS              `json:"-"`
	VoiceID  string           `json:"-"`
	Messages []map[string]any `json:"-"`

	// pipecat runner
	RunnerListener    net.Listener    `json:"-"`
	RunnerPort        int             `json:"-"`
	RunnerServer      *http.Server    `json:"-"`
	RunnerWebsocket   *websocket.Conn `json:"-"`
	RunnerWebsocketMu sync.Mutex      `json:"-"`

	// asterisk info
	AsteriskStreamingID uuid.UUID `json:"-"`
	AsteriskConn        net.Conn  `json:"-"`
}

type ReferenceType string

const (
	ReferenceTypeCall   ReferenceType = "call"
	ReferenceTypeAICall ReferenceType = "ai_call"
)

// LLM
// consist of (vendor) + . + (model)
// e.g. openai.gpt-4, anthropic.claude-2
type LLM string

type STT string

const (
	STTDeepgram = "deepgram"
)

type TTS string

const (
	TTSCartesia   = "cartesia"
	TTSElevenLabs = "elevenlabs"
)
