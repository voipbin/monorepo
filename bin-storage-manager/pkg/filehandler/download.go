package filehandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-storage-manager/models/file"
)

// DownloadURIRefresh generates a fresh signed download URL for the given file,
// updates the database with the new URL and expiration, and returns the URL.
func (h *fileHandler) DownloadURIRefresh(ctx context.Context, id uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "DownloadURIRefresh",
		"id":   id,
	})

	f, err := h.db.FileGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get file info. err: %v", err)
		return "", err
	}
	log.WithField("file", f).Debugf("Retrieved file info. file_id: %s", f.ID)

	expireDuration := downloadURLExpiration
	tmExpire := time.Now().UTC().Add(expireDuration)
	tmDownloadExpire := h.utilHandler.TimeNowAdd(expireDuration)

	downloadURI, err := h.bucketfileGenerateDownloadURI(f.BucketName, f.Filepath, tmExpire, f.Filename)
	if err != nil {
		log.Errorf("Could not generate download URI. err: %v", err)
		return "", err
	}
	log.Debugf("Generated fresh download URI. file_id: %s", f.ID)

	fields := map[file.Field]any{
		file.FieldURIDownload:      downloadURI,
		file.FieldTMDownloadExpire: tmDownloadExpire,
	}

	if err := h.db.FileUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update file download URI. err: %v", err)
		return "", err
	}

	return downloadURI, nil
}
