package streamhandler

// // receiveBinaryFromWebsock receives the binary byte from the websock
// func (h *streamHandler) receiveBinaryFromWebsock(ctx context.Context, ws *websocket.Conn) ([]byte, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "receiveBinaryFromWebsock",
// 	})

// 	if ctx.Err() != nil {
// 		log.Infof("The context is canceled. Exiting the receiving loop. err: %v", ctx.Err())
// 		return nil, ctx.Err()
// 	}

// 	// read the message from the websocket
// 	t, m, err := ws.ReadMessage()
// 	if err != nil {
// 		log.Infof("Could not read the message correctly. err: %v", err)
// 		return nil, err
// 	}

// 	if t != websocket.BinaryMessage {
// 		// wrong message type
// 		return nil, fmt.Errorf("wrong message type")
// 	}

// 	return m, nil
// }

// // receiveBinaryFromWebsock receives the binary byte from the websock
// func (h *streamHandler) receiveBinaryFromAudiosocket(ctx context.Context, conn net.Conn) ([]byte, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "receiveBinaryFromWebsock",
// 	})

// 	m, err := audiosocket.NextMessage(conn)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not receive audiosock data. err: %v", err)
// 	}

// 	switch {
// 	case m.Kind() == audiosocket.KindError:
// 		log.Debugf("Received error. err: %d", m.ErrorCode())

// 	case m.Kind() != audiosocket.KindSlin:
// 		log.Debugf("Ignoring non-slin message")
// 		continue

// 	case m.ContentLength() < 1:
// 		log.Debugf("No content")
// 		continue

// 	default:
// 		if errWrite := st.ConnWebsocket.WriteMessage(websocket.BinaryMessage, m); errWrite != nil {
// 			log.Debugf("Could not write the message. err: %v", errWrite)
// 			return
// 		}
// 	}
// }
