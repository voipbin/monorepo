package main

// type pipecatHandler struct {
// 	llm         string
// 	stt         string
// 	tts         string
// 	messageFile string

// 	// websocket
// 	listener net.Listener
// 	port     int
// 	server   *http.Server

// 	// python
// 	cmd *exec.Cmd
// }

// type PipecatcatHandler interface {
// 	Start(llm string, stt string, tts string, message_file string) error
// 	Stop()

// 	ServeHTTP(w http.ResponseWriter, r *http.Request)
// }

// func NewPipecatHandler() PipecatcatHandler {
// 	return &pipecatHandler{}
// }

// func (h *pipecatHandler) setPort(port int) {
// 	h.port = port
// }

// func (h *pipecatHandler) setServer(server *http.Server) {
// 	h.server = server
// }

// func (h *pipecatHandler) setListener(listener net.Listener) {
// 	h.listener = listener
// }

// func (h *pipecatHandler) setArgs(llm string, stt string, tts string, message_file string) {
// 	h.llm = llm
// 	h.stt = stt
// 	h.tts = tts
// 	h.messageFile = message_file
// }

// func (h *pipecatHandler) Start(llm string, stt string, tts string, message_file string) error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "Start",
// 	})
// 	log.Debugf("Starting Pipecat WebSocket server...")

// 	h.setArgs(llm, stt, tts, message_file)

// 	if errWebsocket := h.startWebsocketListener(); errWebsocket != nil {
// 		log.Errorf("Error starting Pipecat WebSocket server: %v", errWebsocket)
// 		return errors.Wrapf(errWebsocket, "could not start the pipecat websocket server")
// 	}

// 	if errPython := h.startPythonRunner(llm, stt, tts, message_file); errPython != nil {
// 		log.Errorf("Error starting Pipecat Python runner: %v", errPython)
// 		return errors.Wrapf(errPython, "could not start the pipecat python runner")
// 	}

// 	return nil
// }

// func (h *pipecatHandler) Stop() {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "Stop",
// 	})

// 	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	if h.server != nil {
// 		log.Debugf("Shutting down Go WebSocket server...")
// 		if errShutdown := h.server.Shutdown(shutdownCtx); errShutdown != nil {
// 			log.Errorf("Go WebSocket server shutdown failed: %v", errShutdown)
// 		}
// 		log.Debugf("Go WebSocket server shut down.")
// 	}

// 	if h.cmd != nil && h.cmd.Process != nil {
// 		log.Debugf("Terminating Python process with PID %d...", h.cmd.Process.Pid)
// 		if errKill := h.cmd.Process.Kill(); errKill != nil {
// 			log.Errorf("Failed to forcefully kill Python process: %v", errKill)
// 		}
// 		log.Debugf("Python process terminated.")
// 	}

// 	h.listener.Close()
// }

// func (h *pipecatHandler) startWebsocketListener() error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "startPipecatWebsocket",
// 	})

// 	app := http.NewServeMux()
// 	app.Handle("/ws", h)

// 	listener, err := net.Listen("tcp", "localhost:0")
// 	if err != nil {
// 		log.Errorf("Failed to listen on ephemeral port: %v", err)
// 		return errors.Wrapf(err, "failed to listen on ephemeral port")
// 	}
// 	h.setListener(listener)

// 	port := listener.Addr().(*net.TCPAddr).Port
// 	h.setPort(port)
// 	log.Debugf("Server assigned to port: %d", port)

// 	server := &http.Server{
// 		Handler: app,
// 	}
// 	h.setServer(server)

// 	go func() {
// 		log.Debugf("Starting HTTP server on %s", listener.Addr().String())
// 		if errServe := server.Serve(listener); errServe != nil && errServe != http.ErrServerClosed {
// 			log.Errorf("Could not start HTTP server: %v", errServe)
// 		}
// 		log.Debugf("HTTP server stopped")
// 	}()

// 	return nil
// }

// func (h *pipecatHandler) startPythonRunner(llm string, stt string, tts string, message_file string) error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "startPipecatPython",
// 	})

// 	pythonInterpreter := "scripts/pipecat/venv/bin/python3"
// 	pythonScript := "scripts/pipecat/main.py"

// 	args := []string{
// 		"--ws_server_url", fmt.Sprintf("ws://localhost:%d/ws", h.port),
// 		"--llm", llm,
// 		"--stt", stt,
// 		"--tts", tts,
// 		"--messages_file", message_file,
// 	}

// 	cmdArgs := append([]string{pythonScript}, args...)
// 	cmd := exec.Command(pythonInterpreter, cmdArgs...)

// 	stdoutPipe, err := cmd.StdoutPipe()
// 	if err != nil {
// 		return errors.Wrapf(err, "could to get stdout pipe for python")
// 	}
// 	stderrPipe, err := cmd.StderrPipe()
// 	if err != nil {
// 		return errors.Wrapf(err, "could to get stderr pipe for python")
// 	}

// 	if errStart := cmd.Start(); errStart != nil {
// 		return errors.Wrapf(errStart, "could not start python client")
// 	}
// 	log.Debugf("Python client process started with PID: %d", cmd.Process.Pid)

// 	go func() {
// 		scanner := bufio.NewScanner(stdoutPipe)
// 		for scanner.Scan() {
// 			log.Debugf("[PYTHON-CLIENT-STDOUT] %s\n", scanner.Text())
// 		}
// 		if err := scanner.Err(); err != nil {
// 			log.Errorf("Error reading Python client stdout: %v", err)
// 		}
// 	}()

// 	go func() {
// 		scanner := bufio.NewScanner(stderrPipe)
// 		for scanner.Scan() {
// 			log.Errorf("[PYTHON-CLIENT-STDERR] %s\n", scanner.Text())
// 		}
// 		if err := scanner.Err(); err != nil {
// 			log.Errorf("Error reading Python client stderr: %v", err)
// 		}
// 	}()

// 	go func() {
// 		// wait for the python process to exit
// 		if errPython := cmd.Wait(); errPython != nil {
// 			log.Errorf("Python client process exited with error: %v", errPython)
// 		}
// 	}()

// 	return nil
// }

// func (h *pipecatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "ServeHTTP",
// 	})

// 	ws, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Printf("[WebSocket] Error upgrading to WebSocket: %v", err)
// 		return
// 	}
// 	defer func() {
// 		ws.Close()
// 		log.Println("[WebSocket] Client connection closed.")
// 	}()

// 	clientAddr := ws.RemoteAddr().String()
// 	log.Printf("[WebSocket] Client connected from %s", clientAddr)

// 	for {
// 		// ReadMessage()는 메시지 유형(text/binary)과 데이터를 byte 슬라이스로 반환합니다.
// 		msgType, message, err := ws.ReadMessage()
// 		if err != nil {
// 			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
// 				log.Printf("[WebSocket] Client %s disconnected gracefully.", clientAddr)
// 			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
// 				log.Printf("[WebSocket] Client %s disconnected unexpectedly: %v", clientAddr, err)
// 			} else {
// 				log.Printf("[WebSocket] Error reading message from client %s: %v", clientAddr, err)
// 			}
// 			break // 오류 발생 시 연결 종료
// 		}

// 		switch msgType {
// 		case websocket.BinaryMessage:
// 			// Protobuf 메시지로 역직렬화
// 			var frame pipecat_pb.Frame
// 			if err := proto.Unmarshal(message, &frame); err != nil {
// 				log.Printf("[WebSocket] Error unmarshaling Protobuf message from %s: %v", clientAddr, err)
// 				continue // 역직렬화 오류 시 다음 메시지로
// 			}

// 			// 어떤 종류의 프레임인지 확인하고 처리
// 			switch x := frame.Frame.(type) {
// 			case *pipecat_pb.Frame_Text:
// 				log.Printf("[WebSocket] Received TextFrame from %s: ID=%d, Name=%s, Text='%s'",
// 					clientAddr, x.Text.Id, x.Text.Name, x.Text.Text)
// 				// TextFrame에 대한 응답 (예시: 받은 텍스트를 포함하는 새로운 TextFrame 생성)
// 				responseFrame := &pipecat_pb.Frame{
// 					Frame: &pipecat_pb.Frame_Text{
// 						Text: &pipecat_pb.TextFrame{
// 							Id:   x.Text.Id + 1, // ID를 증가시키는 예시
// 							Name: "GoServerResponse",
// 							Text: fmt.Sprintf("Go server received your text: '%s'", x.Text.Text),
// 						},
// 					},
// 				}
// 				sendProtobufFrame(ws, responseFrame, clientAddr)

// 			case *pipecat_pb.Frame_Audio:
// 				log.Printf("[WebSocket] Received AudioRawFrame from %s: ID=%d, Name=%s, SampleRate=%d, Channels=%d, AudioLen=%d",
// 					clientAddr, x.Audio.Id, x.Audio.Name, x.Audio.SampleRate, x.Audio.NumChannels, len(x.Audio.Audio))
// 				// AudioRawFrame에 대한 응답 (예시: 받은 오디오를 그대로 돌려보냅니다)
// 				responseFrame := &pipecat_pb.Frame{
// 					Frame: &pipecat_pb.Frame_Audio{
// 						Audio: &pipecat_pb.AudioRawFrame{
// 							Id:          x.Audio.Id + 1,
// 							Name:        "GoServerAudioEcho",
// 							Audio:       x.Audio.Audio,
// 							SampleRate:  x.Audio.SampleRate,
// 							NumChannels: x.Audio.NumChannels,
// 							// Pts 필드는 optional이므로, 원래 값이 있으면 함께 보냅니다.
// 							Pts: x.Audio.Pts,
// 						},
// 					},
// 				}
// 				sendProtobufFrame(ws, responseFrame, clientAddr)

// 			case *pipecat_pb.Frame_Transcription:
// 				log.Printf("[WebSocket] Received TranscriptionFrame from %s: ID=%d, Name=%s, Text='%s', UserID=%s, Timestamp=%s",
// 					clientAddr, x.Transcription.Id, x.Transcription.Name, x.Transcription.Text, x.Transcription.UserId, x.Transcription.Timestamp)
// 				// TranscriptionFrame에 대한 응답 (예시)
// 				responseFrame := &pipecat_pb.Frame{
// 					Frame: &pipecat_pb.Frame_Transcription{
// 						Transcription: &pipecat_pb.TranscriptionFrame{
// 							Id:        x.Transcription.Id + 1,
// 							Name:      "GoServerTranscriptionResponse",
// 							Text:      fmt.Sprintf("Go server heard: '%s'", x.Transcription.Text),
// 							UserId:    x.Transcription.UserId,
// 							Timestamp: time.Now().Format(time.RFC3339),
// 						},
// 					},
// 				}
// 				sendProtobufFrame(ws, responseFrame, clientAddr)

// 			case *pipecat_pb.Frame_Message:
// 				log.Printf("[WebSocket] Received MessageFrame from %s: Data='%s'", clientAddr, x.Message.Data)
// 				if errMessage := h.receiveMessageFrameMessage(ws, []byte(x.Message.Data)); errMessage != nil {
// 					log.Errorf("Error processing MessageFrame from %s: %v", clientAddr, errMessage)
// 				}
// 				// MessageFrame에 대한 응답 (예시)
// 				// responseFrame := &pipecat_pb.Frame{
// 				// 	Frame: &pipecat_pb.Frame_Message{
// 				// 		Message: &pipecat_pb.MessageFrame{
// 				// 			Data: fmt.Sprintf("Go server received generic message: '%s'", x.Message.Data),
// 				// 		},
// 				// 	},
// 				// }
// 				// sendProtobufFrame(ws, responseFrame, clientAddr)

// 			default:
// 				log.Printf("[WebSocket] Received unknown Protobuf Frame type from %s: %T", clientAddr, x)
// 			}

// 		case websocket.TextMessage:
// 			// Protobuf 통신으로 전환했으므로, 텍스트 메시지는 예외적인 경우에만 처리합니다.
// 			// 여기서는 로깅만 하고 응답은 하지 않습니다.
// 			log.Printf("[WebSocket] Received unexpected Text from %s: %s (Expecting Protobuf Binary)", clientAddr, string(message))

// 		case websocket.CloseMessage:
// 			log.Printf("[WebSocket] Client %s sent close message.", clientAddr)
// 			return
// 		case websocket.PingMessage:
// 			log.Printf("[WebSocket] Received Ping from %s. Sending Pong.", clientAddr)
// 			if err := ws.WriteMessage(websocket.PongMessage, []byte{}); err != nil {
// 				log.Printf("[WebSocket] Error sending Pong to client %s: %v", clientAddr, err)
// 				return
// 			}
// 		case websocket.PongMessage:
// 			log.Printf("[WebSocket] Received Pong from %s.", clientAddr)
// 		default:
// 			log.Printf("[WebSocket] Received unknown message type %d from %s", msgType, clientAddr)
// 		}
// 	}
// }

// func (h *pipecatHandler) receiveMessageFrameMessage(ws *websocket.Conn, message []byte) error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "receiveMessageFrameMessage",
// 	})

// 	frame := pipecat.CommonFrameMessage{}
// 	if errUnmarshal := json.Unmarshal(message, &frame); errUnmarshal != nil {
// 		log.Errorf("Error unmarshaling JSON message: %v", errUnmarshal)
// 		return errUnmarshal
// 	}

// 	log.Printf("Received message frame: Label=%s, Type=%s", frame.Label, frame.Type)
// 	return nil
// }
