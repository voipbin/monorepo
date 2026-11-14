package emailhandler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	testAttachmentCustomerID = uuid.FromStringOrNil("5e8f0f0a-9999-4999-8999-000000000001")
	testAttachmentOtherID    = uuid.FromStringOrNil("5e8f0f0a-9999-4999-8999-000000000002")
	testAttachmentFileID     = uuid.FromStringOrNil("5e8f0f0a-9999-4999-8999-000000000011")
	testAttachmentFileID2    = uuid.FromStringOrNil("5e8f0f0a-9999-4999-8999-000000000012")
	testAttachmentRecording  = uuid.FromStringOrNil("5e8f0f0a-9999-4999-8999-000000000021")
)

func testRecordingFilters(customerID uuid.UUID, recordingID uuid.UUID) map[smfile.Field]any {
	return map[smfile.Field]any{
		smfile.FieldCustomerID:    customerID,
		smfile.FieldReferenceType: smfile.ReferenceTypeRecording,
		smfile.FieldReferenceID:   recordingID,
		smfile.FieldDeleted:       false,
	}
}

func testRecordingFiles(customerID uuid.UUID, sizes ...int64) []smfile.File {
	res := []smfile.File{}
	for _, size := range sizes {
		res = append(res, smfile.File{
			Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
			Type:     smfile.TypeRecording,
			Filesize: size,
		})
	}
	return res
}

func testStorageFile(id uuid.UUID, customerID uuid.UUID, fileType smfile.Type, filename string, size int64) *smfile.File {
	return &smfile.File{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		Type:     fileType,
		Filename: filename,
		Filesize: size,
	}
}

func testFileAttachment(id uuid.UUID) email.Attachment {
	return email.Attachment{ReferenceType: email.AttachmentReferenceTypeFile, ReferenceID: id}
}

func testRecordingAttachment(id uuid.UUID) email.Attachment {
	return email.Attachment{ReferenceType: email.AttachmentReferenceTypeRecording, ReferenceID: id}
}

// testErrorOf returns the status and the reason of a typed error.
func testErrorOf(err error) (cerrors.Status, string) {
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		return ve.Status, ve.Reason
	}
	return "", ""
}

func Test_validateAttachments(t *testing.T) {
	const mib = int64(1024 * 1024)

	deleted := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	deletedFile := testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.png", 10)
	deletedFile.TMDelete = &deleted

	tests := []struct {
		name string

		attachments []email.Attachment
		setup       func(m *requesthandler.MockRequestHandler)

		expectStatus cerrors.Status
		expectReason string
	}{
		{
			name:        "nil",
			attachments: nil,
		},
		{
			name:        "empty",
			attachments: []email.Attachment{},
		},
		{
			name:        "file_talk",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeTalk, "a.png", 10), nil)
			},
		},
		{
			name:        "file_media",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.png", 10), nil)
			},
		},
		{
			name:        "file_recording_type",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeRecording, "a.wav", 10), nil)
			},
		},
		{
			name:        "file_rag",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeRAG, "a.pdf", 10), nil)
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_FILE_TYPE_INVALID",
		},
		{
			name:        "file_empty_type",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeNone, "a.pdf", 10), nil)
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_FILE_TYPE_INVALID",
		},
		{
			name:        "file_of_another_customer",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentOtherID, smfile.TypeMedia, "a.png", 10), nil)
			},
			expectStatus: cerrors.StatusNotFound,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:        "file_deleted",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(deletedFile, nil)
			},
			expectStatus: cerrors.StatusNotFound,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:        "file_nil_response",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(nil, nil)
			},
			expectStatus: cerrors.StatusNotFound,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:        "file_typed_not_found",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(nil, cerrors.NotFound("storage-manager", "FILE_NOT_FOUND", "The file was not found."))
			},
			expectStatus: cerrors.StatusNotFound,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:        "file_sentinel_not_found",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(nil, fmt.Errorf("wrapped: %w", requesthandler.ErrNotFound))
			},
			expectStatus: cerrors.StatusNotFound,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:        "file_storage_error",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(nil, fmt.Errorf("storage is down"))
			},
			expectStatus: cerrors.StatusUnavailable,
			expectReason: "ATTACHMENT_CHECK_UNAVAILABLE",
		},
		{
			name:        "recording",
			attachments: []email.Attachment{testRecordingAttachment(testAttachmentRecording)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return(testRecordingFiles(testAttachmentCustomerID, 10, 20), nil)
			},
		},
		{
			name:        "recording_empty_list",
			attachments: []email.Attachment{testRecordingAttachment(testAttachmentRecording)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return([]smfile.File{}, nil)
			},
			expectStatus: cerrors.StatusNotFound,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:        "recording_file_of_another_customer",
			attachments: []email.Attachment{testRecordingAttachment(testAttachmentRecording)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return(testRecordingFiles(testAttachmentOtherID, 10), nil)
			},
			expectStatus: cerrors.StatusNotFound,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:        "recording_list_cut_at_the_page_size",
			attachments: []email.Attachment{testRecordingAttachment(testAttachmentRecording)},
			setup: func(m *requesthandler.MockRequestHandler) {
				sizes := make([]int64, 100)
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return(testRecordingFiles(testAttachmentCustomerID, sizes...), nil)
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_TOO_LARGE",
		},
		{
			name:        "recording_storage_error",
			attachments: []email.Attachment{testRecordingAttachment(testAttachmentRecording)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return(nil, fmt.Errorf("storage is down"))
			},
			expectStatus: cerrors.StatusUnavailable,
			expectReason: "ATTACHMENT_CHECK_UNAVAILABLE",
		},
		{
			name:         "none_type",
			attachments:  []email.Attachment{{ReferenceType: email.AttachmentReferenceTypeNone, ReferenceID: testAttachmentFileID}},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_REFERENCE_INVALID",
		},
		{
			name:         "unknown_type",
			attachments:  []email.Attachment{{ReferenceType: email.AttachmentReferenceType("rag"), ReferenceID: testAttachmentFileID}},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_REFERENCE_INVALID",
		},
		{
			name:         "nil_id",
			attachments:  []email.Attachment{testFileAttachment(uuid.Nil)},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_REFERENCE_INVALID",
		},
		{
			name: "six_attachments",
			attachments: []email.Attachment{
				testFileAttachment(uuid.Must(uuid.NewV4())), testFileAttachment(uuid.Must(uuid.NewV4())), testFileAttachment(uuid.Must(uuid.NewV4())),
				testFileAttachment(uuid.Must(uuid.NewV4())), testFileAttachment(uuid.Must(uuid.NewV4())), testFileAttachment(uuid.Must(uuid.NewV4())),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_TOO_MANY",
		},
		{
			name: "five_attachments",
			attachments: []email.Attachment{
				testFileAttachment(testAttachmentFileID), testFileAttachment(testAttachmentFileID2), testRecordingAttachment(testAttachmentRecording),
				testFileAttachment(uuid.FromStringOrNil("5e8f0f0a-9999-4999-8999-000000000013")), testFileAttachment(uuid.FromStringOrNil("5e8f0f0a-9999-4999-8999-000000000014")),
			},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), gomock.Any()).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.png", 10), nil).Times(4)
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return(testRecordingFiles(testAttachmentCustomerID, 10), nil)
			},
		},
		{
			name:         "duplicate",
			attachments:  []email.Attachment{testFileAttachment(testAttachmentFileID), testFileAttachment(testAttachmentFileID)},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_DUPLICATE",
		},
		{
			name:        "same_id_with_another_type_is_not_a_duplicate",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID), testRecordingAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.png", 10), nil)
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentFileID)).Return(testRecordingFiles(testAttachmentCustomerID, 10), nil)
			},
		},
		{
			name:        "total_exactly_at_the_limit",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.bin", 18*mib), nil)
			},
		},
		{
			name:        "total_one_byte_over_the_limit",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.bin", 18*mib+1), nil)
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_TOO_LARGE",
		},
		{
			name:        "file_negative_size",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.bin", -1), nil)
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_SIZE_INVALID",
		},
		{
			name:        "recording_negative_size",
			attachments: []email.Attachment{testRecordingAttachment(testAttachmentRecording)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return(testRecordingFiles(testAttachmentCustomerID, 10, -5), nil)
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_SIZE_INVALID",
		},
		{
			name:        "total_over_the_limit_by_the_sum",
			attachments: []email.Attachment{testFileAttachment(testAttachmentFileID), testRecordingAttachment(testAttachmentRecording)},
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.bin", 10*mib), nil)
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return(testRecordingFiles(testAttachmentCustomerID, 4*mib, 4*mib+1), nil)
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "ATTACHMENT_TOO_LARGE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			if tt.setup != nil {
				tt.setup(mockReq)
			}

			err := validateAttachments(context.Background(), mockReq, testAttachmentCustomerID, tt.attachments)

			if tt.expectReason == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}

			status, reason := testErrorOf(err)
			if status != tt.expectStatus || reason != tt.expectReason {
				t.Errorf("Wrong match. expect: %s %s, got: %s %s (%v)", tt.expectStatus, tt.expectReason, status, reason, err)
			}
		})
	}
}

// The check shares one deadline of two seconds and a storage call that runs into it is Unavailable.
func Test_validateAttachments_deadline(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockReq.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).DoAndReturn(
		func(ctx context.Context, id uuid.UUID) (*smfile.File, error) {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > attachmentCheckTimeout {
				t.Errorf("Wrong deadline. expect: within %v, got: %v (%v)", attachmentCheckTimeout, deadline, ok)
			}
			return nil, context.DeadlineExceeded
		})

	err := validateAttachments(context.Background(), mockReq, testAttachmentCustomerID, []email.Attachment{testFileAttachment(testAttachmentFileID)})

	status, reason := testErrorOf(err)
	if status != cerrors.StatusUnavailable || reason != "ATTACHMENT_CHECK_UNAVAILABLE" {
		t.Errorf("Wrong match. expect: UNAVAILABLE, got: %s %s (%v)", status, reason, err)
	}
}

func newTestCreateHandler(mc *gomock.Controller) (*emailHandler, *requesthandler.MockRequestHandler, *dbhandler.MockDBHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &emailHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		utilHandler:   utilhandler.NewMockUtilHandler(mc),

		engineSendgrid: NewMockEngineSendgrid(mc),
		engineMailgun:  NewMockEngineMailgun(mc),
	}

	return h, mockReq, mockDB
}

func testCreateDestinations() []commonaddress.Address {
	return []commonaddress.Address{{Type: commonaddress.TypeEmail, Target: "test@voipbin.net"}}
}

// A refused attachment stops Create before the balance check and before anything is stored. The
// mocks have no expectation for the balance check, the database, the events and the engines, so any
// call of them fails the test.
func Test_Create_attachmentRefused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq, _ := newTestCreateHandler(mc)
	ctx := context.Background()

	mockReq.EXPECT().CustomerV1CustomerGet(ctx, testAttachmentCustomerID).Return(&cucustomer.Customer{
		ID:                         testAttachmentCustomerID,
		IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
	}, nil)
	mockReq.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentOtherID, smfile.TypeMedia, "a.png", 10), nil)

	res, err := h.Create(ctx, testAttachmentCustomerID, uuid.Nil, testCreateDestinations(), "subject", "content", []email.Attachment{testFileAttachment(testAttachmentFileID)})
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if status, reason := testErrorOf(err); status != cerrors.StatusNotFound || reason != "ATTACHMENT_NOT_FOUND" {
		t.Errorf("Wrong match. expect: NOT_FOUND, got: %s %s (%v)", status, reason, err)
	}
}

// The attachments are checked after the identity gate and before the balance check.
func Test_Create_attachmentCheckOrder(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq, _ := newTestCreateHandler(mc)
	ctx := context.Background()

	gomock.InOrder(
		mockReq.EXPECT().CustomerV1CustomerGet(ctx, testAttachmentCustomerID).Return(&cucustomer.Customer{
			ID:                         testAttachmentCustomerID,
			IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
		}, nil),
		mockReq.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.png", 10), nil),
		mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, testAttachmentCustomerID, bmbilling.ReferenceTypeEmail, "", 1).Return(false, nil),
	)

	_, err := h.Create(ctx, testAttachmentCustomerID, uuid.Nil, testCreateDestinations(), "subject", "content", []email.Attachment{testFileAttachment(testAttachmentFileID)})
	if err == nil || !strings.Contains(err.Error(), "insufficient balance") {
		t.Errorf("Wrong match. expect: insufficient balance, got: %v", err)
	}
}

// An email without attachments (the other services of the platform send none) makes no storage call.
func Test_Create_noAttachmentNoStorageCall(t *testing.T) {
	tests := []struct {
		name        string
		attachments []email.Attachment
	}{
		{"nil", nil},
		{"empty", []email.Attachment{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockReq, _ := newTestCreateHandler(mc)
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, testAttachmentCustomerID).Return(&cucustomer.Customer{
				ID:                         testAttachmentCustomerID,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, testAttachmentCustomerID, bmbilling.ReferenceTypeEmail, "", 1).Return(false, nil)

			_, err := h.Create(ctx, testAttachmentCustomerID, uuid.Nil, testCreateDestinations(), "subject", "content", tt.attachments)
			if err == nil || !strings.Contains(err.Error(), "insufficient balance") {
				t.Errorf("Wrong match. expect: insufficient balance, got: %v", err)
			}
		})
	}
}

func Test_resolveAttachment(t *testing.T) {
	deleted := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	deletedFile := testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.png", 10)
	deletedFile.TMDelete = &deleted

	tests := []struct {
		name string

		attachment email.Attachment
		setup      func(m *requesthandler.MockRequestHandler)

		expectFilename string
		expectType     string
		expectURI      string
		expectReason   string
		expectErr      bool
	}{
		{
			name:       "file",
			attachment: testFileAttachment(testAttachmentFileID),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "photo.PNG", 10), nil)
				m.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testAttachmentFileID).Return("https://download.example.com/f", nil)
			},
			expectFilename: "photo.PNG",
			expectType:     "image/png",
			expectURI:      "https://download.example.com/f",
		},
		{
			name:       "file_unknown_extension",
			attachment: testFileAttachment(testAttachmentFileID),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeTalk, "noext", 10), nil)
				m.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testAttachmentFileID).Return("https://download.example.com/f", nil)
			},
			expectFilename: "noext",
			expectType:     "application/octet-stream",
			expectURI:      "https://download.example.com/f",
		},
		{
			name:       "file_name_with_path_parts",
			attachment: testFileAttachment(testAttachmentFileID),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "..\\..\\etc/pass\twd.txt", 10), nil)
				m.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testAttachmentFileID).Return("https://download.example.com/f", nil)
			},
			expectFilename: "passwd.txt",
			expectType:     "text/plain",
			expectURI:      "https://download.example.com/f",
		},
		{
			name:       "file_deleted_after_create",
			attachment: testFileAttachment(testAttachmentFileID),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(deletedFile, nil)
			},
			expectErr:    true,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:       "file_of_another_customer_at_send",
			attachment: testFileAttachment(testAttachmentFileID),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentOtherID, smfile.TypeMedia, "a.png", 10), nil)
			},
			expectErr:    true,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
		{
			name:       "file_empty_uri",
			attachment: testFileAttachment(testAttachmentFileID),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "a.png", 10), nil)
				m.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testAttachmentFileID).Return("", nil)
			},
			expectErr: true,
		},
		{
			name:       "recording",
			attachment: testRecordingAttachment(testAttachmentRecording),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return(testRecordingFiles(testAttachmentCustomerID, 10), nil)
				m.EXPECT().StorageV1RecordingGet(gomock.Any(), testAttachmentRecording, 60000).Return(&smbucketfile.BucketFile{DownloadURI: "https://download.example.com/r"}, nil)
			},
			expectFilename: testAttachmentRecording.String() + ".zip",
			expectType:     "application/zip",
			expectURI:      "https://download.example.com/r",
		},
		{
			name:       "recording_of_another_customer",
			attachment: testRecordingAttachment(testAttachmentRecording),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(100), testRecordingFilters(testAttachmentCustomerID, testAttachmentRecording)).Return([]smfile.File{}, nil)
			},
			expectErr:    true,
			expectReason: "ATTACHMENT_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			tt.setup(mockReq)

			filename, contentType, uri, err := resolveAttachment(context.Background(), mockReq, testAttachmentCustomerID, &tt.attachment)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: ok")
				}
				if _, reason := testErrorOf(err); reason != tt.expectReason {
					t.Errorf("Wrong match. expect: %s, got: %s (%v)", tt.expectReason, reason, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if filename != tt.expectFilename || contentType != tt.expectType || uri != tt.expectURI {
				t.Errorf("Wrong match. expect: %s %s %s, got: %s %s %s", tt.expectFilename, tt.expectType, tt.expectURI, filename, contentType, uri)
			}
		})
	}
}

func Test_sanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain", "report.pdf", "report.pdf"},
		{"unix_path", "a/b/report.pdf", "report.pdf"},
		{"windows_path", `C:\docs\report.pdf`, "report.pdf"},
		{"parent_path", "../../report.pdf", "report.pdf"},
		{"control_characters", "re\x00po\nrt.pdf", "report.pdf"},
		{"spaces_kept_inside", "my report.pdf", "my report.pdf"},
		{"empty", "", "fallback"},
		{"dot", ".", "fallback"},
		{"dot_dot", "..", "fallback"},
		{"only_slash", "/", "fallback"},
		{"only_control", "\x01\x02", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := sanitizeFilename(tt.input, "fallback"); res != tt.expected {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expected, res)
			}
		})
	}
}

func Test_uniqueFilename(t *testing.T) {
	used := map[string]bool{}

	res := []string{
		uniqueFilename(used, "a.png"),
		uniqueFilename(used, "a.png"),
		uniqueFilename(used, "A.PNG"),
		uniqueFilename(used, "a-1.png"),
		uniqueFilename(used, "b"),
		uniqueFilename(used, "b"),
	}
	expect := []string{"a.png", "a-1.png", "A-2.PNG", "a-1-1.png", "b", "b-1"}

	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res)
	}
}

func Test_engineMailgun_Send_fileAttachments(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	capture := &mailgunCapture{}
	server := newMailgunServer(t, capture, http.StatusOK, http.StatusOK)
	defer server.Close()

	// two files with the same name and one recording
	for _, id := range []uuid.UUID{testAttachmentFileID, testAttachmentFileID2} {
		mockReq.EXPECT().StorageV1FileGet(gomock.Any(), id).Return(testStorageFile(id, testAttachmentCustomerID, smfile.TypeMedia, "photo.png", 10), nil)
		mockReq.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), id).Return(server.URL+"/download", nil)
	}

	h := newTestEngineMailgun(t, mockReq, server.URL)
	_, err := h.Send(context.Background(), testMailgunEmail([]email.Attachment{testFileAttachment(testAttachmentFileID), testFileAttachment(testAttachmentFileID2)}))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	capture.mu.Lock()
	defer capture.mu.Unlock()
	expect := []string{"photo.png=synthetic-recording-bytes", "photo-1.png=synthetic-recording-bytes"}
	if !reflect.DeepEqual(capture.files, expect) {
		t.Errorf("Wrong attachments. expect: %v, got: %v", expect, capture.files)
	}
}

func Test_engineMailgun_Send_fileDeletedAfterCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	capture := &mailgunCapture{}
	server := newMailgunServer(t, capture, http.StatusOK, http.StatusOK)
	defer server.Close()

	deleted := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	f := testStorageFile(testAttachmentFileID, testAttachmentCustomerID, smfile.TypeMedia, "photo.png", 10)
	f.TMDelete = &deleted
	mockReq.EXPECT().StorageV1FileGet(gomock.Any(), testAttachmentFileID).Return(f, nil)

	h := newTestEngineMailgun(t, mockReq, server.URL)
	_, err := h.Send(context.Background(), testMailgunEmail([]email.Attachment{testFileAttachment(testAttachmentFileID)}))
	if err == nil || !strings.Contains(err.Error(), "ATTACHMENT_NOT_FOUND") {
		t.Errorf("Wrong match. expect: ATTACHMENT_NOT_FOUND, got: %v", err)
	}
	if capture.count() != 0 {
		t.Errorf("Expected no send call. got: %d", capture.count())
	}
}

func Test_engineSendgrid_Send_fileAttachment(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	capture := &sendgridCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/mail/send":
			b, _ := io.ReadAll(r.Body)
			capture.add(b)
			w.Header().Set("X-Message-Id", "synthetic-message-id")
			w.WriteHeader(http.StatusAccepted)
		case "/download":
			_, _ = w.Write([]byte("synthetic-file-bytes"))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	for _, id := range []uuid.UUID{testAttachmentFileID, testAttachmentFileID2} {
		mockReq.EXPECT().StorageV1FileGet(gomock.Any(), id).Return(testStorageFile(id, testAttachmentCustomerID, smfile.TypeTalk, "note.txt", 10), nil)
		mockReq.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), id).Return(server.URL+"/download", nil)
	}

	h := newTestEngineSendgrid(t, mockReq, server.URL)
	if _, err := h.Send(context.Background(), testSendgridEmail([]email.Attachment{testFileAttachment(testAttachmentFileID), testFileAttachment(testAttachmentFileID2)})); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var body struct {
		Attachments []struct {
			Content  string `json:"content"`
			Type     string `json:"type"`
			Filename string `json:"filename"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal(capture.last(), &body); err != nil {
		t.Fatalf("could not parse the request body. err: %v", err)
	}
	if len(body.Attachments) != 2 {
		t.Fatalf("Wrong attachment count. expect: 2, got: %d", len(body.Attachments))
	}

	content := base64.StdEncoding.EncodeToString([]byte("synthetic-file-bytes"))
	for i, expectName := range []string{"note.txt", "note-1.txt"} {
		a := body.Attachments[i]
		if a.Filename != expectName || a.Type != "text/plain" || a.Content != content {
			t.Errorf("Wrong attachment. index: %d, expect: %s text/plain, got: %s %s", i, expectName, a.Filename, a.Type)
		}
	}
}
