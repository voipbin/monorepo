package pipecatcallhandler

import (
	"context"
	"fmt"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
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
