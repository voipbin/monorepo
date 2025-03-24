package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/voip-asterisk-proxy/pkg/listenhandler/request"

	"github.com/sirupsen/logrus"
)

// processProxyRecordingFileMovePost handles POST /proxy/recording_file_move request
// It moves recording files.
func (h *listenHandler) processProxyRecordingFileMovePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processProxyRecordingFileMovePost",
		"request": m,
	})

	var req request.ProxyDataRecordingFileMovePost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	if errMove := h.serviceHandler.RecordingFileMove(ctx, req.Filenames); errMove != nil {
		log.Errorf("Could not move the recording files. filenames: %v, err: %v", req.Filenames, errMove)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}
