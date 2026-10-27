package rag

import (
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	rmdocument "monorepo/bin-rag-manager/models/document"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	deleted := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	fileID := uuid.FromStringOrNil("55555555-5555-4555-8555-555555555555")

	r := &Rag{
		ID:          uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111"),
		CustomerID:  uuid.FromStringOrNil("22222222-2222-4222-8222-222222222222"),
		Name:        "kb",
		Description: "desc",
		TMCreate:    &created,
		TMUpdate:    &updated,
		TMDelete:    &deleted,
		Status:      rmdocument.StatusReady,
		Sources: []Source{
			{StorageFileID: &fileID, SourceURL: "https://example.invalid/a.txt", Status: rmdocument.StatusError, StatusMessage: "failed"},
		},
	}

	expectCreated := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	expectUpdated := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	expectFileID := uuid.FromStringOrNil("55555555-5555-4555-8555-555555555555")
	expect := &WebhookMessage{
		ID:          uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111"),
		CustomerID:  uuid.FromStringOrNil("22222222-2222-4222-8222-222222222222"),
		Name:        "kb",
		Description: "desc",
		Status:      rmdocument.StatusReady,
		Sources: []Source{
			{StorageFileID: &expectFileID, SourceURL: "https://example.invalid/a.txt", Status: rmdocument.StatusError, StatusMessage: "failed"},
		},
		TMCreate: &expectCreated,
		TMUpdate: &expectUpdated,
	}

	res := r.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res)
	}
}
