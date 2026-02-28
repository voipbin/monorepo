package pipecatcallhandler

import (
	"context"
	"fmt"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amteam "monorepo/bin-ai-manager/models/team"
	aitool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	defaultMediaSampleRate = 16000
	defaultMediaNumChannel = 1
)

// resolvedTeamData is the Python-facing team struct sent via HTTP POST to the pipecat runner.
// Each member carries its own EngineKey so the Python side can call LLM APIs directly.
type resolvedTeamData struct {
	ID            uuid.UUID            `json:"id"`
	StartMemberID uuid.UUID            `json:"start_member_id"`
	Members       []resolvedMemberData `json:"members"`
}

// resolvedMemberData holds a single team member's AI config, available tools, and transitions.
type resolvedMemberData struct {
	ID          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	AI          resolvedAIData      `json:"ai"`
	Tools       []aitool.Tool       `json:"tools"`
	Transitions []amteam.Transition `json:"transitions"`
}

// resolvedAIData contains the AI engine configuration for a team member,
// including credentials, model, prompt, and TTS/STT settings.
type resolvedAIData struct {
	EngineModel string         `json:"engine_model"`
	EngineKey   string         `json:"engine_key"`
	InitPrompt  string         `json:"init_prompt"`
	Parameter   map[string]any `json:"parameter,omitempty"`
	TTSType     string         `json:"tts_type"`
	TTSVoiceID  string         `json:"tts_voice_id"`
	STTType     string         `json:"stt_type"`
}

func (h *pipecatcallHandler) runAsteriskReceivedMediaHandle(se *pipecatcall.Session) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runAsteriskReceivedMediaHandle",
		"pipecatcall_id": se.ID,
	})

	if se.ConnAst == nil {
		log.Debugf("No Asterisk WebSocket connection, skipping media handle.")
		return
	}

	// This goroutine is the sole reader on se.ConnAst. gorilla/websocket supports
	// only one concurrent reader, so no other goroutine should read from this
	// connection. Close ConnAstDone on exit to signal the lifecycle monitor.
	if se.ConnAstDone != nil {
		defer close(se.ConnAstDone)
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

func (h *pipecatcallHandler) runGetLLMKey(ctx context.Context, pc *pipecatcall.Pipecatcall) string {
	switch pc.ReferenceType {
	case pipecatcall.ReferenceTypeAICall:
		c, err := h.requestHandler.AIV1AIcallGet(ctx, pc.ReferenceID)
		if err != nil {
			logrus.Errorf("Could not get ai call info. err: %v", err)
			return ""
		}

		a, err := h.resolveAIFromAIcall(ctx, c)
		if err != nil {
			logrus.Errorf("Could not resolve ai info. err: %v", err)
			return ""
		}

		return a.EngineKey

	default:
		return ""
	}
}

// resolveTeamForPython builds the full team data for the Python runner, including engine keys.
// Returns nil if the AIcall is not team-backed.
func (h *pipecatcallHandler) resolveTeamForPython(
	ctx context.Context, c *amaicall.AIcall,
) (*resolvedTeamData, error) {
	if c.AssistanceType != amaicall.AssistanceTypeTeam {
		return nil, nil
	}

	team, err := h.requestHandler.AIV1TeamGet(ctx, c.AssistanceID)
	if err != nil {
		return nil, fmt.Errorf("could not get team: %w", err)
	}
	logrus.WithField("team", team).Debugf("Retrieved team info. team_id: %s", team.ID)

	resolved := &resolvedTeamData{
		ID:            team.ID,
		StartMemberID: team.StartMemberID,
	}

	for _, m := range team.Members {
		ai, errAI := h.requestHandler.AIV1AIGet(ctx, m.AIID)
		if errAI != nil {
			return nil, fmt.Errorf("could not get AI for member %s: %w", m.ID, errAI)
		}
		logrus.WithField("ai", ai).Debugf("Retrieved AI info for member. member_id: %s, ai_id: %s", m.ID, m.AIID)

		tools := h.toolHandler.GetByNames(ai.ToolNames)

		resolved.Members = append(resolved.Members, resolvedMemberData{
			ID:   m.ID,
			Name: m.Name,
			AI: resolvedAIData{
				EngineModel: string(ai.EngineModel),
				EngineKey:   ai.EngineKey,
				InitPrompt:  ai.InitPrompt,
				Parameter:   ai.Parameter,
				TTSType:     string(ai.TTSType),
				TTSVoiceID:  ai.TTSVoiceID,
				STTType:     string(ai.STTType),
			},
			Tools:       tools,
			Transitions: m.Transitions,
		})
	}

	return resolved, nil
}

// resolveAIFromAIcall resolves the AI entity from the AIcall's assistance type and ID.
// For AssistanceTypeAI, AssistanceID is the AI ID directly.
// For AssistanceTypeTeam, it fetches the team, finds the start member, and returns that member's AI.
func (h *pipecatcallHandler) resolveAIFromAIcall(ctx context.Context, c *amaicall.AIcall) (*amai.AI, error) {
	switch c.AssistanceType {
	case amaicall.AssistanceTypeTeam:
		team, err := h.requestHandler.AIV1TeamGet(ctx, c.AssistanceID)
		if err != nil {
			return nil, err
		}

		// find the start member's AI ID
		for _, m := range team.Members {
			if m.ID == team.StartMemberID {
				return h.requestHandler.AIV1AIGet(ctx, m.AIID)
			}
		}
		return nil, fmt.Errorf("could not find start member in team. team_id: %s, start_member_id: %s", c.AssistanceID, team.StartMemberID)

	default:
		// AssistanceTypeAI or any other: AssistanceID is the AI ID
		return h.requestHandler.AIV1AIGet(ctx, c.AssistanceID)
	}
}
