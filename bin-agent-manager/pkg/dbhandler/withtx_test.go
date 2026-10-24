package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func Test_withTx(t *testing.T) {
	errCallback := errors.New("simulated callback failure")
	errBegin := errors.New("simulated begin failure")
	errCommit := errors.New("simulated commit failure")

	t.Run("commit on success", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		called := false
		err := h.withTx(context.Background(), func(tx *sql.Tx) error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if !called {
			t.Errorf("Wrong match. expect the callback to run")
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unmet expectations: %v", errMet)
		}
	})

	t.Run("begin failure is wrapped and the callback does not run", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectBegin().WillReturnError(errBegin)

		called := false
		err := h.withTx(context.Background(), func(tx *sql.Tx) error {
			called = true
			return nil
		})
		if err == nil || !strings.HasPrefix(err.Error(), "could not begin transaction. err: ") {
			t.Errorf("Wrong match. expect the begin error, got: %v", err)
		}
		if !errors.Is(err, errBegin) {
			t.Errorf("Wrong match. expect the cause to be kept in the chain, got: %v", err)
		}
		if called {
			t.Errorf("Wrong match. the callback must not run when the transaction did not begin")
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unmet expectations: %v", errMet)
		}
	})

	t.Run("callback failure is returned as it is and rolled back without commit", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		err := h.withTx(context.Background(), func(tx *sql.Tx) error {
			return errCallback
		})
		if err != errCallback {
			t.Errorf("Wrong match. expect the callback error unchanged, got: %v", err)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unmet expectations: %v", errMet)
		}
	})

	t.Run("commit failure is wrapped", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(errCommit)

		err := h.withTx(context.Background(), func(tx *sql.Tx) error { return nil })
		if err == nil || !strings.HasPrefix(err.Error(), "could not commit transaction. err: ") {
			t.Errorf("Wrong match. expect the commit error, got: %v", err)
		}
		if !errors.Is(err, errCommit) {
			t.Errorf("Wrong match. expect the cause to be kept in the chain, got: %v", err)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unmet expectations: %v", errMet)
		}
	})

	t.Run("panic rolls back and is raised again", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = h.withTx(context.Background(), func(tx *sql.Tx) error {
				panic("simulated panic")
			})
		}()

		if recovered != "simulated panic" {
			t.Errorf("Wrong match. expect the original panic value, got: %v", recovered)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unmet expectations: %v", errMet)
		}
	})
}
