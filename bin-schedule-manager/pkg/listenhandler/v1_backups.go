package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processV1BackupsPost handles POST /v1/backups request.
// It runs a database backup now (design §7; invoked by the seeded
// database-backup schedule) and responds with the backuphandler.Result
// domain type marshaled directly (style A).
func (h *listenHandler) processV1BackupsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1BackupsPost",
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.backupHandler.Backup(ctx)
	if err != nil {
		log.Errorf("Could not run the backup. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
