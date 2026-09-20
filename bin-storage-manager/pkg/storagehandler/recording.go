package storagehandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-storage-manager/models/bucketfile"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/models/recordingpeak"
)

// RecordingGet returns given recording's bucketfile info.
func (h *storageHandler) RecordingGet(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingGet",
		"recoding_id": id,
	})

	filters := map[file.Field]any{
		file.FieldDeleted:     false,
		file.FieldReferenceID: id,
	}

	files, err := h.FileList(ctx, "", 100, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get files. reference_id: %s", id)
	}

	// create compress file
	bucketName, filepath, err := h.fileHandler.CompressCreate(ctx, files)
	if err != nil {
		return nil, errors.Wrapf(err, "could not compress the files. bucket_name: %s, filepath: %s", bucketName, filepath)
	}
	log.Debugf("Created compress file. bucket_name: %s, filepath: %s", bucketName, filepath)

	// get download uri
	bucketURI, downloadURI, err := h.fileHandler.DownloadURIGet(ctx, bucketName, filepath, time.Hour*24)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get download link. bucket_name: %s, filepath: %s", bucketName, filepath)
	}
	log.Debugf("Created download uri. len: %d", len(downloadURI))

	// create recording.Recording
	tmExpire := h.utilHandler.TimeNowAdd(24 * time.Hour)
	res := &bucketfile.BucketFile{
		ReferenceType:    bucketfile.ReferenceTypeRecording,
		ReferenceID:      id,
		BucketURI:        bucketURI,
		DownloadURI:      downloadURI,
		TMDownloadExpire: tmExpire,
	}

	return res, nil
}

// RecordingPeaks returns the recording's per-file waveform peaks and durations,
// keyed by filename. Computed on demand from the WAV bytes and cached for 24h.
// Peak computation for any individual file degrades gracefully: a failing file
// yields empty peaks and 0 duration rather than failing the whole request.
func (h *storageHandler) RecordingPeaks(ctx context.Context, id uuid.UUID) (map[string]recordingpeak.RecordingFilePeak, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "RecordingPeaks",
		"recording_id": id,
	})

	// cache hit
	if cached, err := h.cacheHandler.RecordingPeaksGet(ctx, id); err == nil && cached != nil {
		return cached, nil
	}

	filters := map[file.Field]any{
		file.FieldDeleted:     false,
		file.FieldReferenceID: id,
	}

	files, err := h.FileList(ctx, "", 100, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get files. reference_id: %s", id)
	}

	res := make(map[string]recordingpeak.RecordingFilePeak, len(files))
	for _, f := range files {
		peaks, duration, err := h.fileHandler.PeaksComputeByFile(ctx, f, 0)
		if err != nil {
			// graceful degrade: keep an empty entry so playback still works.
			log.Warnf("Could not compute peaks. filename: %s, err: %v", f.Filename, err)
			res[f.Filename] = recordingpeak.RecordingFilePeak{Peaks: []float64{}, Duration: 0}
			continue
		}
		if peaks == nil {
			peaks = []float64{}
		}
		res[f.Filename] = recordingpeak.RecordingFilePeak{Peaks: peaks, Duration: duration}
	}

	if errSet := h.cacheHandler.RecordingPeaksSet(ctx, id, res); errSet != nil {
		log.Warnf("Could not cache recording peaks. err: %v", errSet)
	}

	return res, nil
}

// RecordingDelete deletes the given recording file
func (h *storageHandler) RecordingDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingDelete",
		"recoding_id": id,
	})

	filters := map[file.Field]any{
		file.FieldDeleted:     false,
		file.FieldReferenceID: id,
	}

	files, err := h.FileList(ctx, "", 100, filters)
	if err != nil {
		return errors.Wrapf(err, "could not get files. reference_id: %s", id)
	}

	for _, f := range files {
		tmp, err := h.fileHandler.Delete(ctx, f.ID)
		if err != nil {
			return errors.Wrapf(err, "could not delete the file. file_id: %s", f.ID)
		}
		log.WithField("file", tmp).Debugf("Deleted file. file_id: %s", f.ID)
	}

	return nil
}
