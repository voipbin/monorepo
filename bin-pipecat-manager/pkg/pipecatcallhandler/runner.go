package pipecatcallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	amai "monorepo/bin-ai-manager/models/ai"
	ammessage "monorepo/bin-ai-manager/models/message"
	aitool "monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// StopReason identifies why an LLM-related goroutine exited.
// Stored atomically on Session.LLMStopReason (as int32) and
// read by the flush goroutine to attribute the exit in metrics.
type StopReason int32

const (
	StopReasonUnset StopReason = iota
	StopReasonNormal
	StopReasonIdleWatchdog
	StopReasonTerminateForce
	StopReasonContextCancel
)

// idleWatchdogTimeout is the duration of inactivity (no tokens received) after
// which the flush goroutine self-terminates with StopReasonIdleWatchdog. The
// watchdog only arms after the first token arrives — it never fires on a
// generation that produced zero tokens.
//
// idleWatchdogTickRate is how often the watchdog ticker fires to check elapsed
// idle time. A finer tick rate reduces detection jitter at the cost of slightly
// more select-loop wakeups.
//
// flushFinalizeTimeout is the upper bound flushAndFinalize will wait for the
// flush goroutine to exit after closing LLMStopChan. If the goroutine has not
// published its final event and closed LLMDoneChan within this window,
// flushAndFinalize gives up and lets terminate() proceed to teardown so a
// stuck flush cannot indefinitely block hangup.
//
// These are var (not const) so tests can patch them to short values.
var (
	idleWatchdogTimeout  = 8 * time.Second
	idleWatchdogTickRate = 1 * time.Second
	flushFinalizeTimeout = 3 * time.Second
)

// reasonLabel returns the Prometheus label associated with a StopReason.
// Any value not explicitly mapped (including StopReasonUnset and future
// additions) is returned as "unknown" so metrics never panic on new values.
func reasonLabel(r StopReason) string {
	switch r {
	case StopReasonNormal:
		return "stopped_normal"
	case StopReasonIdleWatchdog:
		return "idle_watchdog"
	case StopReasonTerminateForce:
		return "terminate_force"
	case StopReasonContextCancel:
		return "context_cancelled"
	default:
		return "unknown"
	}
}

func (h *pipecatcallHandler) RunnerStart(pc *pipecatcall.Pipecatcall, se *pipecatcall.Session) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RunnerStart",
		"pipecatcall_id": pc.ID,
	})

	// start python script to run the pipecat runner
	if errScript := h.runnerStartScript(pc, se); errScript != nil {
		log.Errorf("Could not start the pipecat runner script: %v", errScript)
		return
	}
	log.Debugf("Pipecat runner script started.")

	<-se.Ctx.Done()
	log.Debugf("Pipecat runner script finished.")
}

func (h *pipecatcallHandler) runnerStartScript(pc *pipecatcall.Pipecatcall, se *pipecatcall.Session) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"pipecatcall_id": pc.ID,
	})
	log.Debugf("Starting pipecat runner. pipecatcall_id: %s", pc.ID)

	// Get tools and resolve team based on reference type.
	// For AICall references, fetch AIcall once and use for both tool and team resolution.
	// Team-backed calls skip resolveAIFromAIcall since per-member tools come from resolvedTeam.
	var tools []aitool.Tool
	var resolvedTeam *resolvedTeamData
	var vadConfig *amai.VADConfig
	var smartTurnEnabled bool

	if pc.ReferenceType == pipecatcall.ReferenceTypeAICall {
		aicall, err := h.requestHandler.AIV1AIcallGet(se.Ctx, pc.ReferenceID)
		if err != nil {
			return fmt.Errorf("could not get AIcall for pipecatcall %s: %w", pc.ID, err)
		}

		vadConfig = aicall.AIVADConfig
		smartTurnEnabled = aicall.AISmartTurnEnabled

		// Resolve team first — if team-backed, per-member tools come from resolvedTeam
		resolvedTeam, err = h.resolveTeamForPython(se.Ctx, aicall)
		if err != nil {
			return fmt.Errorf("could not resolve team for python: %w", err)
		}

		if resolvedTeam != nil {
			// Team pipeline: per-member tools are in resolvedTeam, no top-level tools needed
			log.WithField("team_id", resolvedTeam.ID).Debugf("Resolved team for python runner")
		} else {
			// Single AI: resolve tools from the AI's configuration
			ai, errAI := h.resolveAIFromAIcall(se.Ctx, aicall)
			if errAI != nil {
				// Fail-open by design (VOIP-1234 §6 v4): this AI lookup failure
				// cannot be scoped to Insight-typed AIs specifically (pipecatcall.ReferenceType
				// only distinguishes ReferenceTypeCall/ReferenceTypeAICall, not AI.Type), and
				// there is no observed incident motivating a fail-closed change that would
				// affect every AICall-backed session (not just Insight). Falling back to
				// GetAll() keeps the session usable at the cost of over-broad tool exposure
				// on this rare error path. Metric + alert-worthy log below give operators
				// visibility so a real incident can be detected and this decision revisited.
				metricsToolResolveFallbackTotal.Inc()
				log.WithFields(logrus.Fields{
					"aicall_id":     aicall.ID,
					"assistance_id": aicall.AssistanceID,
				}).WithError(errAI).Errorf("Could not resolve AI for pipecat session %s; falling back to all tools (fail-open, over-broad tool exposure)", pc.ID)
				tools = h.toolHandler.GetAll()
			} else {
				tools = h.toolHandler.GetByNames(ai.ToolNames)

				// filter out search_knowledge if no RAG is configured
				if ai.RagID == uuid.Nil {
					filtered := make([]aitool.Tool, 0, len(tools))
					for _, t := range tools {
						if t.Name != aitool.ToolNameSearchKnowledge {
							filtered = append(filtered, t)
						}
					}
					tools = filtered
				}
			}
		}
	} else {
		tools = h.toolHandler.GetAll()
	}
	log.WithField("tool_count", len(tools)).Debugf("Retrieved tools for pipecat call")

	if errStart := h.pythonRunner.Start(
		se.Ctx,
		pc.ID,
		string(pc.LLMType),
		string(se.LLMKey),
		pc.LLMMessages,
		string(pc.STTType),
		string(pc.STTLanguage),
		string(pc.TTSType),
		string(pc.TTSLanguage),
		pc.TTSVoiceID,
		tools,
		resolvedTeam,
		vadConfig,
		smartTurnEnabled,
	); errStart != nil {
		return errors.Wrapf(errStart, "could not start python client")
	}
	log.Debugf("Pipecat runner started successfully.")

	return nil
}

func (h *pipecatcallHandler) RunnerWebsocketHandle(id uuid.UUID, c *gin.Context) error {
	direction := c.Query("direction")
	switch direction {
	case "input":
		return h.RunnerWebsocketHandleInput(id, c)

	case "output":
		return h.RunnerWebsocketHandleOutput(id, c)

	default:
		return fmt.Errorf("invalid direction parameter: %s", direction)
	}
}

func (h *pipecatcallHandler) RunnerWebsocketHandleInput(id uuid.UUID, c *gin.Context) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RunnerWebsocketHandleInput",
		"pipecatcall_id": id,
		"remote_addr":    c.Request.RemoteAddr,
		"uri":            c.Request.RequestURI,
		"method":         c.Request.Method,
		"headers":        c.Request.Header,
		"params":         c.Params,
	})

	se, err := h.SessionGet(id)
	if err != nil {
		return fmt.Errorf("could not get pipecatcall session: %w", err)
	}
	log.WithField("session", se).Debugf("Pipecatcall session retrieved. pipecatcall_id: %s", id)

	ws, err := h.websocketHandler.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return fmt.Errorf("could not upgrade to WebSocket: %w", err)
	}
	defer func() {
		log.Debugf("Closing pipecatcall input websocket connection. pipecatcall_id: %s", id)
		_ = ws.Close()
	}()
	log.Debugf("WebSocket connection established with pipecat runner for input direction. pipecatcall_id: %s", id)

	// run input receiver in a separate goroutine
	go h.runnerWebsocketHandleInputReceiver(se, ws)

	// handle sending messages to websocket
	// this will run until the session context is done
	h.pipecatframeHandler.RunSender(se, ws)
	log.Debugf("Pipecatcall input websocket session is done. pipecatcall_id: %s", id)

	return nil
}

// runnerWebsocketHandleInputReceiver handles control messages on the input WebSocket connection.
// This goroutine's responsibility is to keep the input socket healthy by answering WebSocket
// control frames (ping/pong/close). It runs concurrently with the main sender loop.
//
// Clarification about direction and data flow:
//   - This is the INPUT direction — our side is sending the audio stream toward the pipecat app/runner.
//     Because the primary purpose of this socket is to transmit audio, any text or binary messages
//     received from the remote peer are unexpected and are only logged for debugging; they are not
//     processed as application data here. The opposite (output) direction is responsible for receiving
//     and handling streamed audio coming from the pipecat app.
//   - Even though we normally ignore non-control incoming messages on the input socket, we must
//     respond to WebSocket control frames (especially ping) to maintain the connection. That is the
//     reason this receiver exists.
//
// Note: the main audio sending logic runs in RunSender; this function only preserves connection health
// (control-frame handling and logging) for the input connection.
func (h *pipecatcallHandler) runnerWebsocketHandleInputReceiver(se *pipecatcall.Session, ws *websocket.Conn) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runnerWebsocketHandleInputReceiver",
		"pipecatcall_id": se.ID,
	})

	for {
		msgType, message, err := h.websocketHandler.ReadMessage(ws)
		if err != nil {
			log.Errorf("Could not read message from websocket: %v", err)
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			log.WithField("message", message).Debugf("Received Protobuf Frame from client.")
		case websocket.TextMessage:
			log.WithField("message", message).Debugf("Received Text message from client.")
		case websocket.CloseMessage:
			log.Debugf("Received Close message from client.")
			return
		case websocket.PingMessage:
			// Note: gorilla/websocket handles Ping/Pong control frames internally
			// via WriteControl (concurrent-safe with WriteMessage). ReadMessage
			// never returns PingMessage to the caller, so this case is defensive
			// only. Do NOT route Pong through the audio channel (SendData) as that
			// competes with audio frames and can be dropped under backpressure.
			log.Debugf("Received Ping message from client (handled by gorilla internally).")
		case websocket.PongMessage:
			log.Debugf("Received Pong message from client.")
		default:
			log.Debugf("Received unknown message type %d, message: %s", msgType, message)
		}
	}
}

func (h *pipecatcallHandler) RunnerWebsocketHandleOutput(id uuid.UUID, c *gin.Context) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RunnerWebsocketHandleOutput",
		"pipecatcall_id": id,
		"remote_addr":    c.Request.RemoteAddr,
		"uri":            c.Request.RequestURI,
		"method":         c.Request.Method,
		"headers":        c.Request.Header,
		"params":         c.Params,
	})

	se, err := h.SessionGet(id)
	if err != nil {
		return fmt.Errorf("could not get pipecatcall session: %w", err)
	}
	log.WithField("session", se).Debugf("Pipecatcall session retrieved. pipecatcall_id: %s", id)

	ws, err := h.websocketHandler.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return fmt.Errorf("could not upgrade to WebSocket: %w", err)
	}
	log.Debugf("WebSocket connection established with pipecat runner.")
	defer func() {
		log.Debugf("Closing pipecatcall output websocket connection. pipecatcall_id: %s", id)
		_ = ws.Close()
	}()

	pc, err := h.Get(context.Background(), id)
	if err != nil {
		return errors.Wrapf(err, "Could not get pipecatcall info. pipecatcall_id: %s", id)
	}

	// notify the event that the output websocket is connected
	// this is used to start sending audio from asterisk to pipecat runner
	// we consider the pipecatcall initialized when the output websocket is connected
	// because once the output websocket is connected, we are ready to receiving the audio from the pipecat runner.
	log.WithField("pipecatcall", pc).Debugf("Publishing pipecatcall_initialized event. tm_create: %v, tm_update: %v, tm_delete: %v", pc.TMCreate, pc.TMUpdate, pc.TMDelete)
	h.notifyHandler.PublishEvent(se.Ctx, pipecatcall.EventTypeInitialized, pc)
	log.Debugf("Notified that pipecatcall is initialized. pipecatcall_id: %s", id)

	// handle received messages from websocket
	var audioErrors int
	for {
		msgType, message, err := h.websocketHandler.ReadMessage(ws)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Debugf("Client disconnected gracefully.")
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("Client disconnected unexpectedly: %v", err)
			} else {
				log.Errorf("Error reading message from client: %v", err)
			}
			break
		}

		switch msgType {
		case websocket.BinaryMessage:
			var frame pipecatframe.Frame
			if errUnmarshal := proto.Unmarshal(message, &frame); errUnmarshal != nil {
				log.Errorf("Could not unmarshal Protobuf message: %v", errUnmarshal)
				continue
			}

			switch x := frame.Frame.(type) {
			case *pipecatframe.Frame_Text:
				log.Debugf("Received TextFrame: ID=%d, Name=%s, Text='%s'", x.Text.Id, x.Text.Name, x.Text.Text)
				h.runnerHandleTextFrame(se, x.Text.Text)

			case *pipecatframe.Frame_Audio:
				audio := x.Audio
				if errAudio := h.runnerWebsocketHandleAudio(se, int(audio.SampleRate), int(audio.NumChannels), audio.Audio); errAudio != nil {
					// Log and continue instead of terminating the output handler.
					// A single transient write error should not kill all TTS audio
					// for the remainder of the call. The Asterisk WebSocket lifecycle
					// monitor (ConnAstDone) handles true disconnects.
					audioErrors++
					if audioErrors == 1 || audioErrors%100 == 0 {
						log.Errorf("Could not handle audio frame, skipping. consecutive_errors: %d, err: %v", audioErrors, errAudio)
					}
					if audioErrors > 500 {
						log.Errorf("Too many consecutive audio write errors (%d), stopping output handler.", audioErrors)
						return nil
					}
				} else {
					audioErrors = 0
				}

			case *pipecatframe.Frame_Transcription:
				log.Debugf("Received TranscriptionFrame: ID=%d, Name=%s, Text='%s', UserID=%s, Timestamp=%s", x.Transcription.Id, x.Transcription.Name, x.Transcription.Text, x.Transcription.UserId, x.Transcription.Timestamp)

			case *pipecatframe.Frame_Message:
				if errMessage := h.receiveMessageFrameTypeMessage(se, []byte(x.Message.Data)); errMessage != nil {
					log.Errorf("Could not process MessageFrame: %v", errMessage)
				}

			default:
				log.Errorf("Could not recognize the Protobuf Frame type. type: %T", x)
			}

		case websocket.TextMessage:
			// because of we switched to Protobuf communication, text messages are only handled in exceptional cases.
			log.Errorf("Could not recognize the message type. type: %d (Expecting Protobuf Binary)", msgType)
		case websocket.CloseMessage:
			log.Debugf("Received Close message from client.")
			return nil
		case websocket.PingMessage:
			// Note: gorilla/websocket handles Ping/Pong control frames internally
			// via WriteControl (concurrent-safe with WriteMessage). ReadMessage
			// never returns PingMessage to the caller, so this case is defensive
			// only. Do NOT route Pong through the audio channel (SendData) as that
			// competes with audio frames and can be dropped under backpressure.
			log.Debugf("Received Ping message from client (handled by gorilla internally).")
		case websocket.PongMessage:
			log.Debugf("Received Pong message from client.")
		default:
			log.Debugf("Received unknown message type %d", msgType)
		}
	}

	log.Debugf("Pipecatcall output websocket session is done. pipecatcall_id: %s", id)
	return nil
}

func (h *pipecatcallHandler) RunnerToolHandle(id uuid.UUID, c *gin.Context) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "RunnerToolHandle",
	})
	ctx := c.Request.Context()

	// get pipecatcall session
	se, err := h.SessionGet(id)
	if err != nil {
		return fmt.Errorf("could not get pipecatcall session: %w", err)
	}
	log.WithField("session", se).Debugf("Pipecatcall session retrieved. pipecatcall_id: %s", id)

	pc, err := h.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("could not get pipecatcall: %w", err)
	}
	log.WithField("pipecatcall", pc).Debugf("Pipecatcall retrieved. pipecatcall_id: %s", id)

	// send request to the ai-manager to get tool information
	if pc.ReferenceType != pipecatcall.ReferenceTypeAICall {
		return fmt.Errorf("pipecatcall reference type is not ai-call. reference_type: %s", pc.ReferenceType)
	}

	request := struct {
		ID       string                 `json:"id"`
		Type     string                 `json:"type"`
		Function ammessage.FunctionCall `json:"function"`
	}{}

	if errBind := c.BindJSON(&request); errBind != nil {
		return fmt.Errorf("could not bind tool request JSON: %w", errBind)
	}

	res, err := h.requestHandler.AIV1AIcallToolExecute(ctx, pc.ReferenceID, request.ID, ammessage.ToolType(request.Type), &request.Function)
	if err != nil {
		return fmt.Errorf("could not execute tool via ai-manager: %w", err)
	}

	c.JSON(http.StatusOK, res)

	return nil
}

func (h *pipecatcallHandler) RunnerMemberSwitchedHandle(id uuid.UUID, c *gin.Context) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RunnerMemberSwitchedHandle",
		"pipecatcall_id": id,
	})
	ctx := c.Request.Context()

	se, err := h.SessionGet(id)
	if err != nil {
		return fmt.Errorf("could not get pipecatcall session: %w", err)
	}
	log.WithField("session", se).Debugf("Pipecatcall session retrieved. pipecatcall_id: %s", id)

	pc, err := h.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("could not get pipecatcall: %w", err)
	}
	log.WithField("pipecatcall", pc).Debugf("Pipecatcall retrieved. pipecatcall_id: %s", id)

	if pc.ReferenceType != pipecatcall.ReferenceTypeAICall {
		return fmt.Errorf("pipecatcall reference type is not ai-call. reference_type: %s", pc.ReferenceType)
	}

	request := struct {
		TransitionFunctionName string             `json:"transition_function_name"`
		FromMember             message.MemberInfo `json:"from_member"`
		ToMember               message.MemberInfo `json:"to_member"`
	}{}
	if errBind := c.BindJSON(&request); errBind != nil {
		return fmt.Errorf("could not bind member-switched request JSON: %w", errBind)
	}

	evt := message.MemberSwitchedEvent{
		CustomerID:               pc.CustomerID,
		PipecatcallID:            pc.ID,
		PipecatcallReferenceType: pc.ReferenceType,
		PipecatcallReferenceID:   pc.ReferenceID,
		ActiveflowID:             pc.ActiveflowID,
		TransitionFunctionName:   request.TransitionFunctionName,
		FromMember:               request.FromMember,
		ToMember:                 request.ToMember,
	}

	h.notifyHandler.PublishEvent(ctx, message.EventTypeTeamMemberSwitched, evt)
	log.WithField("event", evt).Debugf("Published team member switched event.")

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
	return nil
}

// newMessageEvent creates a message.Message populated from the session and the given text.
func (h *pipecatcallHandler) newMessageEvent(se *pipecatcall.Session, text string) message.Message {
	return message.Message{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: se.CustomerID,
		},

		PipecatcallID:            se.ID,
		PipecatcallReferenceType: se.PipecatcallReferenceType,
		PipecatcallReferenceID:   se.PipecatcallReferenceID,
		ActiveflowID:             se.ActiveflowID,

		Text: text,
	}
}

// receiveMessageFrameTypeMessage handles RTVI message frames from the pipecat runner.
//
// All PublishEvent calls are dispatched in goroutines so that RabbitMQ publish
// latency does not stall the WebSocket read loop (which also ingests audio frames).
func (h *pipecatcallHandler) receiveMessageFrameTypeMessage(se *pipecatcall.Session, m []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "receiveMessageFrameMessage",
		"pipecatcall_id": se.ID,
	})

	frame := pipecatframe.CommonFrameMessage{}
	if errUnmarshal := json.Unmarshal(m, &frame); errUnmarshal != nil {
		log.Errorf("Error unmarshaling JSON message: %v", errUnmarshal)
		return errUnmarshal
	}

	if frame.Label != pipecatframe.RTVIMessageLabel {
		// other message types can be handled here
		log.Errorf("Unrecognized message label: %s", frame.Label)
		return nil
	}

	switch frame.Type {
	case pipecatframe.RTVIFrameTypeBotTranscription:
		msg := pipecatframe.RTVIBotTranscriptionMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal bot-transcription message")
		}

		go h.notifyHandler.PublishEvent(se.Ctx, message.EventTypeBotTranscription, h.newMessageEvent(se, msg.Data.Text))

	case pipecatframe.RTVIFrameTypeUserTranscription:
		msg := pipecatframe.RTVIUserTranscriptionMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal user-transcription message")
		}

		if !msg.Data.Final {
			// ignore non-final user transcriptions
			return nil
		}

		go h.notifyHandler.PublishEvent(se.Ctx, message.EventTypeUserTranscription, h.newMessageEvent(se, msg.Data.Text))

	case pipecatframe.RTVIFrameTypeUserLLMText:
		msg := pipecatframe.RTVIUserLLMTextMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal user-llm-text message")
		}

		go h.notifyHandler.PublishEvent(se.Ctx, message.EventTypeUserLLM, h.newMessageEvent(se, msg.Data.Text))

	case pipecatframe.RTVIFrameTypeBotLLMText:
		msg := pipecatframe.RTVIBotLLMTextMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal bot-llm-text message")
		}

		if !se.LLMFlushing.Load() {
			se.LLMMessageID = h.utilHandler.UUIDCreate()
			// Snapshot the pending in-reply-to correlation at generation start
			// (VOIP-1234 §4-1). The pending value may be overwritten by a
			// subsequent SendMessage before this generation finishes; the
			// snapshot ensures every event this generation emits reports the
			// message that actually triggered it. Goes through Session's
			// exported getter (not a direct field read) because SendMessage
			// writes this value from the RPC worker pool goroutine while this
			// read loop reads it concurrently.
			se.LLMInReplyToMessageID = se.SnapshotPendingInReplyToMessageID()
			se.LLMTokenChan = make(chan string, 64)
			se.LLMStopChan = make(chan struct{})
			se.LLMDoneChan = make(chan struct{})
			// Reset per-generation primitives BEFORE arming the flush flag so the
			// new generation's machinery is in place before the goroutine launches.
			// Without this reset, a second generation's LLMFlushOnce.Do(close) is a
			// no-op (Once already fired in gen 1) — LLMStopChan never closes, the
			// flush goroutine blocks forever, and the per-generation final event
			// is never published. Likewise, LLMStopReason still holds gen 1's
			// value, so the metric label is wrong for gen 2+.
			se.LLMFlushOnce = sync.Once{}
			se.LLMStopReason.Store(int32(StopReasonUnset))
			se.LLMFlushing.Store(true)
			go h.runLLMIntermediateFlush(se, se.LLMMessageID)
		}

		// Non-blocking send to avoid stalling the WebSocket read loop (which also
		// handles audio frames) if the flush goroutine is slow due to RabbitMQ
		// backpressure. Dropped tokens will be missing from both intermediate and
		// final events. In practice, the 64-token buffer drains every 200ms (~8
		// tokens/tick), so drops only occur if PublishEvent blocks for >1.6s —
		// extremely unlikely since it uses its own context.Background() with timeout.
		select {
		case se.LLMTokenChan <- msg.Data.Text:
		default:
			log.Warnf("LLM token channel full, dropping intermediate token. pipecatcall_id: %s", se.ID)
		}

	case pipecatframe.RTVIFrameTypeBotLLMStopped:
		msg := pipecatframe.RTVIBotLLMStoppedMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal bot-llm-stopped message")
		}

		if !se.LLMFlushing.Load() {
			log.Debugf("BotLLMStopped received but no tokens were received for this generation.")
			break
		}

		// Attribute this stop to a normal completion (CAS so a competing closer
		// — watchdog, terminate, context cancel — that already wrote a more
		// specific reason wins). sync.Once guarantees close(LLMStopChan) runs
		// at most once across all closers; the goroutine's defer in Task 1.7
		// will reset LLMFlushing to false after it exits.
		se.LLMStopReason.CompareAndSwap(int32(StopReasonUnset), int32(StopReasonNormal))
		se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })

		// Bound the wait so a slow RabbitMQ publish in the flush goroutine
		// cannot stall the WebSocket read loop (which also handles audio
		// frames). Reuses flushFinalizeTimeout so all "wait for flush
		// goroutine to exit" paths share the same upper bound.
		timer := time.NewTimer(flushFinalizeTimeout)
		select {
		case <-se.LLMDoneChan:
		case <-timer.C:
			log.Warnf("BotLLMStopped: timed out waiting for flush goroutine after %s", flushFinalizeTimeout)
		}
		timer.Stop()

	default:
		log.WithField("frame", frame).Debugf("Unrecognized RTVI message type: %s", frame.Type)
	}

	return nil
}

func (h *pipecatcallHandler) runnerHandleTextFrame(se *pipecatcall.Session, text string) {
	if text != "FLUSH_MEDIA" {
		return
	}

	if se.ConnAst == nil {
		logrus.Warnf("Cannot send FLUSH_MEDIA: Asterisk WebSocket not connected")
		return
	}

	if err := h.websocketHandler.WriteMessage(se.ConnAst, websocket.TextMessage, []byte("FLUSH_MEDIA")); err != nil {
		logrus.Errorf("Could not send FLUSH_MEDIA to Asterisk: %v", err)
		return
	}

	logrus.Debugf("Sent FLUSH_MEDIA to Asterisk to flush audio buffer")
}

// runLLMIntermediateFlush is the per-generation flush goroutine.
// It owns all accumulated text state locally and periodically publishes
// intermediate events with the delta text received since the last tick.
// It runs until LLMStopChan is closed (by the BotLLMStopped handler).
func (h *pipecatcallHandler) runLLMIntermediateFlush(se *pipecatcall.Session, messageID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runLLMIntermediateFlush",
		"pipecatcall_id": se.ID,
		"message_id":     messageID,
	})

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	watchdog := time.NewTicker(idleWatchdogTickRate)
	defer watchdog.Stop()
	defer close(se.LLMDoneChan)       // close-broadcasts goroutine exit to readers
	defer se.LLMFlushing.Store(false) // LIFO: runs before close(LLMDoneChan), so observers see flushing=false

	var fullText string
	var deltaBuffer string
	var sequence int
	var lastToken time.Time // zero until first token arrives — watchdog uses IsZero() guard

	for {
		select {
		case token := <-se.LLMTokenChan:
			fullText += token
			deltaBuffer += token
			lastToken = time.Now()

		case <-ticker.C:
			if deltaBuffer != "" {
				sequence++
				// Use context.Background() so a cancelled session ctx (from
				// terminate path) does not drop the partial reply event on its
				// way to ai-manager.
				h.publishIntermediateEvent(context.Background(), se, messageID, deltaBuffer, sequence)
				log.Debugf("Published intermediate event. sequence: %d, delta_len: %d", sequence, len(deltaBuffer))
				deltaBuffer = ""
			}

		case now := <-watchdog.C:
			// Watchdog only arms after the first token has arrived. A generation
			// with zero tokens (Python upstream stalled before producing any
			// output) is the responsibility of the terminate / context-cancel
			// paths, not the watchdog.
			if !lastToken.IsZero() && now.Sub(lastToken) >= idleWatchdogTimeout {
				if se.LLMStopReason.CompareAndSwap(int32(StopReasonUnset), int32(StopReasonIdleWatchdog)) {
					metricsIdleWatchdogFired.Inc()
				}
				se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })
			}

		case <-se.LLMStopChan:
			// Drain remaining tokens from the channel.
			for {
				select {
				case token := <-se.LLMTokenChan:
					fullText += token
					deltaBuffer += token
				default:
					goto drained
				}
			}
		drained:
			// Flush any remaining delta as the last intermediate event.
			if deltaBuffer != "" {
				sequence++
				h.publishIntermediateEvent(context.Background(), se, messageID, deltaBuffer, sequence)
				log.Debugf("Published final intermediate event. sequence: %d, delta_len: %d", sequence, len(deltaBuffer))
			}

			// Publish the final complete bot LLM event. Use context.Background()
			// because terminate() may have cancelled se.Ctx and we still want
			// the partial reply to reach ai-manager.
			h.publishFinalBotLLMEvent(context.Background(), se, messageID, fullText)
			log.Debugf("Published final bot LLM event. full_text_len: %d", len(fullText))
			metricsLLMFlushExit.WithLabelValues(reasonLabel(StopReason(se.LLMStopReason.Load()))).Inc()
			return

		case <-se.Ctx.Done():
			// CAS in StopReasonContextCancel only if no other closer (watchdog,
			// terminate, normal stop) already attributed the exit. The reason
			// label on metricsLLMFlushExit reflects whichever reason won.
			se.LLMStopReason.CompareAndSwap(int32(StopReasonUnset), int32(StopReasonContextCancel))

			// Drain and publish final event to preserve partial LLM response in
			// conversation history (used by summary handler). Use context.Background()
			// because the original context is already cancelled.
			for {
				select {
				case token := <-se.LLMTokenChan:
					fullText += token
				default:
					goto ctxDrained
				}
			}
		ctxDrained:
			if fullText != "" {
				h.publishFinalBotLLMEvent(context.Background(), se, messageID, fullText)
				log.Debugf("Context cancelled, published partial final bot LLM event. full_text_len: %d", len(fullText))
			} else {
				log.Debugf("Context cancelled, no text accumulated.")
			}
			metricsLLMFlushExit.WithLabelValues(reasonLabel(StopReason(se.LLMStopReason.Load()))).Inc()
			return
		}
	}
}

// flushAndFinalize is the synchronous helper that terminate() calls to force
// the per-generation flush goroutine to publish its final message_bot_llm
// event before SessionStop tears down the session. It is safe to call when no
// flush is in progress (it returns immediately as a no-op) and safe to call
// concurrently with the flush goroutine's normal completion path because the
// CompareAndSwap on LLMStopReason and sync.Once around close(LLMStopChan)
// preserve the first-writer-wins invariant.
//
// Outcomes (recorded on metricsFlushFinalizeOutcome):
//   - "noop_never_started": no flush goroutine ever ran for this pipecatcall
//     (no BotLLMText was received).
//   - "noop_already_done": the flush goroutine ran and exited cleanly before
//     terminate() arrived (e.g. BotLLMStopped was received first).
//   - "done": the flush goroutine was running; we closed LLMStopChan and it
//     drained, published its final event, and exited within flushFinalizeTimeout.
//   - "timeout": the flush goroutine was running but did not exit within
//     flushFinalizeTimeout, so we abandon the wait and let terminate() proceed.
//     Partial replies may be lost in this rare path; the metric makes that visible.
func (h *pipecatcallHandler) flushAndFinalize(se *pipecatcall.Session) {
	if !se.LLMFlushing.Load() {
		// Distinguish never-started (no generation ever) from already-done
		// (a generation completed cleanly) using LLMMessageID — set when the
		// flush goroutine is armed and not cleared on normal exit.
		if se.LLMMessageID == uuid.Nil {
			metricsFlushFinalizeOutcome.WithLabelValues("noop_never_started").Inc()
		} else {
			metricsFlushFinalizeOutcome.WithLabelValues("noop_already_done").Inc()
		}
		return
	}

	// Attribute this exit to the terminate path (CAS so a concurrent watchdog
	// or normal-stop closer that already wrote a more specific reason wins),
	// then close LLMStopChan exactly once across all closers.
	se.LLMStopReason.CompareAndSwap(int32(StopReasonUnset), int32(StopReasonTerminateForce))
	se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })

	timer := time.NewTimer(flushFinalizeTimeout)
	defer timer.Stop()

	select {
	case <-se.LLMDoneChan:
		metricsFlushFinalizeOutcome.WithLabelValues("done").Inc()
	case <-timer.C:
		metricsFlushFinalizeOutcome.WithLabelValues("timeout").Inc()
	}
}

// publishIntermediateEvent publishes a message_bot_llm_intermediate event with the delta text.
// Accepts an explicit context so callers can use context.Background() when se.Ctx is cancelled
// (e.g. during the terminate path) and the partial reply must still reach ai-manager.
func (h *pipecatcallHandler) publishIntermediateEvent(ctx context.Context, se *pipecatcall.Session, messageID uuid.UUID, delta string, sequence int) {
	evt := message.Message{
		Identity: commonidentity.Identity{
			ID:         messageID,
			CustomerID: se.CustomerID,
		},

		PipecatcallID:            se.ID,
		PipecatcallReferenceType: se.PipecatcallReferenceType,
		PipecatcallReferenceID:   se.PipecatcallReferenceID,
		ActiveflowID:             se.ActiveflowID,
		InReplyToMessageID:       se.LLMInReplyToMessageID,

		Text:     delta,
		Sequence: sequence,
	}

	h.notifyHandler.PublishEvent(ctx, message.EventTypeBotLLMIntermediate, evt)
}

// publishFinalBotLLMEvent publishes the final message_bot_llm event with the complete text.
// Accepts an explicit context so callers can use context.Background() when se.Ctx is cancelled.
func (h *pipecatcallHandler) publishFinalBotLLMEvent(ctx context.Context, se *pipecatcall.Session, messageID uuid.UUID, fullText string) {
	evt := message.Message{
		Identity: commonidentity.Identity{
			ID:         messageID,
			CustomerID: se.CustomerID,
		},

		PipecatcallID:            se.ID,
		PipecatcallReferenceType: se.PipecatcallReferenceType,
		PipecatcallReferenceID:   se.PipecatcallReferenceID,
		ActiveflowID:             se.ActiveflowID,
		InReplyToMessageID:       se.LLMInReplyToMessageID,

		Text: fullText,
	}

	h.notifyHandler.PublishEvent(ctx, message.EventTypeBotLLM, evt)
}

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

	if len(audioData) == 0 {
		return nil
	}

	select {
	case <-se.ConnAstReady:
	case <-se.Ctx.Done():
		return nil
	}

	if se.ConnAst == nil {
		return nil
	}

	if err := h.websocketHandler.WriteMessage(se.ConnAst, websocket.BinaryMessage, audioData); err != nil {
		return errors.Wrapf(err, "could not write audio data to asterisk websocket")
	}

	return nil
}
