package dbhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/pkg/cachehandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_SpeakingCreate(t *testing.T) {
	tests := []struct {
		name      string
		speaking  *speaking.Speaking
		mockError error
		expectErr bool
	}{
		{
			name: "normal",
			speaking: &speaking.Speaking{
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				Language:      "en-US",
				Provider:      "elevenlabs",
				VoiceID:       "voice123",
			},
			expectErr: false,
		},
		{
			name: "database error",
			speaking: &speaking.Speaking{
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			},
			mockError: fmt.Errorf("db error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create mock: %v", err)
			}
			defer func() {
				_ = db.Close()
			}()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				db:    db,
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			if tt.mockError != nil {
				mock.ExpectExec("INSERT INTO").WillReturnError(tt.mockError)
			} else {
				mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
			}

			err = h.SpeakingCreate(ctx, tt.speaking)
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil && !tt.expectErr {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func Test_SpeakingGet(t *testing.T) {
	speakingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	customerID := uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")
	referenceID := uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13")
	tmCreate := time.Now()

	tests := []struct {
		name      string
		id        uuid.UUID
		mockRows  *sqlmock.Rows
		mockError error
		expectErr bool
	}{
		{
			name: "normal",
			id:   speakingID,
			mockRows: sqlmock.NewRows([]string{
				"id", "customer_id", "reference_type", "reference_id",
				"language", "provider", "voice_id", "direction",
				"status", "pod_id", "tm_create", "tm_update", "tm_delete",
			}).AddRow(
				speakingID.Bytes(), customerID.Bytes(), "call", referenceID.Bytes(),
				"en-US", "elevenlabs", "voice123", "in",
				"active", "pod1", tmCreate, nil, nil,
			),
			expectErr: false,
		},
		{
			name:      "not found",
			id:        speakingID,
			mockRows:  sqlmock.NewRows([]string{"id"}),
			expectErr: true,
		},
		{
			name:      "query error",
			id:        speakingID,
			mockError: fmt.Errorf("query error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create mock: %v", err)
			}
			defer func() {
				_ = db.Close()
			}()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				db:    db,
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			if tt.mockError != nil {
				mock.ExpectQuery("SELECT .* FROM").WillReturnError(tt.mockError)
			} else {
				mock.ExpectQuery("SELECT .* FROM").WillReturnRows(tt.mockRows)
			}

			res, err := h.SpeakingGet(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if res == nil {
					t.Errorf("expected result, got nil")
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil && !tt.expectErr {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func Test_SpeakingGets(t *testing.T) {
	speakingID1 := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	speakingID2 := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")
	customerID := uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13")
	referenceID := uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14")
	tmCreate := time.Now()

	tests := []struct {
		name      string
		token     string
		size      uint64
		filters   map[speaking.Field]any
		mockRows  *sqlmock.Rows
		mockError error
		expectErr bool
		expectLen int
	}{
		{
			name:  "normal with results",
			token: "",
			size:  10,
			filters: map[speaking.Field]any{
				speaking.FieldCustomerID: customerID,
			},
			mockRows: sqlmock.NewRows([]string{
				"id", "customer_id", "reference_type", "reference_id",
				"language", "provider", "voice_id", "direction",
				"status", "pod_id", "tm_create", "tm_update", "tm_delete",
			}).AddRow(
				speakingID1.Bytes(), customerID.Bytes(), "call", referenceID.Bytes(),
				"en-US", "elevenlabs", "voice123", "in",
				"active", "pod1", tmCreate, nil, nil,
			).AddRow(
				speakingID2.Bytes(), customerID.Bytes(), "call", referenceID.Bytes(),
				"en-US", "elevenlabs", "voice456", "out",
				"stopped", "pod2", tmCreate, nil, nil,
			),
			expectErr: false,
			expectLen: 2,
		},
		{
			name:    "empty result",
			token:   "2024-01-01T00:00:00.000000Z",
			size:    10,
			filters: map[speaking.Field]any{},
			mockRows: sqlmock.NewRows([]string{
				"id", "customer_id", "reference_type", "reference_id",
				"language", "provider", "voice_id", "direction",
				"status", "pod_id", "tm_create", "tm_update", "tm_delete",
			}),
			expectErr: false,
			expectLen: 0,
		},
		{
			name:      "query error",
			token:     "",
			size:      10,
			filters:   map[speaking.Field]any{},
			mockError: fmt.Errorf("query error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create mock: %v", err)
			}
			defer func() {
				_ = db.Close()
			}()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				db:    db,
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			if tt.mockError != nil {
				mock.ExpectQuery("SELECT .* FROM").WillReturnError(tt.mockError)
			} else {
				mock.ExpectQuery("SELECT .* FROM").WillReturnRows(tt.mockRows)
			}

			res, err := h.SpeakingGets(ctx, tt.token, tt.size, tt.filters)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if len(res) != tt.expectLen {
					t.Errorf("expected %d results, got %d", tt.expectLen, len(res))
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil && !tt.expectErr {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func Test_SpeakingUpdate(t *testing.T) {
	speakingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	tests := []struct {
		name      string
		id        uuid.UUID
		fields    map[speaking.Field]any
		mockError error
		expectErr bool
	}{
		{
			name: "normal",
			id:   speakingID,
			fields: map[speaking.Field]any{
				speaking.FieldStatus: speaking.StatusActive,
			},
			expectErr: false,
		},
		{
			name:      "empty fields no-op",
			id:        speakingID,
			fields:    map[speaking.Field]any{},
			expectErr: false,
		},
		{
			name: "database error",
			id:   speakingID,
			fields: map[speaking.Field]any{
				speaking.FieldStatus: speaking.StatusStopped,
			},
			mockError: fmt.Errorf("update error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create mock: %v", err)
			}
			defer func() {
				_ = db.Close()
			}()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				db:    db,
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			if len(tt.fields) > 0 {
				if tt.mockError != nil {
					mock.ExpectExec("UPDATE").WillReturnError(tt.mockError)
				} else {
					mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
				}
			}

			err = h.SpeakingUpdate(ctx, tt.id, tt.fields)
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil && !tt.expectErr {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func Test_SpeakingDelete(t *testing.T) {
	speakingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	tests := []struct {
		name      string
		id        uuid.UUID
		mockError error
		expectErr bool
	}{
		{
			name:      "normal",
			id:        speakingID,
			expectErr: false,
		},
		{
			name:      "database error",
			id:        speakingID,
			mockError: fmt.Errorf("update error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create mock: %v", err)
			}
			defer func() {
				_ = db.Close()
			}()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				db:    db,
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			if tt.mockError != nil {
				mock.ExpectExec("UPDATE").WillReturnError(tt.mockError)
			} else {
				mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err = h.SpeakingDelete(ctx, tt.id)
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil && !tt.expectErr {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func Test_speakingGetFromRow(t *testing.T) {
	speakingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	customerID := uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")
	referenceID := uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13")
	tmCreate := time.Now()

	tests := []struct {
		name      string
		mockRows  *sqlmock.Rows
		expectErr bool
	}{
		{
			name: "normal",
			mockRows: sqlmock.NewRows([]string{
				"id", "customer_id", "reference_type", "reference_id",
				"language", "provider", "voice_id", "direction",
				"status", "pod_id", "tm_create", "tm_update", "tm_delete",
			}).AddRow(
				speakingID.Bytes(), customerID.Bytes(), "call", referenceID.Bytes(),
				"en-US", "elevenlabs", "voice123", "in",
				"active", "pod1", tmCreate, nil, nil,
			),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create mock: %v", err)
			}
			defer func() {
				_ = db.Close()
			}()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				db:    db,
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}

			mock.ExpectQuery("SELECT").WillReturnRows(tt.mockRows)
			rows, err := db.Query("SELECT * FROM test")
			if err != nil {
				t.Fatalf("could not query: %v", err)
			}
			defer func() {
				_ = rows.Close()
			}()

			if rows.Next() {
				res, err := h.speakingGetFromRow(rows)
				if tt.expectErr && err == nil {
					t.Errorf("expected error, got nil")
				}
				if !tt.expectErr && err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !tt.expectErr && res == nil {
					t.Errorf("expected result, got nil")
				}
			}
		})
	}
}
