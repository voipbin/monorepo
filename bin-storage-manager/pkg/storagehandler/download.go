package storagehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// FileDownloadURIRefresh generates a fresh signed download URL for the given file.
func (h *storageHandler) FileDownloadURIRefresh(ctx context.Context, id uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "FileDownloadURIRefresh",
		"id":   id,
	})

	res, err := h.fileHandler.DownloadURIRefresh(ctx, id)
	if err != nil {
		log.Errorf("Could not refresh download URI. err: %v", err)
		return "", err
	}

	return res, nil
}
