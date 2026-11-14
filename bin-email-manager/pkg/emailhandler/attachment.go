package emailhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"mime"
	"path"
	"strings"
	"time"
	"unicode"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-email-manager/models/email"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// limits of the attachments of one email. The total is the sum of the original sizes of the files
// (base64 and the mail headers add to it on the wire). They are fixed limits, not settings.
const (
	attachmentMaxCount          = 5
	attachmentMaxTotalSize      = int64(18 * 1024 * 1024)
	attachmentRecordingListSize = uint64(100) // the page size that the recording download of the storage service uses
	attachmentRecordingTimeout  = 60000       // milliseconds, the timeout of the recording download request
	attachmentCheckTimeout      = 2 * time.Second
)

const (
	defaultAttachmentContentType = "application/octet-stream"
	recordingAttachmentType      = "application/zip"
)

// attachmentInfo is the checked description of one attachment.
type attachmentInfo struct {
	filename    string
	contentType string
	size        int64 // original size in bytes
}

func attachmentInvalid(reason string, message string) error {
	return cerrors.InvalidArgument(commonoutline.ServiceNameEmailManager, reason, message)
}

// attachmentNotFound is the one answer for a missing, deleted and foreign file, so the existence of
// the files of other customers is not shown.
func attachmentNotFound() error {
	return cerrors.NotFound(commonoutline.ServiceNameEmailManager, "ATTACHMENT_NOT_FOUND", "An attachment was not found.")
}

// attachmentLookupError maps the error of a storage request. A missing file is NotFound, any other
// error is Unavailable: an untyped error would reach the caller as a bad request (400).
func attachmentLookupError(err error) error {
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound {
		return attachmentNotFound()
	}
	if stderrors.Is(err, requesthandler.ErrNotFound) {
		return attachmentNotFound()
	}

	return cerrors.Unavailable(
		commonoutline.ServiceNameEmailManager,
		"ATTACHMENT_CHECK_UNAVAILABLE",
		"The attachments could not be checked. The email was not created and the request can be retried.",
	).Wrap(err)
}

// validateAttachmentReference checks the reference of an attachment without any request.
func validateAttachmentReference(a *email.Attachment) error {
	if a.ReferenceID == uuid.Nil {
		return attachmentInvalid("ATTACHMENT_REFERENCE_INVALID", "The reference id of an attachment is not valid.")
	}

	switch a.ReferenceType {
	case email.AttachmentReferenceTypeFile, email.AttachmentReferenceTypeRecording:
		return nil
	default:
		return attachmentInvalid("ATTACHMENT_REFERENCE_INVALID", "The reference type of an attachment must be file or recording.")
	}
}

// validateAttachments checks the attachments of an email of the customer: the count, the
// duplicates, the owner and state of every referenced file, and the total size. All storage
// requests share one deadline, so the check does not outlast the wait of the caller.
func validateAttachments(ctx context.Context, reqHandler requesthandler.RequestHandler, customerID uuid.UUID, attachments []email.Attachment) error {
	if len(attachments) == 0 {
		return nil
	}
	if len(attachments) > attachmentMaxCount {
		return attachmentInvalid("ATTACHMENT_TOO_MANY", "An email can have at most 5 attachments.")
	}

	type key struct {
		referenceType email.AttachmentReferenceType
		referenceID   uuid.UUID
	}
	seen := map[key]bool{}
	for i := range attachments {
		if err := validateAttachmentReference(&attachments[i]); err != nil {
			return err
		}

		k := key{attachments[i].ReferenceType, attachments[i].ReferenceID}
		if seen[k] {
			return attachmentInvalid("ATTACHMENT_DUPLICATE", "An attachment is listed more than once.")
		}
		seen[k] = true
	}

	cctx, cancel := context.WithTimeout(ctx, attachmentCheckTimeout)
	defer cancel()

	var total int64
	for i := range attachments {
		info, err := inspectAttachment(cctx, reqHandler, customerID, &attachments[i])
		if err != nil {
			return err
		}
		total += info.size
	}
	if total > attachmentMaxTotalSize {
		return attachmentInvalid("ATTACHMENT_TOO_LARGE", "The total size of the attachments can be at most 18 MiB.")
	}

	return nil
}

// inspectAttachment checks one attachment of the customer and returns its description.
func inspectAttachment(ctx context.Context, reqHandler requesthandler.RequestHandler, customerID uuid.UUID, a *email.Attachment) (*attachmentInfo, error) {
	if err := validateAttachmentReference(a); err != nil {
		return nil, err
	}

	if a.ReferenceType == email.AttachmentReferenceTypeRecording {
		return inspectRecording(ctx, reqHandler, customerID, a.ReferenceID)
	}

	f, err := reqHandler.StorageV1FileGet(ctx, a.ReferenceID)
	if err != nil {
		return nil, attachmentLookupError(err)
	}

	// a missing, deleted and foreign file are one and the same answer.
	if f == nil || f.TMDelete != nil || f.CustomerID != customerID {
		return nil, attachmentNotFound()
	}

	switch f.Type {
	case smfile.TypeTalk, smfile.TypeMedia, smfile.TypeRecording:
	default:
		return nil, attachmentInvalid("ATTACHMENT_FILE_TYPE_INVALID", "The type of the file can not be attached. Only talk, media and recording files can be attached.")
	}

	if f.Filesize < 0 {
		return nil, attachmentInvalid("ATTACHMENT_SIZE_INVALID", "The size of an attachment is not valid.")
	}

	return &attachmentInfo{
		filename:    sanitizeFilename(f.Filename, f.ID.String()),
		contentType: contentTypeByFilename(f.Filename),
		size:        f.Filesize,
	}, nil
}

// inspectRecording checks a recording of the customer by listing its files with the customer id,
// because the download request of the storage service does not know the customer.
func inspectRecording(ctx context.Context, reqHandler requesthandler.RequestHandler, customerID uuid.UUID, recordingID uuid.UUID) (*attachmentInfo, error) {
	filters := map[smfile.Field]any{
		smfile.FieldCustomerID:    customerID,
		smfile.FieldReferenceType: smfile.ReferenceTypeRecording,
		smfile.FieldReferenceID:   recordingID,
		smfile.FieldDeleted:       false,
	}

	files, err := reqHandler.StorageV1FileList(ctx, "", attachmentRecordingListSize, filters)
	if err != nil {
		return nil, attachmentLookupError(err)
	}
	if len(files) == 0 {
		return nil, attachmentNotFound()
	}
	if uint64(len(files)) >= attachmentRecordingListSize {
		// the list may be cut, so the sum would be too small.
		return nil, attachmentInvalid("ATTACHMENT_TOO_LARGE", "The recording has too many files to attach.")
	}

	var size int64
	for i := range files {
		if files[i].CustomerID != customerID || files[i].TMDelete != nil {
			return nil, attachmentNotFound()
		}
		if files[i].Filesize < 0 {
			return nil, attachmentInvalid("ATTACHMENT_SIZE_INVALID", "The size of an attachment is not valid.")
		}
		size += files[i].Filesize
	}

	return &attachmentInfo{
		filename:    fmt.Sprintf("%s.zip", recordingID),
		contentType: recordingAttachmentType,
		size:        size,
	}, nil
}

// resolveAttachment checks the attachment again (the file can be deleted after the email was
// created) and returns the file name, the content type and a download uri of the attachment.
func resolveAttachment(ctx context.Context, reqHandler requesthandler.RequestHandler, customerID uuid.UUID, a *email.Attachment) (string, string, string, error) {
	info, err := inspectAttachment(ctx, reqHandler, customerID, a)
	if err != nil {
		return "", "", "", err
	}

	if a.ReferenceType == email.AttachmentReferenceTypeRecording {
		f, errGet := reqHandler.StorageV1RecordingGet(ctx, a.ReferenceID, attachmentRecordingTimeout)
		if errGet != nil {
			return "", "", "", errors.Wrap(errGet, "could not get recording")
		}

		return info.filename, info.contentType, f.DownloadURI, nil
	}

	uri, err := reqHandler.StorageV1FileDownloadURIRefresh(ctx, a.ReferenceID)
	if err != nil {
		return "", "", "", errors.Wrap(err, "could not get the download uri of the file")
	}
	if uri == "" {
		return "", "", "", errors.New("the download uri of the file is empty")
	}

	return info.filename, info.contentType, uri, nil
}

// sanitizeFilename returns the file name without path parts and control characters. The fallback is
// used when nothing is left.
func sanitizeFilename(name string, fallback string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)

	switch name {
	case "", ".", "..", "/":
		return fallback
	}

	return name
}

// uniqueFilename returns the name, or the name with a -<n> suffix before the extension when an
// earlier attachment of the email already uses it (the comparison ignores the case).
func uniqueFilename(used map[string]bool, name string) string {
	if !used[strings.ToLower(name)] {
		used[strings.ToLower(name)] = true
		return name
	}

	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s-%d%s", base, n, ext)
		if !used[strings.ToLower(candidate)] {
			used[strings.ToLower(candidate)] = true
			return candidate
		}
	}
}

// contentTypeByFilename returns the media type of the extension of the file name.
func contentTypeByFilename(filename string) string {
	tmp := mime.TypeByExtension(strings.ToLower(path.Ext(filename)))
	if tmp == "" {
		return defaultAttachmentContentType
	}

	mediaType, _, err := mime.ParseMediaType(tmp)
	if err != nil {
		return defaultAttachmentContentType
	}

	return mediaType
}
