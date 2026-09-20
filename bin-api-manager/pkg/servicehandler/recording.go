package servicehandler

import (
	"context"
	"sort"
	"strings"
	"time"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmrecording "monorepo/bin-call-manager/models/recording"
	smfile "monorepo/bin-storage-manager/models/file"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// recordingPlayfileRefreshThreshold is the minimum remaining lifetime of a
// file's signed download URL below which it is proactively refreshed on a
// playfiles request, reducing the chance of mid-playback expiry.
const recordingPlayfileRefreshThreshold = 30 * time.Minute

// recordingGet validates the recording's ownership and returns the recording info.
func (h *serviceHandler) recordingGet(ctx context.Context, recordingID uuid.UUID) (*cmrecording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "recordingGet",
		"transcribe_id": recordingID,
	})

	// send request
	res, err := h.reqHandler.CallV1RecordingGet(ctx, recordingID)
	if err != nil {
		log.Errorf("Could not get the call info. err: %v", err)
		return nil, err
	}
	log.WithField("recording", res).Debug("Received result.")

	if res.TMDelete != nil {
		log.Debugf("Deleted recording. recording_id: %s", res.ID)
		return nil, serviceerrors.ErrNotFound
	}

	return res, nil
}

// RecordingGet returns downloadable url for recording
func (h *serviceHandler) RecordingGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmrecording.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingGet",
		"customer_id": a.CustomerID,
		"recording":   id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get recording info from call-manager
	rec, err := h.recordingGet(ctx, id)
	if err != nil {
		// no call info found
		log.Infof("Could not get recording info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, rec.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := rec.ConvertWebhookMessage()
	return res, nil
}

// RecordingPlayfilesGet returns the recording's individual playable audio files
// with streaming download URLs and precomputed waveform peaks for inline playback.
func (h *serviceHandler) RecordingPlayfilesGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) ([]openapi_server.ApiManagerRecordingPlayfile, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "RecordingPlayfilesGet",
		"customer_id":  a.CustomerID,
		"recording_id": id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// gate: the caller must be admin/manager of the recording's owning customer.
	rec, err := h.recordingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get recording info. err: %v", err)
		return nil, err
	}
	if !h.hasPermission(ctx, a, rec.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// list the recording's storage files (the individual directional wavs).
	fileFilters := map[smfile.Field]any{
		smfile.FieldReferenceType: smfile.ReferenceTypeRecording,
		smfile.FieldReferenceID:   id,
		smfile.FieldDeleted:       false,
	}
	files, err := h.reqHandler.StorageV1FileList(ctx, "", 100, fileFilters)
	if err != nil {
		log.Errorf("Could not get storage files for the recording. err: %v", err)
		return nil, err
	}

	// waveform peaks, keyed by filename. Failure is non-fatal: playback works
	// without peaks, so an error here degrades to empty peaks.
	peaksByFile, err := h.reqHandler.StorageV1RecordingPeaks(ctx, id, 30000)
	if err != nil {
		log.Warnf("Could not get recording peaks; returning files without waveforms. err: %v", err)
		peaksByFile = nil
	}

	// direction is only meaningful for call recordings that have separate
	// in/out files; confbridge (single file, named with an _in suffix) and any
	// single-file recording must not be labeled with a direction.
	assignDirection := rec.ReferenceType == cmrecording.ReferenceTypeCall && len(files) == 2

	res := make([]openapi_server.ApiManagerRecordingPlayfile, 0, len(files))
	for i := range files {
		f := files[i]

		// proactively refresh a soon-to-expire signed URL (design §5.4 1st line).
		uriDownload := f.URIDownload
		if f.TMDownloadExpire != nil && time.Until(*f.TMDownloadExpire) < recordingPlayfileRefreshThreshold {
			if refreshed, errRefresh := h.reqHandler.StorageV1FileDownloadURIRefresh(ctx, f.ID); errRefresh != nil {
				log.Warnf("Could not refresh the download uri. file_id: %s, err: %v", f.ID, errRefresh)
			} else if refreshed != "" {
				uriDownload = refreshed
			}
		}

		item := openapi_server.ApiManagerRecordingPlayfile{
			Filename:    strPtr(f.Filename),
			UriDownload: strPtr(uriDownload),
			Filesize:    int64Ptr(f.Filesize),
			Direction:   directionPtr(fileDirection(f.Filename, assignDirection)),
		}
		if f.TMDownloadExpire != nil {
			item.TmDownloadExpire = strPtr(f.TMDownloadExpire.Format(time.RFC3339Nano))
		}
		if peak, ok := peaksByFile[f.Filename]; ok {
			peaks := peak.Peaks
			item.Peaks = &peaks
			item.Duration = float64Ptr(peak.Duration)
		} else {
			empty := []float64{}
			item.Peaks = &empty
			item.Duration = float64Ptr(0)
		}

		res = append(res, item)
	}

	// stable order: in, out, then remaining by filename.
	sort.SliceStable(res, func(i, j int) bool {
		return playfileSortKey(res[i]) < playfileSortKey(res[j])
	})

	return res, nil
}

// fileDirection derives the audio direction from the filename suffix, but only
// when the recording is eligible (call with two directional files). Otherwise
// it returns an empty direction so single/confbridge files are not mislabeled.
func fileDirection(filename string, eligible bool) openapi_server.ApiManagerRecordingPlayfileDirection {
	if !eligible {
		return openapi_server.ApiManagerRecordingPlayfileDirectionNone
	}
	base := strings.TrimSuffix(filename, ".wav")
	switch {
	case strings.HasSuffix(base, "_in"):
		return openapi_server.ApiManagerRecordingPlayfileDirectionIn
	case strings.HasSuffix(base, "_out"):
		return openapi_server.ApiManagerRecordingPlayfileDirectionOut
	default:
		return openapi_server.ApiManagerRecordingPlayfileDirectionNone
	}
}

// playfileSortKey orders in(0) before out(1) before everything else(2).
func playfileSortKey(p openapi_server.ApiManagerRecordingPlayfile) int {
	if p.Direction == nil {
		return 2
	}
	switch *p.Direction {
	case openapi_server.ApiManagerRecordingPlayfileDirectionIn:
		return 0
	case openapi_server.ApiManagerRecordingPlayfileDirectionOut:
		return 1
	default:
		return 2
	}
}

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }
func float64Ptr(f float64) *float64 {
	return &f
}
func directionPtr(d openapi_server.ApiManagerRecordingPlayfileDirection) *openapi_server.ApiManagerRecordingPlayfileDirection {
	return &d
}

// RecordingList sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) RecordingList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmrecording.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingGets",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertRecordingFilters(filters)
	if err != nil {
		return nil, err
	}

	tmp, err := h.reqHandler.CallV1RecordingList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get recordings from the call manager. err: %v", err)
		return nil, err
	}

	res := []*cmrecording.WebhookMessage{}
	for _, tmpRecord := range tmp {
		record := tmpRecord.ConvertWebhookMessage()
		res = append(res, record)
	}

	return res, nil
}

// RecordingDelete sends a request to call-manager
// to deleting a recording.
// it returns deleted recording info if it succeed.
func (h *serviceHandler) RecordingDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmrecording.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "RecordingDelete",
		"customer_id":  a.CustomerID,
		"username":     a.DisplayName(),
		"recording_id": id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	r, err := h.recordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}
	log.WithField("recording", r).Debugf("Validated recording info. recording_id: %s", r.ID)

	if !h.hasPermission(ctx, a, r.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.CallV1RecordingDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the recording. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// convertRecordingFilters converts map[string]string to map[cmrecording.Field]any
func (h *serviceHandler) convertRecordingFilters(filters map[string]string) (map[cmrecording.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, cmrecording.Recording{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[cmrecording.Field]any, len(typed))
	for k, v := range typed {
		result[cmrecording.Field(k)] = v
	}

	return result, nil
}

// RecordingTranscribeList returns the transcribes tied to the given recording,
// regardless of the transcribe owner. This includes transcribes created
// internally for AI summaries (owned by the ai-manager system account), which
// are not reachable through the customer-scoped TranscribeList. Access is gated
// by ownership of the recording itself.
func (h *serviceHandler) RecordingTranscribeList(ctx context.Context, a *auth.AuthIdentity, recordingID uuid.UUID, size uint64, token string) ([]*tmtranscribe.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":         "RecordingTranscribeList",
		"customer_id":  a.CustomerID,
		"username":     a.DisplayName(),
		"recording_id": recordingID,
		"size":         size,
		"token":        token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// gate: the caller must be admin/manager of the recording's owning customer
	rec, err := h.recordingGet(ctx, recordingID)
	if err != nil {
		log.Infof("Could not get recording info. err: %v", err)
		return nil, err
	}
	if !h.hasPermission(ctx, a, rec.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// owner-agnostic: scope by the recording's id only (no customer_id filter),
	// so transcribes owned by the ai-manager account are returned too.
	filters := map[string]string{
		"reference_type": string(tmtranscribe.ReferenceTypeRecording),
		"reference_id":   recordingID.String(),
		"deleted":        "false",
	}
	typedFilters, err := h.convertTranscribeFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.TranscribeV1TranscribeList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get transcribes for the recording. err: %v", err)
		return nil, err
	}

	res := []*tmtranscribe.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// RecordingTranscriptList returns the transcript lines for a transcribe tied to
// the given recording, regardless of the transcribe owner. Access is gated
// twice: the caller must be admin/manager of the recording's owning customer,
// and the transcribe must actually belong to this recording.
func (h *serviceHandler) RecordingTranscriptList(ctx context.Context, a *auth.AuthIdentity, recordingID uuid.UUID, transcribeID uuid.UUID, size uint64, token string) ([]*tmtranscript.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "RecordingTranscriptList",
		"customer_id":   a.CustomerID,
		"username":      a.DisplayName(),
		"recording_id":  recordingID,
		"transcribe_id": transcribeID,
		"size":          size,
		"token":         token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// gate 1: the caller must be admin/manager of the recording's owning customer
	rec, err := h.recordingGet(ctx, recordingID)
	if err != nil {
		log.Infof("Could not get recording info. err: %v", err)
		return nil, err
	}
	if !h.hasPermission(ctx, a, rec.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// gate 2: the transcribe must belong to this recording. This prevents using
	// a valid recording permission to read transcripts of another recording's
	// (or another customer's) transcribe.
	t, err := h.transcribeGet(ctx, transcribeID)
	if err != nil {
		log.Infof("Could not get transcribe info. err: %v", err)
		return nil, err
	}
	if t.ReferenceType != tmtranscribe.ReferenceTypeRecording || t.ReferenceID != recordingID {
		log.Infof("The transcribe does not belong to the recording. transcribe_reference_type: %s, transcribe_reference_id: %s", t.ReferenceType, t.ReferenceID)
		return nil, serviceerrors.ErrPermissionDenied
	}

	// owner-agnostic: scope by transcribe_id only (no customer_id filter).
	filters := map[string]string{
		"transcribe_id": transcribeID.String(),
		"deleted":       "false",
	}
	typedFilters, err := h.convertTranscriptFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.TranscribeV1TranscriptList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get transcripts for the recording. err: %v", err)
		return nil, err
	}

	res := []*tmtranscript.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}
