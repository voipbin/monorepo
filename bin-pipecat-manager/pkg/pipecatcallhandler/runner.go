package pipecatcallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *pipecatcallHandler) RunnerStart(ctx context.Context, pc *pipecatcall.Pipecatcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RunnerStart",
		"pipecatcall_id": pc.ID,
	})

	// start websocket server for pipecat runner to connect
	if errWebsocket := h.runnerStartWebsocket(ctx, pc); errWebsocket != nil {
		log.Errorf("Could not start the websocket server for pipecat runner: %v", errWebsocket)
		return
	}
	log.Debugf("WebSocket server started. port %d", pc.RunnerPort)

	// start python script to run the pipecat runner
	if errScript := h.runnerStartScript(ctx, pc); errScript != nil {
		log.Errorf("Could not start the pipecat runner script: %v", errScript)
		return
	}
	log.Debugf("Pipecat runner script started.")

	// wait
	if errWait := pc.RunnerCMD.Wait(); errWait != nil {
		log.Errorf("Could not wait for the pipecat runner script to finish: %v", errWait)
	}
	log.Debugf("Pipecat runner script finished.")
}

func (h *pipecatcallHandler) runnerStartWebsocket(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runnerStartWebsocket",
	})

	app := http.NewServeMux()
	app.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		h.runnerWebsocketHandle(ctx, w, r, pc)
	})

	listener, err := net.Listen("tcp", defaultRunnerWebsocketListenAddress)
	if err != nil {
		log.Errorf("Failed to listen on ephemeral port: %v", err)
		return errors.Wrapf(err, "failed to listen on ephemeral port")
	}

	server := &http.Server{
		Handler: app,
	}
	h.setRunnerInfo(pc, listener, server)

	go func() {
		log.Debugf("Starting HTTP server on %s", listener.Addr().String())
		if errServe := server.Serve(listener); errServe != nil && errServe != http.ErrServerClosed {
			log.Errorf("Could not start HTTP server: %v", errServe)
		}
		log.Debugf("HTTP server stopped")
	}()

	return nil
}

func (h *pipecatcallHandler) runnerStartScript(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"pipecatcall_id": pc.ID,
	})
	log.Debugf("Starting pipecat runner. pipecatcall_id: %s", pc.ID)

	filepath, err := h.runnerCreateMessageFile(pc.Messages)
	if err != nil {
		return errors.Wrapf(err, "could not create message file")
	}
	log.Debugf("Message file created at: %s", filepath)

	url := h.runnerGetURL(pc)
	log.Debugf("Pipecat WebSocket server URL: %s", url)

	if errPython := h.runnerStartPython(pc, filepath, url); errPython != nil {
		log.Errorf("Error starting Pipecat Python runner: %v", errPython)
		return errors.Wrapf(errPython, "could not start the pipecat python runner")
	}

	log.Debugf("Pipecat runner started successfully.")
	return nil
}

func (h *pipecatcallHandler) runnerWebsocketHandle(ctx context.Context, w http.ResponseWriter, r *http.Request, pc *pipecatcall.Pipecatcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runnerWebsocketHandle",
		"pipecatcall_id": pc.ID,
	})

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not upgrade to WebSocket: %v", err)
		return
	}
	h.setRunnerWebsocket(pc, ws)

	for {
		msgType, message, err := ws.ReadMessage()
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

				// responseFrame := &pipecatframe.Frame{
				// 	Frame: &pipecatframe.Frame_Text{
				// 		Text: &pipecatframe.TextFrame{
				// 			Id:   x.Text.Id + 1, // ID를 증가시키는 예시
				// 			Name: "GoServerResponse",
				// 			Text: fmt.Sprintf("Go server received your text: '%s'", x.Text.Text),
				// 		},
				// 	},
				// }
				// if errSend := h.sendProtobufFrame(ws, responseFrame); errSend != nil {
				// 	log.Errorf("Could not send the frame.")
				// }

			case *pipecatframe.Frame_Audio:
				audio := x.Audio
				if errAudio := h.runnerWebsocketHandleAudio(ctx, pc, int(audio.SampleRate), int(audio.NumChannels), audio.Audio); errAudio != nil {
					return
				}

			case *pipecatframe.Frame_Transcription:
				log.Debugf("Received TranscriptionFrame: ID=%d, Name=%s, Text='%s', UserID=%s, Timestamp=%s", x.Transcription.Id, x.Transcription.Name, x.Transcription.Text, x.Transcription.UserId, x.Transcription.Timestamp)
				// responseFrame := &pipecatframe.Frame{
				// 	Frame: &pipecatframe.Frame_Transcription{
				// 		Transcription: &pipecatframe.TranscriptionFrame{
				// 			Id:        x.Transcription.Id + 1,
				// 			Name:      "GoServerTranscriptionResponse",
				// 			Text:      fmt.Sprintf("Go server heard: '%s'", x.Transcription.Text),
				// 			UserId:    x.Transcription.UserId,
				// 			Timestamp: time.Now().Format(time.RFC3339),
				// 		},
				// 	},
				// }
				// if errSend := h.sendProtobufFrame(ws, responseFrame); errSend != nil {
				// 	log.Errorf("Could not send the frame.")
				// }

			case *pipecatframe.Frame_Message:
				log.Debugf("Received MessageFrame: Data='%s'", x.Message.Data)
				if errMessage := h.receiveMessageFrameMessage([]byte(x.Message.Data)); errMessage != nil {
					log.Errorf("Could not process MessageFrame: %v", errMessage)
				}
				// MessageFrame에 대한 응답 (예시)
				// responseFrame := &pipecatframe.Frame{
				// 	Frame: &pipecatframe.Frame_Message{
				// 		Message: &pipecatframe.MessageFrame{
				// 			Data: fmt.Sprintf("Go server received generic message: '%s'", x.Message.Data),
				// 		},
				// 	},
				// }
				// sendProtobufFrame(ws, responseFrame, clientAddr)

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
			if errWrite := ws.WriteMessage(websocket.PongMessage, []byte{}); errWrite != nil {
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

func (h *pipecatcallHandler) sendProtobufFrame(ws *websocket.Conn, frame *pipecatframe.Frame) error {
	marshaledFrame, err := proto.Marshal(frame)
	if err != nil {
		return errors.Wrapf(err, "could not marshaling the protobuf frame")
	}
	if err := ws.WriteMessage(websocket.BinaryMessage, marshaledFrame); err != nil {
		return errors.Wrapf(err, "could not write message")
	}

	return nil
}

func (h *pipecatcallHandler) receiveMessageFrameMessage(message []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "receiveMessageFrameMessage",
	})

	frame := pipecatframe.CommonFrameMessage{}
	if errUnmarshal := json.Unmarshal(message, &frame); errUnmarshal != nil {
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
	default:
		log.Errorf("Unrecognized RTVI message type: %s", frame.Type)
	}

	return nil
}

func (h *pipecatcallHandler) runnerWebsocketHandleAudio(ctx context.Context, pc *pipecatcall.Pipecatcall, sampleRate int, numChannels int, data []byte) error {
	if numChannels != 1 {
		return errors.Errorf("only mono audio is supported. num_channels: %d", numChannels)
	}

	audioData, err := audiosocketGetDataSamples(sampleRate, data)
	if err != nil {
		return errors.Wrapf(err, "could not get audio data samples")
	}

	if errWrite := audiosocketWrite(ctx, pc.AsteriskConn, audioData); errWrite != nil {
		return errors.Wrapf(errWrite, "could not write processed audio data to asterisk connection")
	}

	return nil
}

func (h *pipecatcallHandler) runnerCreateMessageFile(messages []map[string]any) (string, error) {

	data, err := json.Marshal(messages)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal messages to JSON")
	}

	// Write the JSON data to the file
	fileName := filepath.Join("/tmp", fmt.Sprintf("%s.json", h.utilHandler.UUIDCreate()))
	err = os.WriteFile(fileName, data, 0644) // 0644 are the file permissions
	if err != nil {
		return "", errors.Wrapf(err, "could not write messages to file: %s", fileName)
	}

	return fileName, nil
}

func (h *pipecatcallHandler) runnerGetURL(pc *pipecatcall.Pipecatcall) string {
	return fmt.Sprintf("ws://localhost:%d/ws", pc.RunnerPort)
}

func (h *pipecatcallHandler) runnerStartPython(pc *pipecatcall.Pipecatcall, message_file string, url string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runnerStartPython",
	})

	pythonInterpreter := "python"
	pythonScript := "/app/scripts/pipecat/main.py"

	args := []string{
		"--ws_server_url", url,
		"--llm", string(pc.LLM),
		"--stt", string(pc.STT),
		"--tts", string(pc.TTS),
		"--messages_file", message_file,
	}

	cmdArgs := append([]string{pythonScript}, args...)
	cmd, err := h.pythonRunner.Start(pythonInterpreter, cmdArgs)
	if err != nil {
		return errors.Wrapf(err, "could not start python client")
	}
	log.Debugf("Started Python script with PID %d", cmd.Process.Pid)

	h.setRunnerCMD(pc, cmd)
	return nil
}
