package pipecatcallhandler

import (
	"encoding/json"
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

func (h *pipecatcallHandler) RunnerWebsocketHandle(id uuid.UUID, c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "RunnerWebsocketHandle",
		"id":   id,
	})

	ws, err := h.websocketHandler.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("Could not upgrade to WebSocket: %v", err)
		return
	}
	log.Debugf("WebSocket connection established with pipecat runner.")

	se, err := h.SessionGet(id)
	if err != nil {
		log.Errorf("Could not get pipecatcall session: %v", err)
		return
	}
	log.WithField("session", se).Debugf("Pipecatcall session retrieved. pipecatcall_id: %s", id)

	h.SessionsetRunnerWebsocket(se, ws)
	go h.pipecatframeHandler.RunSender(se)

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
					return
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
			return
		case websocket.PingMessage:
			log.Debugf("Received Ping message from client. Sending Pong.")
			if errWrite := h.websocketHandler.WriteMessage(ws, websocket.PongMessage, []byte{}); errWrite != nil {
				log.Errorf("Could not send Pong message: %v", errWrite)
				return
			}
		case websocket.PongMessage:
			log.Debugf("Received Pong message from client.")
		default:
			log.Debugf("Received unknown message type %d", msgType)
		}
	}
}

func (h *pipecatcallHandler) RunnerToolHandle(id uuid.UUID, c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "RunnerToolHandle",
	})
	ctx := c.Request.Context()

	pc, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get pipecatcall: %v", err)
	}
	log.WithField("pipecatcall", pc).Debugf("Pipecatcall retrieved. pipecatcall_id: %s", id)

	// send request to the ai-manager to get tool information
	if pc.ReferenceType != pipecatcall.ReferenceTypeAICall {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// get pipecatcall session
	se, err := h.SessionGet(id)
	if err != nil {
		log.Errorf("Could not get pipecatcall session: %v", err)
		return
	}
	log.WithField("session", se).Debugf("Pipecatcall session retrieved. pipecatcall_id: %s", id)

	// note
	// todo: need to implement tool handling logic here
	// for now, just logging the request
	// send the request to the ai-manager later.
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
	log.Debugf("Received message frame: Label=%s, Type=%s", frame.Label, frame.Type)

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

		se.LLMBotText = ""

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
