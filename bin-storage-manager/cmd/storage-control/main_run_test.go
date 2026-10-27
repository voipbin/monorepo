package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/spf13/viper"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-storage-manager/internal/config"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
)

const (
	testAccountID  = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testCustomerID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	testFile1ID    = "f1f1f1f1-f1f1-4f1f-8f1f-f1f1f1f1f1f1"
	testFile2ID    = "f2f2f2f2-f2f2-4f2f-8f2f-f2f2f2f2f2f2"
	testFile3ID    = "f3f3f3f3-f3f3-4f3f-8f3f-f3f3f3f3f3f3"
	testRefID      = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
)

func seedAccount(t *testing.T, env *testEnv) {
	t.Helper()

	h := env.seedHandler(t)
	a := &account.Account{ID: uuid.FromStringOrNil(testAccountID), CustomerID: uuid.FromStringOrNil(testCustomerID), TotalFileCount: 2, TotalFileSize: 300}
	if err := h.AccountCreate(context.Background(), a); err != nil {
		t.Fatalf("could not seed the account. err: %v", err)
	}
}

func seedFile(t *testing.T, env *testEnv, id string, refType file.ReferenceType, refID string, size int64) {
	t.Helper()

	h := env.seedHandler(t)
	f := &file.File{
		Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil(id), CustomerID: uuid.FromStringOrNil(testCustomerID)},
		AccountID:     uuid.FromStringOrNil(testAccountID),
		ReferenceType: refType,
		ReferenceID:   uuid.FromStringOrNil(refID),
		Name:          "name-" + id[:2],
		BucketName:    "test-media",
		Filepath:      "path/" + id,
		Filesize:      size,
	}
	if err := h.FileCreate(context.Background(), f); err != nil {
		t.Fatalf("could not seed the file. err: %v", err)
	}
}

// runCapture calls the subcommand body and returns its standard output and error.
func runCapture(t *testing.T, run func() error) (string, error) {
	t.Helper()

	var err error
	out := captureStdout(t, func() { err = run() })

	return out, err
}

func Test_initCache(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		newTestEnv(t)

		res, err := initCache()
		if err != nil || res == nil {
			t.Fatalf("Wrong match. expect: ok, got: %v, %v", res, err)
		}
	})

	t.Run("server down", func(t *testing.T) {
		env := newTestEnv(t)
		env.redis.Close()

		res, err := initCache()
		if err == nil || !strings.HasPrefix(err.Error(), "could not connect to the cache") {
			t.Fatalf("Wrong match. expect prefix: could not connect to the cache, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil handler, got: %v", res)
		}
	})
}

func Test_initHandlers(t *testing.T) {
	t.Run("storage handler success", func(t *testing.T) {
		newTestEnv(t)

		res, err := initStorageHandler()
		if err != nil || res == nil {
			t.Fatalf("Wrong match. expect: ok, got: %v, %v", res, err)
		}
	})

	t.Run("account handler success", func(t *testing.T) {
		newTestEnv(t)

		res, err := initAccountHandler()
		if err != nil || res == nil {
			t.Fatalf("Wrong match. expect: ok, got: %v, %v", res, err)
		}
	})

	t.Run("cache down", func(t *testing.T) {
		env := newTestEnv(t)
		env.redis.Close()

		if _, err := initStorageHandler(); err == nil || !strings.HasPrefix(err.Error(), "could not initialize the cache: could not connect to the cache") {
			t.Errorf("Wrong match. initStorageHandler, got: %v", err)
		}
		if _, err := initAccountHandler(); err == nil || !strings.HasPrefix(err.Error(), "could not initialize the cache: could not connect to the cache") {
			t.Errorf("Wrong match. initAccountHandler, got: %v", err)
		}
	})

	t.Run("unreadable credential file", func(t *testing.T) {
		newTestEnv(t)
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", t.TempDir()+"/missing.json")

		res, err := initStorageHandler()
		if err == nil || !strings.HasPrefix(err.Error(), "could not create the file handler") {
			t.Fatalf("Wrong match. expect prefix: could not create the file handler, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil handler, got: %v", res)
		}
	})
}

func Test_runFileGet(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		env := newTestEnv(t)
		seedAccount(t, env)
		seedFile(t, env, testFile1ID, file.ReferenceTypeNormal, testRefID, 100)
		viper.Set("id", testFile1ID)

		out, err := runCapture(t, func() error { return runFileGet(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		var got file.File
		if errJSON := json.Unmarshal([]byte(out), &got); errJSON != nil {
			t.Fatalf("Wrong match. output is not json: %v, out: %s", errJSON, out)
		}
		if got.ID.String() != testFile1ID || got.Name != "name-f1" || got.Filesize != 100 || got.CustomerID.String() != testCustomerID {
			t.Errorf("Wrong match. got: %+v", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		newTestEnv(t)
		viper.Set("id", testFile1ID)

		out, err := runCapture(t, func() error { return runFileGet(nil, nil) })
		if err == nil || !strings.HasPrefix(err.Error(), "failed to retrieve file") {
			t.Errorf("Wrong match. expect prefix: failed to retrieve file, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect: no output, got: %s", out)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		newTestEnv(t)
		viper.Set("id", "not-a-uuid")

		if err := runFileGet(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to resolve file ID") {
			t.Errorf("Wrong match. expect prefix: failed to resolve file ID, got: %v", err)
		}
	})
}

func Test_runFileList(t *testing.T) {
	tests := []struct {
		name       string
		refID      string
		refType    string
		expectedID []string
	}{
		{"all of the customer", "", "", []string{testFile1ID, testFile2ID}},
		{"by reference id", testRefID, "", []string{testFile1ID}},
		{"by reference type", "", "recording", []string{testFile2ID}},
		{"nothing matches", testRefID, "recording", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			seedAccount(t, env)
			seedFile(t, env, testFile1ID, file.ReferenceTypeNormal, testRefID, 100)
			seedFile(t, env, testFile2ID, file.ReferenceTypeRecording, testUUID3, 200)
			// a deleted file of the same customer is never listed
			seedFile(t, env, testFile3ID, file.ReferenceTypeNormal, testRefID, 300)
			if errDel := env.seedHandler(t).FileDelete(context.Background(), uuid.FromStringOrNil(testFile3ID)); errDel != nil {
				t.Fatalf("could not delete the seeded file. err: %v", errDel)
			}
			viper.Set("customer-id", testCustomerID)
			viper.Set("limit", 10)
			if tt.refID != "" {
				viper.Set("reference-id", tt.refID)
			}
			if tt.refType != "" {
				viper.Set("reference-type", tt.refType)
			}

			out, err := runCapture(t, func() error { return runFileList(nil, nil) })
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			var got []*file.File
			if errJSON := json.Unmarshal([]byte(out), &got); errJSON != nil {
				t.Fatalf("Wrong match. output is not json: %v, out: %s", errJSON, out)
			}

			ids := map[string]bool{}
			for _, f := range got {
				ids[f.ID.String()] = true
			}
			if len(got) != len(tt.expectedID) {
				t.Fatalf("Wrong match. expect: %v, got: %d files", tt.expectedID, len(got))
			}
			for _, id := range tt.expectedID {
				if !ids[id] {
					t.Errorf("Wrong match. expect file: %s, got: %v", id, ids)
				}
			}
		})
	}

	t.Run("database failure", func(t *testing.T) {
		env := newTestEnv(t)
		viper.Set("customer-id", testCustomerID)
		_ = env.db.Close()

		if err := runFileList(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to retrieve files") {
			t.Errorf("Wrong match. expect prefix: failed to retrieve files, got: %v", err)
		}
	})

	t.Run("invalid customer id", func(t *testing.T) {
		newTestEnv(t)

		if err := runFileList(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to resolve customer ID") {
			t.Errorf("Wrong match. expect prefix: failed to resolve customer ID, got: %v", err)
		}
	})
}

func Test_runFileDelete(t *testing.T) {
	t.Run("deleted", func(t *testing.T) {
		env := newTestEnv(t)
		seedAccount(t, env)
		seedFile(t, env, testFile1ID, file.ReferenceTypeNormal, testRefID, 100)
		viper.Set("id", testFile1ID)

		out, err := runCapture(t, func() error { return runFileDelete(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		var got file.File
		if errJSON := json.Unmarshal([]byte(out), &got); errJSON != nil {
			t.Fatalf("Wrong match. output is not json: %v, out: %s", errJSON, out)
		}
		if got.ID.String() != testFile1ID || got.TMDelete == nil {
			t.Errorf("Wrong match. expect the deleted file %s, got: %+v", testFile1ID, got)
		}

		// the account counters were decreased by the deleted file
		acc, errAcc := env.seedHandler(t).AccountGet(context.Background(), uuid.FromStringOrNil(testAccountID))
		if errAcc != nil {
			t.Fatalf("Wrong match. could not read the account: %v", errAcc)
		}
		if acc.TotalFileCount != 1 || acc.TotalFileSize != 200 {
			t.Errorf("Wrong match. expect count 1 and size 200, got count %d and size %d", acc.TotalFileCount, acc.TotalFileSize)
		}
	})

	t.Run("not found", func(t *testing.T) {
		newTestEnv(t)
		viper.Set("id", testFile1ID)

		if err := runFileDelete(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to delete file") {
			t.Errorf("Wrong match. expect prefix: failed to delete file, got: %v", err)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		newTestEnv(t)

		if err := runFileDelete(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to resolve file ID") {
			t.Errorf("Wrong match. expect prefix: failed to resolve file ID, got: %v", err)
		}
	})
}

func Test_runAccount(t *testing.T) {
	t.Run("create then get then list then delete", func(t *testing.T) {
		newTestEnv(t)
		viper.Set("customer-id", testCustomerID)

		out, err := runCapture(t, func() error { return runAccountCreate(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. create expect: ok, got: %v", err)
		}
		var created account.Account
		if errJSON := json.Unmarshal([]byte(out), &created); errJSON != nil {
			t.Fatalf("Wrong match. output is not json: %v, out: %s", errJSON, out)
		}
		if created.CustomerID.String() != testCustomerID || created.ID == uuid.Nil || created.TotalFileCount != 0 || created.TotalFileSize != 0 {
			t.Fatalf("Wrong match. got: %+v", created)
		}

		// a second account for the same customer is refused
		if errDup := runAccountCreate(nil, nil); errDup == nil || !strings.HasPrefix(errDup.Error(), "failed to create account") {
			t.Errorf("Wrong match. duplicate expect prefix: failed to create account, got: %v", errDup)
		}

		viper.Set("id", created.ID.String())
		out, err = runCapture(t, func() error { return runAccountGet(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. get expect: ok, got: %v", err)
		}
		var got account.Account
		if errJSON := json.Unmarshal([]byte(out), &got); errJSON != nil {
			t.Fatalf("Wrong match. output is not json: %v, out: %s", errJSON, out)
		}
		if got.ID != created.ID || got.CustomerID != created.CustomerID {
			t.Errorf("Wrong match. expect: %+v, got: %+v", created, got)
		}

		viper.Set("limit", 10)
		out, err = runCapture(t, func() error { return runAccountList(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. list expect: ok, got: %v", err)
		}
		var list []*account.Account
		if errJSON := json.Unmarshal([]byte(out), &list); errJSON != nil {
			t.Fatalf("Wrong match. output is not json: %v, out: %s", errJSON, out)
		}
		if len(list) != 1 || list[0].ID != created.ID {
			t.Errorf("Wrong match. expect one account %s, got: %d", created.ID, len(list))
		}

		out, err = runCapture(t, func() error { return runAccountDelete(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. delete expect: ok, got: %v", err)
		}
		var deleted account.Account
		if errJSON := json.Unmarshal([]byte(out), &deleted); errJSON != nil {
			t.Fatalf("Wrong match. output is not json: %v, out: %s", errJSON, out)
		}
		if deleted.ID != created.ID || deleted.TMDelete == nil {
			t.Errorf("Wrong match. expect the deleted account %s, got: %+v", created.ID, deleted)
		}

		// a deleted account is not listed any more
		out, err = runCapture(t, func() error { return runAccountList(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. list expect: ok, got: %v", err)
		}
		list = nil
		if errJSON := json.Unmarshal([]byte(out), &list); errJSON != nil {
			t.Fatalf("Wrong match. output is not json: %v, out: %s", errJSON, out)
		}
		if len(list) != 0 {
			t.Errorf("Wrong match. expect no account, got: %d", len(list))
		}
	})

	t.Run("not found", func(t *testing.T) {
		newTestEnv(t)
		viper.Set("id", testAccountID)

		if err := runAccountGet(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to retrieve account") {
			t.Errorf("Wrong match. get expect prefix: failed to retrieve account, got: %v", err)
		}
		if err := runAccountDelete(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to delete account") {
			t.Errorf("Wrong match. delete expect prefix: failed to delete account, got: %v", err)
		}
	})

	t.Run("invalid ids", func(t *testing.T) {
		newTestEnv(t)

		if err := runAccountGet(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to resolve account ID") {
			t.Errorf("Wrong match. get, got: %v", err)
		}
		if err := runAccountDelete(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to resolve account ID") {
			t.Errorf("Wrong match. delete, got: %v", err)
		}
		if err := runAccountList(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to resolve customer ID") {
			t.Errorf("Wrong match. list, got: %v", err)
		}
	})

	t.Run("database failure", func(t *testing.T) {
		env := newTestEnv(t)
		viper.Set("customer-id", testCustomerID)
		_ = env.db.Close()

		if err := runAccountList(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to retrieve accounts") {
			t.Errorf("Wrong match. expect prefix: failed to retrieve accounts, got: %v", err)
		}
	})
}

func Test_runRecording(t *testing.T) {
	t.Run("delete removes every file of the reference", func(t *testing.T) {
		env := newTestEnv(t)
		seedAccount(t, env)
		seedFile(t, env, testFile1ID, file.ReferenceTypeRecording, testRefID, 100)
		seedFile(t, env, testFile2ID, file.ReferenceTypeRecording, testRefID, 200)
		viper.Set("reference-id", testRefID)

		out, err := runCapture(t, func() error { return runRecordingDelete(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if strings.TrimSpace(out) != `{"status": "deleted"}` {
			t.Errorf("Wrong match. expect: {\"status\": \"deleted\"}, got: %s", out)
		}

		remaining, errList := env.seedHandler(t).FileList(context.Background(), "", 10, map[file.Field]any{file.FieldDeleted: false, file.FieldReferenceID: uuid.FromStringOrNil(testRefID)})
		if errList != nil {
			t.Fatalf("Wrong match. could not list the files: %v", errList)
		}
		if len(remaining) != 0 {
			t.Errorf("Wrong match. expect no file left, got: %d", len(remaining))
		}
	})

	t.Run("delete without files", func(t *testing.T) {
		newTestEnv(t)
		viper.Set("reference-id", testRefID)

		out, err := runCapture(t, func() error { return runRecordingDelete(nil, nil) })
		if err != nil || strings.TrimSpace(out) != `{"status": "deleted"}` {
			t.Errorf("Wrong match. expect: ok and status deleted, got: %v, %s", err, out)
		}
	})

	t.Run("database failure", func(t *testing.T) {
		env := newTestEnv(t)
		viper.Set("reference-id", testRefID)
		_ = env.db.Close()

		if err := runRecordingDelete(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to delete recording") {
			t.Errorf("Wrong match. delete expect prefix: failed to delete recording, got: %v", err)
		}
		if err := runRecordingGet(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to retrieve recording") {
			t.Errorf("Wrong match. get expect prefix: failed to retrieve recording, got: %v", err)
		}
	})

	t.Run("invalid reference id", func(t *testing.T) {
		newTestEnv(t)

		if err := runRecordingDelete(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to resolve reference ID") {
			t.Errorf("Wrong match. delete, got: %v", err)
		}
		if err := runRecordingGet(nil, nil); err == nil || !strings.HasPrefix(err.Error(), "failed to resolve reference ID") {
			t.Errorf("Wrong match. get, got: %v", err)
		}
	})
}

func Test_newTestEnv_restoresConfig(t *testing.T) {
	before := *config.Get()
	t.Run("inner", func(t *testing.T) {
		newTestEnv(t)
		if config.Get().GCPBucketNameMedia != "test-media" {
			t.Errorf("Wrong match. expect: test-media, got: %s", config.Get().GCPBucketNameMedia)
		}
	})
	if *config.Get() != before {
		t.Errorf("Wrong match. the configuration was not restored.\nexpect: %+v\ngot: %+v", before, *config.Get())
	}
}
