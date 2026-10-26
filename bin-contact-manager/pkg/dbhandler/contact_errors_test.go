package dbhandler

// Error path tests for the contact, tag assignment and case note queries.
// They use sqlmock only, so they do not touch the shared sqlite database.

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
)

var (
	errTestCustomerID = uuid.FromStringOrNil("d8000000-0000-0000-0000-000000000001")
	errTestContactID  = uuid.FromStringOrNil("d8000000-0000-0000-0000-000000000002")
)

// badUUIDRow is a one column row whose value is not a 16 byte uuid.
func badUUIDRow(column string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{column}).AddRow([]byte("short"))
}

// twoColumnRow makes a scan into a single destination fail.
func twoColumnRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"a", "b"}).AddRow([]byte("x"), []byte("y"))
}

func Test_Contact_QueryFailures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func(mock sqlmock.Sqlmock)
		call     func(h *handler) error
		wantText string
		wantIs   error
	}{
		{
			name: "ContactList query failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				_, err := h.ContactList(ctx, 10, "2026-01-01 00:00:00.000000", map[contact.Field]any{contact.FieldCustomerID: errTestCustomerID})
				return err
			},
			wantText: "could not query. ContactList",
			wantIs:   errCaseBoom,
		},
		{
			name: "ContactList scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(twoColumnRow())
			},
			call: func(h *handler) error {
				_, err := h.ContactList(ctx, 10, "2026-01-01 00:00:00.000000", nil)
				return err
			},
			wantText: "could not scan the row. ContactList",
		},
		{
			name: "ContactList row iteration failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCaseBoom))
			},
			call: func(h *handler) error {
				_, err := h.ContactList(ctx, 10, "2026-01-01 00:00:00.000000", nil)
				return err
			},
			wantText: "row iteration error. ContactList",
			wantIs:   errCaseBoom,
		},
		{
			name: "contactGetFromDB scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(twoColumnRow())
			},
			call: func(h *handler) error {
				_, err := h.contactGetFromDB(ctx, errTestContactID)
				return err
			},
			wantText: "could not scan the row. contactGetFromDB",
		},
		{
			name: "ContactUpdate exec failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_contacts").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				return h.ContactUpdate(ctx, errTestContactID, map[contact.Field]any{contact.FieldFirstName: "Alice"})
			},
			wantText: "could not execute. ContactUpdate",
			wantIs:   errCaseBoom,
		},
		{
			name: "ContactDelete exec failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_contacts").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				return h.ContactDelete(ctx, errTestContactID)
			},
			wantText: "could not execute. ContactDelete",
			wantIs:   errCaseBoom,
		},
		{
			name: "AddressLookupContactIDByTypeTarget query failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				_, err := h.AddressLookupContactIDByTypeTarget(ctx, errTestCustomerID, commonaddress.TypeTel, "+15550000001")
				return err
			},
			wantText: "could not query. AddressLookupContactIDByTypeTarget",
			wantIs:   errCaseBoom,
		},
		{
			name: "AddressLookupContactIDByTypeTarget scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnRows(twoColumnRow())
			},
			call: func(h *handler) error {
				_, err := h.AddressLookupContactIDByTypeTarget(ctx, errTestCustomerID, commonaddress.TypeTel, "+15550000001")
				return err
			},
			wantText: "could not scan contact_id. AddressLookupContactIDByTypeTarget",
		},
		{
			name: "AddressLookupContactIDByTypeTarget parse failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnRows(badUUIDRow("contact_id"))
			},
			call: func(h *handler) error {
				_, err := h.AddressLookupContactIDByTypeTarget(ctx, errTestCustomerID, commonaddress.TypeTel, "+15550000001")
				return err
			},
			wantText: "could not parse contact_id. AddressLookupContactIDByTypeTarget",
		},
		{
			name: "ContactLookupByPhone query failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				_, err := h.ContactLookupByPhone(ctx, errTestCustomerID, "+15550000001")
				return err
			},
			wantText: "could not query. ContactLookupByPhone",
			wantIs:   errCaseBoom,
		},
		{
			name: "ContactLookupByPhone scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnRows(twoColumnRow())
			},
			call: func(h *handler) error {
				_, err := h.ContactLookupByPhone(ctx, errTestCustomerID, "+15550000001")
				return err
			},
			wantText: "could not scan contact_id. ContactLookupByPhone",
		},
		{
			name: "ContactLookupByPhone parse failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnRows(badUUIDRow("contact_id"))
			},
			call: func(h *handler) error {
				_, err := h.ContactLookupByPhone(ctx, errTestCustomerID, "+15550000001")
				return err
			},
			wantText: "could not parse contact_id. ContactLookupByPhone",
		},
		{
			name: "ContactLookupByEmail query failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				_, err := h.ContactLookupByEmail(ctx, errTestCustomerID, "user@example.com")
				return err
			},
			wantText: "could not query. ContactLookupByEmail",
			wantIs:   errCaseBoom,
		},
		{
			name: "ContactLookupByEmail scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnRows(twoColumnRow())
			},
			call: func(h *handler) error {
				_, err := h.ContactLookupByEmail(ctx, errTestCustomerID, "user@example.com")
				return err
			},
			wantText: "could not scan contact_id. ContactLookupByEmail",
		},
		{
			name: "ContactLookupByEmail parse failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT contact_id").WillReturnRows(badUUIDRow("contact_id"))
			},
			call: func(h *handler) error {
				_, err := h.ContactLookupByEmail(ctx, errTestCustomerID, "user@example.com")
				return err
			},
			wantText: "could not parse contact_id. ContactLookupByEmail",
		},
		{
			name: "ContactDeleteByCustomerID query failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM contact_contacts").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				return h.ContactDeleteByCustomerID(ctx, errTestCustomerID)
			},
			wantText: "could not query contact ids. ContactDeleteByCustomerID",
			wantIs:   errCaseBoom,
		},
		{
			name: "ContactDeleteByCustomerID scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM contact_contacts").WillReturnRows(twoColumnRow())
			},
			call: func(h *handler) error {
				return h.ContactDeleteByCustomerID(ctx, errTestCustomerID)
			},
			wantText: "could not scan contact id. ContactDeleteByCustomerID",
		},
		{
			name: "ContactDeleteByCustomerID parse failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM contact_contacts").WillReturnRows(badUUIDRow("id"))
			},
			call: func(h *handler) error {
				return h.ContactDeleteByCustomerID(ctx, errTestCustomerID)
			},
			wantText: "could not parse contact id. ContactDeleteByCustomerID",
		},
		{
			name: "ContactDeleteByCustomerID row iteration failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM contact_contacts").WillReturnRows(rowErrRows(errCaseBoom))
			},
			call: func(h *handler) error {
				return h.ContactDeleteByCustomerID(ctx, errTestCustomerID)
			},
			wantText: "row iteration error. ContactDeleteByCustomerID",
			wantIs:   errCaseBoom,
		},
		{
			name: "ContactDeleteByCustomerID exec failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM contact_contacts").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(errTestContactID.Bytes()))
				mock.ExpectExec("UPDATE contact_contacts").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				return h.ContactDeleteByCustomerID(ctx, errTestCustomerID)
			},
			wantText: "could not execute. ContactDeleteByCustomerID",
			wantIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tt.setup(mock)

			assertWrappedErr(t, tt.call(h), tt.wantText, tt.wantIs)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func Test_TagAssignment_QueryFailures(t *testing.T) {
	ctx := context.Background()
	tagID := uuid.FromStringOrNil("d8000000-0000-0000-0000-000000000003")

	tests := []struct {
		name     string
		setup    func(mock sqlmock.Sqlmock)
		call     func(h *handler) error
		wantText string
		wantIs   error
	}{
		{
			name: "TagAssignmentCreate exec failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO contact_tag_assignments").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				return h.TagAssignmentCreate(ctx, errTestContactID, tagID)
			},
			wantText: "could not execute. TagAssignmentCreate",
			wantIs:   errCaseBoom,
		},
		{
			name: "TagAssignmentDelete exec failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM contact_tag_assignments").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				return h.TagAssignmentDelete(ctx, errTestContactID, tagID)
			},
			wantText: "could not execute. TagAssignmentDelete",
			wantIs:   errCaseBoom,
		},
		{
			name: "TagAssignmentListByContactID scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT tag_id").WillReturnRows(twoColumnRow())
			},
			call: func(h *handler) error {
				_, err := h.TagAssignmentListByContactID(ctx, errTestContactID)
				return err
			},
			wantText: "could not scan tag_id. TagAssignmentListByContactID",
		},
		{
			name: "TagAssignmentListByContactID parse failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT tag_id").WillReturnRows(badUUIDRow("tag_id"))
			},
			call: func(h *handler) error {
				_, err := h.TagAssignmentListByContactID(ctx, errTestContactID)
				return err
			},
			wantText: "could not parse tag_id. TagAssignmentListByContactID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tt.setup(mock)

			assertWrappedErr(t, tt.call(h), tt.wantText, tt.wantIs)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

// columnsOf is a sanity helper: a sqlmock row of nil values for the given
// column names, used where only the column count matters.
func columnsOf(names ...string) *sqlmock.Rows {
	vals := make([]driver.Value, len(names))
	return sqlmock.NewRows(names).AddRow(vals...)
}

func Test_CaseNote_QueryFailures(t *testing.T) {
	ctx := context.Background()
	caseID := uuid.FromStringOrNil("d8000000-0000-0000-0000-000000000004")
	noteID := uuid.FromStringOrNil("d8000000-0000-0000-0000-000000000005")

	tests := []struct {
		name     string
		setup    func(mock sqlmock.Sqlmock)
		call     func(h *handler) error
		wantText string
		wantIs   error
	}{
		{
			name: "CaseNoteDelete exec failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_case_notes").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				return h.CaseNoteDelete(ctx, errTestCustomerID, caseID, noteID)
			},
			wantText: "could not execute. CaseNoteDelete",
			wantIs:   errCaseBoom,
		},
		{
			name: "CaseNoteDelete rows affected failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_case_notes").WillReturnResult(sqlmock.NewErrorResult(errCaseBoom))
			},
			call: func(h *handler) error {
				return h.CaseNoteDelete(ctx, errTestCustomerID, caseID, noteID)
			},
			wantText: "could not get rows affected. CaseNoteDelete",
			wantIs:   errCaseBoom,
		},
		{
			name: "CaseNoteListByCase query failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				_, err := h.CaseNoteListByCase(ctx, errTestCustomerID, caseID)
				return err
			},
			wantText: "could not query. CaseNoteListByCase",
			wantIs:   errCaseBoom,
		},
		{
			name: "CaseNoteListByCase scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(columnsOf("only_one"))
			},
			call: func(h *handler) error {
				_, err := h.CaseNoteListByCase(ctx, errTestCustomerID, caseID)
				return err
			},
			wantText: "could not scan the row. CaseNoteListByCase",
		},
		{
			name: "CaseNoteListByCase row iteration failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCaseBoom))
			},
			call: func(h *handler) error {
				_, err := h.CaseNoteListByCase(ctx, errTestCustomerID, caseID)
				return err
			},
			wantText: "row iteration error. CaseNoteListByCase",
			wantIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tt.setup(mock)

			assertWrappedErr(t, tt.call(h), tt.wantText, tt.wantIs)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}
