package cachehandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func setupTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	cache := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	h := &handler{
		Addr:  mr.Addr(),
		Cache: cache,
	}

	return h, mr
}

func TestHandler_ContactSetAndGet(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		FirstName:   "John",
		LastName:    "Doe",
		DisplayName: "John Doe",
		Company:     "Acme Corp",
		JobTitle:    "Engineer",
		Source:      "manual",
		ExternalID:  "ext-123",
		TMCreate:    timePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		TMUpdate:    timePtr(time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)),
		TMDelete:    nil,
	}

	// Test Set
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Test Get
	result, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}

	if result.ID != testContact.ID {
		t.Errorf("ContactGet() ID = %v, want %v", result.ID, testContact.ID)
	}
	if result.FirstName != testContact.FirstName {
		t.Errorf("ContactGet() FirstName = %v, want %v", result.FirstName, testContact.FirstName)
	}
	if result.LastName != testContact.LastName {
		t.Errorf("ContactGet() LastName = %v, want %v", result.LastName, testContact.LastName)
	}
}

func TestHandler_ContactGet_NotFound(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	// Try to get non-existent contact
	_, err := h.ContactGet(ctx, uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"))
	if err == nil {
		t.Error("ContactGet() expected error for non-existent contact")
	}
}

func TestHandler_ContactDelete(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		FirstName: "John",
	}

	// Set contact first
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Verify it exists
	_, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}

	// Delete contact
	if err := h.ContactDelete(ctx, testContact.ID); err != nil {
		t.Errorf("ContactDelete() error = %v", err)
	}

	// Verify it's deleted
	_, err = h.ContactGet(ctx, testContact.ID)
	if err == nil {
		t.Error("ContactGet() expected error after delete")
	}
}

func TestHandler_ContactWithRelatedData(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		FirstName: "John",
		PhoneNumbers: []contact.PhoneNumber{
			{
				ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				ContactID:  uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				Number:     "+1-555-123-4567",
				NumberE164: "+15551234567",
				Type:       "mobile",
				IsPrimary:  true,
			},
		},
		Emails: []contact.Email{
			{
				ID:         uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				ContactID:  uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				Address:    "john@example.com",
				Type:       "work",
				IsPrimary:  true,
			},
		},
		TagIDs: []uuid.UUID{
			uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
		},
	}

	// Set contact
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Get and verify related data is preserved
	result, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}

	if len(result.PhoneNumbers) != 1 {
		t.Errorf("ContactGet() PhoneNumbers count = %v, want 1", len(result.PhoneNumbers))
	}
	if len(result.Emails) != 1 {
		t.Errorf("ContactGet() Emails count = %v, want 1", len(result.Emails))
	}
	if len(result.TagIDs) != 1 {
		t.Errorf("ContactGet() TagIDs count = %v, want 1", len(result.TagIDs))
	}

	if result.PhoneNumbers[0].NumberE164 != "+15551234567" {
		t.Errorf("ContactGet() PhoneNumber = %v, want +15551234567", result.PhoneNumbers[0].NumberE164)
	}
	if result.Emails[0].Address != "john@example.com" {
		t.Errorf("ContactGet() Email = %v, want john@example.com", result.Emails[0].Address)
	}
}

func TestNewHandler(t *testing.T) {
	h := NewHandler("localhost:6379", "password", 1)
	if h == nil {
		t.Error("NewHandler() returned nil")
	}
}

func TestHandler_ContactSet_Error(t *testing.T) {
	// Create a handler with an invalid/closed connection to simulate error
	cache := redis.NewClient(&redis.Options{
		Addr: "invalid:9999",
	})

	h := &handler{
		Addr:  "invalid:9999",
		Cache: cache,
	}

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		FirstName: "Test",
	}

	// Test Set - should fail due to connection error
	err := h.ContactSet(ctx, testContact)
	if err == nil {
		t.Error("ContactSet() expected error for invalid connection")
	}
}

func TestHandler_getSerialize_UnmarshalError(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	// Set invalid JSON directly in Redis
	key := "contact:11111111-1111-1111-1111-111111111111"
	if err := mr.Set(key, "invalid json {{{"); err != nil {
		t.Fatalf("Failed to set test data: %v", err)
	}

	// Try to get - should fail on unmarshal
	_, err := h.ContactGet(ctx, uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"))
	if err == nil {
		t.Error("ContactGet() expected error for invalid JSON")
	}
}

func TestHandler_ContactGet_EmptyResult(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	// Try to get non-existent key
	_, err := h.ContactGet(ctx, uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	if err == nil {
		t.Error("ContactGet() expected error for non-existent key")
	}
}

func TestHandler_ContactDelete_Error(t *testing.T) {
	// Create a handler with an invalid/closed connection
	cache := redis.NewClient(&redis.Options{
		Addr: "invalid:9999",
	})

	h := &handler{
		Addr:  "invalid:9999",
		Cache: cache,
	}

	ctx := context.Background()

	// Test Delete - should fail due to connection error
	err := h.ContactDelete(ctx, uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"))
	if err == nil {
		t.Error("ContactDelete() expected error for invalid connection")
	}
}

func TestHandler_MultipleContacts(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	contacts := []*contact.Contact{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			},
			FirstName: "First",
		},
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			},
			FirstName: "Second",
		},
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			},
			FirstName: "Third",
		},
	}

	// Set all contacts
	for _, c := range contacts {
		if err := h.ContactSet(ctx, c); err != nil {
			t.Errorf("ContactSet() error = %v", err)
		}
	}

	// Get and verify each
	for _, c := range contacts {
		result, err := h.ContactGet(ctx, c.ID)
		if err != nil {
			t.Errorf("ContactGet() error = %v", err)
		}
		if result.FirstName != c.FirstName {
			t.Errorf("ContactGet() FirstName = %v, want %v", result.FirstName, c.FirstName)
		}
	}

	// Delete one and verify
	if err := h.ContactDelete(ctx, contacts[1].ID); err != nil {
		t.Errorf("ContactDelete() error = %v", err)
	}

	// First should still exist
	_, err := h.ContactGet(ctx, contacts[0].ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v for first contact", err)
	}

	// Second should be gone
	_, err = h.ContactGet(ctx, contacts[1].ID)
	if err == nil {
		t.Error("ContactGet() expected error for deleted contact")
	}

	// Third should still exist
	_, err = h.ContactGet(ctx, contacts[2].ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v for third contact", err)
	}
}

func TestHandler_Connect(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	// Test Connect - should succeed with miniredis
	err := h.Connect()
	if err != nil {
		t.Errorf("Connect() error = %v", err)
	}
}

func TestHandler_Connect_Error(t *testing.T) {
	// Create a handler with an invalid address
	cache := redis.NewClient(&redis.Options{
		Addr: "invalid:9999",
	})

	h := &handler{
		Addr:  "invalid:9999",
		Cache: cache,
	}

	// Test Connect - should fail
	err := h.Connect()
	if err == nil {
		t.Error("Connect() expected error for invalid address")
	}
}

func TestHandler_ContactDelete_NonExistent(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	// Delete non-existent contact - should not error
	err := h.ContactDelete(ctx, uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"))
	if err != nil {
		t.Errorf("ContactDelete() error = %v", err)
	}
}

func TestHandler_ContactSet_Overwrite(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		FirstName: "John",
	}

	// Set contact first
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Verify first name is John
	result, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}
	if result.FirstName != "John" {
		t.Errorf("ContactGet() FirstName = %v, want John", result.FirstName)
	}

	// Update the contact
	testContact.FirstName = "Jane"
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Verify first name is now Jane
	result, err = h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}
	if result.FirstName != "Jane" {
		t.Errorf("ContactGet() FirstName = %v, want Jane", result.FirstName)
	}
}

func TestHandler_ContactGet_AllFields(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		FirstName:   "John",
		LastName:    "Doe",
		DisplayName: "John Doe",
		Company:     "Acme Corp",
		JobTitle:    "Engineer",
		Source:      "manual",
		ExternalID:  "ext-123",
		Notes:       "Some notes here",
		TMCreate:    timePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		TMUpdate:    timePtr(time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)),
		TMDelete:    nil,
	}

	// Set contact
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Get and verify all fields
	result, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}

	if result.DisplayName != testContact.DisplayName {
		t.Errorf("ContactGet() DisplayName = %v, want %v", result.DisplayName, testContact.DisplayName)
	}
	if result.Company != testContact.Company {
		t.Errorf("ContactGet() Company = %v, want %v", result.Company, testContact.Company)
	}
	if result.JobTitle != testContact.JobTitle {
		t.Errorf("ContactGet() JobTitle = %v, want %v", result.JobTitle, testContact.JobTitle)
	}
	if result.Source != testContact.Source {
		t.Errorf("ContactGet() Source = %v, want %v", result.Source, testContact.Source)
	}
	if result.ExternalID != testContact.ExternalID {
		t.Errorf("ContactGet() ExternalID = %v, want %v", result.ExternalID, testContact.ExternalID)
	}
	if result.Notes != testContact.Notes {
		t.Errorf("ContactGet() Notes = %v, want %v", result.Notes, testContact.Notes)
	}
	if result.CustomerID != testContact.CustomerID {
		t.Errorf("ContactGet() CustomerID = %v, want %v", result.CustomerID, testContact.CustomerID)
	}
}

func TestHandler_ContactWithEmptyPhoneNumbers(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			CustomerID: uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"),
		},
		FirstName:    "Empty",
		LastName:     "PhoneNumbers",
		PhoneNumbers: []contact.PhoneNumber{}, // Empty slice
		Emails:       []contact.Email{},       // Empty slice
		TagIDs:       []uuid.UUID{},           // Empty slice
	}

	// Set contact
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Get and verify empty slices are preserved
	result, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}

	if result.FirstName != testContact.FirstName {
		t.Errorf("ContactGet() FirstName = %v, want %v", result.FirstName, testContact.FirstName)
	}
}

func TestHandler_ContactWithNilFields(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("bbbbbbbb-cccc-dddd-eeee-ffffffffffff"),
			CustomerID: uuid.FromStringOrNil("22222222-3333-4444-5555-666666666666"),
		},
		FirstName: "Nil",
		LastName:  "Fields",
		// PhoneNumbers, Emails, TagIDs are nil
	}

	// Set contact
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Get and verify
	result, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}

	if result.FirstName != testContact.FirstName {
		t.Errorf("ContactGet() FirstName = %v, want %v", result.FirstName, testContact.FirstName)
	}
}

func TestHandler_ContactSetUpdateExisting(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-111111111111"),
			CustomerID: uuid.FromStringOrNil("33333333-4444-5555-6666-777777777777"),
		},
		FirstName: "Initial",
		LastName:  "Contact",
	}

	// Set contact initially
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Update the same contact
	testContact.FirstName = "Updated"
	testContact.PhoneNumbers = []contact.PhoneNumber{
		{
			ID:         uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-222222222222"),
			Number:     "+1-555-999-0000",
			NumberE164: "+15559990000",
		},
	}

	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Verify update
	result, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}

	if result.FirstName != "Updated" {
		t.Errorf("ContactGet() FirstName = %v, want Updated", result.FirstName)
	}
	if len(result.PhoneNumbers) != 1 {
		t.Errorf("ContactGet() PhoneNumbers count = %v, want 1", len(result.PhoneNumbers))
	}
}

func TestHandler_ContactGet_ConnectionClosed(t *testing.T) {
	h, mr := setupTestHandler(t)

	ctx := context.Background()

	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("eeeeeeee-ffff-0000-1111-333333333333"),
			CustomerID: uuid.FromStringOrNil("44444444-5555-6666-7777-888888888888"),
		},
		FirstName: "Test",
	}

	// Set contact
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Close miniredis to simulate connection loss
	mr.Close()

	// Try to get - should fail
	_, err := h.ContactGet(ctx, testContact.ID)
	if err == nil {
		t.Error("ContactGet() expected error after connection closed")
	}
}

func TestHandler_ContactDelete_AfterConnectionClosed(t *testing.T) {
	h, mr := setupTestHandler(t)

	ctx := context.Background()

	// Close miniredis to simulate connection loss
	mr.Close()

	// Try to delete - should fail
	err := h.ContactDelete(ctx, uuid.FromStringOrNil("ffffffff-0000-1111-2222-444444444444"))
	if err == nil {
		t.Error("ContactDelete() expected error after connection closed")
	}
}

func TestHandler_ContactWithLargeData(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	// Create a contact with multiple phone numbers and emails
	testContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("00001111-2222-3333-4444-555566667777"),
			CustomerID: uuid.FromStringOrNil("88889999-aaaa-bbbb-cccc-ddddeeeeffff"),
		},
		FirstName:   "Large",
		LastName:    "Data",
		DisplayName: "Large Data Contact",
		Company:     "Large Corp",
		JobTitle:    "Data Manager",
		Source:      "api",
		ExternalID:  "large-ext-001",
		Notes:       "This is a contact with lots of related data for testing serialization",
		PhoneNumbers: []contact.PhoneNumber{
			{ID: uuid.FromStringOrNil("a0001111-1111-1111-1111-111111111111"), Number: "+1-555-111-1111", NumberE164: "+15551111111", Type: "mobile", IsPrimary: true},
			{ID: uuid.FromStringOrNil("a0002222-2222-2222-2222-222222222222"), Number: "+1-555-222-2222", NumberE164: "+15552222222", Type: "work", IsPrimary: false},
			{ID: uuid.FromStringOrNil("a0003333-3333-3333-3333-333333333333"), Number: "+1-555-333-3333", NumberE164: "+15553333333", Type: "home", IsPrimary: false},
		},
		Emails: []contact.Email{
			{ID: uuid.FromStringOrNil("b0001111-1111-1111-1111-111111111111"), Address: "primary@example.com", Type: "work", IsPrimary: true},
			{ID: uuid.FromStringOrNil("b0002222-2222-2222-2222-222222222222"), Address: "secondary@example.com", Type: "personal", IsPrimary: false},
		},
		TagIDs: []uuid.UUID{
			uuid.FromStringOrNil("c0001111-1111-1111-1111-111111111111"),
			uuid.FromStringOrNil("c0002222-2222-2222-2222-222222222222"),
			uuid.FromStringOrNil("c0003333-3333-3333-3333-333333333333"),
		},
		TMCreate: timePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		TMUpdate: timePtr(time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)),
		TMDelete: nil,
	}

	// Set contact
	if err := h.ContactSet(ctx, testContact); err != nil {
		t.Errorf("ContactSet() error = %v", err)
	}

	// Get and verify all data is preserved
	result, err := h.ContactGet(ctx, testContact.ID)
	if err != nil {
		t.Errorf("ContactGet() error = %v", err)
	}

	if len(result.PhoneNumbers) != 3 {
		t.Errorf("ContactGet() PhoneNumbers count = %v, want 3", len(result.PhoneNumbers))
	}
	if len(result.Emails) != 2 {
		t.Errorf("ContactGet() Emails count = %v, want 2", len(result.Emails))
	}
	if len(result.TagIDs) != 3 {
		t.Errorf("ContactGet() TagIDs count = %v, want 3", len(result.TagIDs))
	}
}
