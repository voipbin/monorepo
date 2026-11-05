package messagehandler

import (
	"context"
	stderrors "errors"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// limits of an mms. The values follow the public documentation of the providers: at most 10
// attachments, a total below 1 MB with headroom for the encoding overhead, images up to 1 MB and
// audio and documents up to 600 KB (carrier dependent).
const (
	mmsMediaMaxCount     = 10
	mmsMediaMaxTotalSize = int64(900 * 1024)
	mmsMediaMaxSizeImage = int64(900 * 1024)
	mmsMediaMaxSizeOther = int64(600 * 1024)
	mmsTextMaxLength     = 2000
	mmsMediaCheckTimeout = 10 * time.Second // one deadline for the media validation and the url refresh together
)

// mmsMediaExtensionLimits lists the accepted file name extensions (lower case) with the size limit
// of each. No mime type is stored for a file, so the extension is the first filter and the provider
// is the final judge.
var mmsMediaExtensionLimits = map[string]int64{
	".jpg":  mmsMediaMaxSizeImage,
	".jpeg": mmsMediaMaxSizeImage,
	".png":  mmsMediaMaxSizeImage,
	".gif":  mmsMediaMaxSizeImage,
	".mp3":  mmsMediaMaxSizeOther,
	".wav":  mmsMediaMaxSizeOther,
	".pdf":  mmsMediaMaxSizeOther,
}

func mediaError(reason string, message string) error {
	return cerrors.InvalidArgument(commonoutline.ServiceNameMessageManager, reason, message)
}

func mediaDeadlineError() error {
	return cerrors.Unavailable(commonoutline.ServiceNameMessageManager, "MEDIA_CHECK_TIMEOUT", "The media check did not finish in time. The message was not created and the request can be retried.")
}

// validateMMSText checks the text of an mms.
func validateMMSText(text string) error {
	if utf8.RuneCountInString(text) > mmsTextMaxLength {
		return mediaError("TEXT_TOO_LONG", "The text of a message with media can have at most 2000 characters.")
	}

	return nil
}

// validateMediaIDs checks the list itself, before any storage call. Because the list is at most
// mmsMediaMaxCount long, the parallel storage calls below are bounded by that count.
func validateMediaIDs(ids []uuid.UUID) error {
	if len(ids) == 0 {
		return mediaError("MEDIA_REQUIRED", "At least one media is required.")
	}
	if len(ids) > mmsMediaMaxCount {
		return mediaError("MEDIA_TOO_MANY", "A message can have at most 10 medias.")
	}

	seen := map[uuid.UUID]bool{}
	for _, id := range ids {
		if id == uuid.Nil {
			return mediaError("INVALID_MEDIA_ID", "A media id is not valid.")
		}
		if seen[id] {
			return mediaError("MEDIA_DUPLICATE", "A media id is listed more than once.")
		}
		seen[id] = true
	}

	return nil
}

// validateMedias checks every media of the customer: the file exists, is not deleted, belongs to
// the customer, was uploaded as a media, has an accepted extension and size, and the total size
// fits. The storage calls run in parallel. When more than one check fails, the error of the lowest
// list index is returned, so the result does not depend on timing.
func (h *messageHandler) validateMedias(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "validateMedias",
		"customer_id": customerID,
		"count":       len(ids),
	})

	if err := validateMediaIDs(ids); err != nil {
		return err
	}

	files := make([]*smfile.File, len(ids))
	errs := make([]error, len(ids))

	var wg sync.WaitGroup
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			files[i], errs[i] = h.reqHandler.StorageV1FileGet(ctx, ids[i])
		}(i)
	}
	wg.Wait()

	if ctx.Err() != nil {
		return mediaDeadlineError()
	}

	var total int64
	for i, f := range files {
		if errs[i] != nil {
			var ve *cerrors.VoipbinError
			if stderrors.As(errs[i], &ve) && ve.Status == cerrors.StatusNotFound {
				return mediaError("MEDIA_NOT_FOUND", "A media was not found.")
			}

			log.Errorf("Could not get the media file. index: %d, err: %v", i, errs[i])
			return errs[i]
		}

		// a missing, deleted or foreign file is one and the same answer, so the existence of the
		// files of other customers is not revealed.
		if f == nil || f.TMDelete != nil || f.CustomerID != customerID {
			return mediaError("MEDIA_NOT_FOUND", "A media was not found.")
		}
		if f.Type != smfile.TypeMedia {
			return mediaError("MEDIA_FILE_TYPE_INVALID", "A media file must be uploaded with the type media.")
		}

		limit, ok := mmsMediaExtensionLimits[strings.ToLower(filepath.Ext(f.Filename))]
		if !ok {
			return mediaError("MEDIA_TYPE_UNSUPPORTED", "The file type of a media is not supported.")
		}
		if f.Filesize > limit {
			return mediaError("MEDIA_TOO_LARGE", "A media file is too large.")
		}
		total += f.Filesize
	}
	if total > mmsMediaMaxTotalSize {
		return mediaError("MEDIA_TOO_LARGE", "The total size of the medias is too large.")
	}

	return nil
}

// refreshMediaURLs returns a fresh signed download url of every media, in the order of the ids. The
// urls are handed to the provider and are never stored or logged.
func (h *messageHandler) refreshMediaURLs(ctx context.Context, ids []uuid.UUID) ([]string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "refreshMediaURLs",
		"count": len(ids),
	})

	urls := make([]string, len(ids))
	errs := make([]error, len(ids))

	var wg sync.WaitGroup
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			urls[i], errs[i] = h.reqHandler.StorageV1FileDownloadURIRefresh(ctx, ids[i])
		}(i)
	}
	wg.Wait()

	if ctx.Err() != nil {
		return nil, mediaDeadlineError()
	}

	for i := range ids {
		if errs[i] != nil {
			log.Errorf("Could not refresh the download url. index: %d, err: %v", i, errs[i])
			return nil, errs[i]
		}
		if urls[i] == "" {
			log.Errorf("The download url is empty. index: %d", i)
			return nil, cerrors.Unavailable(commonoutline.ServiceNameMessageManager, "MEDIA_URL_UNAVAILABLE", "The download url of a media is not available.")
		}
	}

	return urls, nil
}
