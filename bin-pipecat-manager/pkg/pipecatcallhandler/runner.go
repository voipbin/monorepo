package pipecatcallhandler

import (
	"encoding/json"
	"fmt"
	ammessage "monorepo/bin-ai-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

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

	if errStart := h.pythonRunner.Start(
		se.Ctx,
		pc.ID,
		string(pc.LLMType),
		string(se.LLMKey),
		string(pc.STTType),
		string(pc.TTSType),
		pc.TTSVoiceID,
		pc.LLMMessages,
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
			log.Debugf("Received Ping message from client. Sending Pong.")
			h.pipecatframeHandler.SendData(se, websocket.PongMessage, []byte{})
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

	// handle received messages from websocket
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

			case *pipecatframe.Frame_Audio:
				audio := x.Audio
				if errAudio := h.runnerWebsocketHandleAudio(se, int(audio.SampleRate), int(audio.NumChannels), audio.Audio); errAudio != nil {
					return nil
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
			log.Debugf("Received Ping message from client. Sending Pong.")
			if errWrite := h.websocketHandler.WriteMessage(ws, websocket.PongMessage, []byte{}); errWrite != nil {
				log.Errorf("Could not send Pong message: %v", errWrite)
				return nil
			}
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

		id := h.utilHandler.UUIDCreate()
		event := message.Message{
			Identity: commonidentity.Identity{
				ID:         id,
				CustomerID: se.CustomerID,
			},

			PipecatcallID:            se.ID,
			PipecatcallReferenceType: se.PipecatcallReferenceType,
			PipecatcallReferenceID:   se.PipecatcallReferenceID,

			Text: msg.Data.Text,
		}
		h.notifyHandler.PublishEvent(se.Ctx, message.EventTypeBotTranscription, event)

	case pipecatframe.RTVIFrameTypeUserTranscription:
		msg := pipecatframe.RTVIUserTranscriptionMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal user-transcription message")
		}

		if !msg.Data.Final {
			// ignore non-final user transcriptions
			return nil
		}

		id := h.utilHandler.UUIDCreate()
		event := message.Message{
			Identity: commonidentity.Identity{
				ID:         id,
				CustomerID: se.CustomerID,
			},

			PipecatcallID:            se.ID,
			PipecatcallReferenceType: se.PipecatcallReferenceType,
			PipecatcallReferenceID:   se.PipecatcallReferenceID,

			Text: msg.Data.Text,
		}
		h.notifyHandler.PublishEvent(se.Ctx, message.EventTypeUserTranscription, event)

	case pipecatframe.RTVIFrameTypeUserLLMText:
		msg := pipecatframe.RTVIUserLLMTextMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal user-llm-text message")
		}

		id := h.utilHandler.UUIDCreate()
		event := message.Message{
			Identity: commonidentity.Identity{
				ID:         id,
				CustomerID: se.CustomerID,
			},

			PipecatcallID:            se.ID,
			PipecatcallReferenceType: se.PipecatcallReferenceType,
			PipecatcallReferenceID:   se.PipecatcallReferenceID,

			Text: msg.Data.Text,
		}
		h.notifyHandler.PublishEvent(se.Ctx, message.EventTypeUserLLM, event)

	case pipecatframe.RTVIFrameTypeBotLLMText:
		msg := pipecatframe.RTVIBotLLMTextMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal bot-llm-text message")
		}

		se.LLMBotText += msg.Data.Text

	case pipecatframe.RTVIFrameTypeBotLLMStopped:
		msg := pipecatframe.RTVIBotLLMStoppedMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal bot-llm-stopped message")
		}

		id := h.utilHandler.UUIDCreate()
		event := message.Message{
			Identity: commonidentity.Identity{
				ID:         id,
				CustomerID: se.CustomerID,
			},

			PipecatcallID:            se.ID,
			PipecatcallReferenceType: se.PipecatcallReferenceType,
			PipecatcallReferenceID:   se.PipecatcallReferenceID,

			Text: se.LLMBotText,
		}
		h.notifyHandler.PublishEvent(se.Ctx, message.EventTypeBotLLM, event)

		log.Debugf("Cleaning BotLLMStopped message. text: %s", se.LLMBotText)
		se.LLMBotText = ""

	case pipecatframe.RTVIFrameTypeMetrics:
		// we do nothing with this for now

	default:
		log.WithField("frame", frame).Errorf("Unrecognized RTVI message type: %s", frame.Type)
	}

	return nil
}

func (h *pipecatcallHandler) runnerWebsocketHandleAudio(se *pipecatcall.Session, sampleRate int, numChannels int, data []byte) error {
	if numChannels != 1 {
		return errors.Errorf("only mono audio is supported. num_channels: %d", numChannels)
	}

	audioData, err := h.audiosocketHandler.GetDataSamples(sampleRate, data)
	if err != nil {
		return errors.Wrapf(err, "could not get audio data samples")
	}

	if errWrite := h.audiosocketHandler.Write(se.Ctx, se.AsteriskConn, audioData); errWrite != nil {
		return errors.Wrapf(errWrite, "could not write processed audio data to asterisk connection")
	}

	return nil
}
