package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/spf13/viper"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

var errInjectedHelperCase = errors.New("injected failure")

// helperCaseDomains builds count synthetic customer domain rows whose ids are derived
// from the offset, in ascending tm_create order.
func helperCaseDomains(offset int, count int, last *time.Time) []*customerdomain.CustomerDomain {
	res := make([]*customerdomain.CustomerDomain, 0, count)
	for i := 0; i < count; i++ {
		res = append(res, &customerdomain.CustomerDomain{
			CustomerID: uuid.FromStringOrNil(fmt.Sprintf("00000000-0000-4000-8000-%012d", offset+i)),
		})
	}
	if count > 0 {
		res[count-1].TMCreate = last
	}

	return res
}

func helperCaseIDs(offset int, count int) []uuid.UUID {
	res := make([]uuid.UUID, 0, count)
	for i := 0; i < count; i++ {
		res = append(res, uuid.FromStringOrNil(fmt.Sprintf("00000000-0000-4000-8000-%012d", offset+i)))
	}

	return res
}

func Test_migrationTargetCustomerIDs_paging_errors(t *testing.T) {
	// a time in a non-UTC zone: the cursor must be rendered in UTC.
	lastCreate := time.Date(2026, 1, 2, 12, 34, 56, 123456000, time.FixedZone("case", 9*3600))
	wantToken := "2026-01-02T03:34:56.123456Z"
	secondCreate := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		onlyCustomerID uuid.UUID
		setup          func(mockDB *dbhandler.MockDBHandler, ctx context.Context)

		expectIDs []uuid.UUID
		expectErr string
	}{
		{
			name:           "a single customer needs no listing",
			onlyCustomerID: uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000aa"),
			setup:          func(mockDB *dbhandler.MockDBHandler, ctx context.Context) {},
			expectIDs:      []uuid.UUID{uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000aa")},
		},
		{
			name: "a page smaller than the page size ends the scan",
			setup: func(mockDB *dbhandler.MockDBHandler, ctx context.Context) {
				mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), "", gomock.Any()).Return(helperCaseDomains(1, migrationPageSize-1, &lastCreate), nil)
			},
			expectIDs: helperCaseIDs(1, migrationPageSize-1),
		},
		{
			name: "a full page is followed by a request from the last tm_create",
			setup: func(mockDB *dbhandler.MockDBHandler, ctx context.Context) {
				first := mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), "", gomock.Any()).Return(helperCaseDomains(1, migrationPageSize, &lastCreate), nil)
				mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), wantToken, gomock.Any()).After(first).Return(helperCaseDomains(migrationPageSize+1, 3, &secondCreate), nil)
			},
			expectIDs: helperCaseIDs(1, migrationPageSize+3),
		},
		{
			name: "a full page followed by an empty page ends the scan",
			setup: func(mockDB *dbhandler.MockDBHandler, ctx context.Context) {
				first := mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), "", gomock.Any()).Return(helperCaseDomains(1, migrationPageSize, &lastCreate), nil)
				mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), wantToken, gomock.Any()).After(first).Return([]*customerdomain.CustomerDomain{}, nil)
			},
			expectIDs: helperCaseIDs(1, migrationPageSize),
		},
		{
			name: "a listing failure is wrapped and no partial result is returned",
			setup: func(mockDB *dbhandler.MockDBHandler, ctx context.Context) {
				first := mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), "", gomock.Any()).Return(helperCaseDomains(1, migrationPageSize, &lastCreate), nil)
				mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), wantToken, gomock.Any()).After(first).Return(nil, errInjectedHelperCase)
			},
			expectErr: "could not list customer domains",
		},
		{
			name: "a full page whose last row has no tm_create cannot be paged",
			setup: func(mockDB *dbhandler.MockDBHandler, ctx context.Context) {
				mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), "", gomock.Any()).Return(helperCaseDomains(1, migrationPageSize, nil), nil)
			},
			expectErr: "nil tm_create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			ctx := context.Background()
			tt.setup(mockDB, ctx)

			res, err := migrationTargetCustomerIDs(ctx, mockDB, tt.onlyCustomerID)

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil result, got: %d ids", len(res))
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(res) != len(tt.expectIDs) {
				t.Fatalf("Wrong match. expect: %d ids, got: %d", len(tt.expectIDs), len(res))
			}
			for i := range res {
				if res[i] != tt.expectIDs[i] {
					t.Fatalf("Wrong match at %d. expect: %s, got: %s", i, tt.expectIDs[i], res[i])
				}
			}
		})
	}
}

func Test_migrationTargetCustomerIDs_listErrorWrapsCause_errors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	ctx := context.Background()

	mockDB.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), "", gomock.Any()).Return(nil, errInjectedHelperCase)

	_, err := migrationTargetCustomerIDs(ctx, mockDB, uuid.Nil)
	if !errors.Is(err, errInjectedHelperCase) {
		t.Errorf("Wrong match. expect: the cause to stay reachable, got: %v", err)
	}
}

func Test_resolveUUID_errors(t *testing.T) {
	tests := []struct {
		name  string
		value string

		expectRes uuid.UUID
		expectErr string
	}{
		{name: "empty value", value: "", expectRes: uuid.Nil, expectErr: "case label is required"},
		{name: "not a uuid", value: "not-a-uuid", expectRes: uuid.Nil, expectErr: "invalid format for case label: 'not-a-uuid' is not a valid UUID"},
		{name: "the nil uuid is not accepted", value: "00000000-0000-0000-0000-000000000000", expectRes: uuid.Nil, expectErr: "invalid format for case label"},
		{name: "valid uuid", value: "00000000-0000-4000-8000-0000000000bb", expectRes: uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000bb")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("case_flag", tt.value)

			res, err := resolveUUID("case_flag", "case label")

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_resolveString_errors(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	if res, err := resolveString("case_flag", "case label"); err == nil || !strings.Contains(err.Error(), "case label is required") || res != "" {
		t.Errorf("Wrong match. expect: required error, got: %q, %v", res, err)
	}

	viper.Set("case_flag", "case value")
	if res, err := resolveString("case_flag", "case label"); err != nil || res != "case value" {
		t.Errorf("Wrong match. expect: case value, got: %q, %v", res, err)
	}
}

func Test_printJSON_errors(t *testing.T) {
	t.Run("a value that cannot be encoded is reported", func(t *testing.T) {
		err := printJSON(make(chan int))
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong match. expect: marshal error, got: %v", err)
		}
	})

	t.Run("a value is printed as indented json", func(t *testing.T) {
		origStdout := os.Stdout
		r, w, errPipe := os.Pipe()
		if errPipe != nil {
			t.Fatalf("could not create a pipe: %v", errPipe)
		}
		os.Stdout = w
		t.Cleanup(func() { os.Stdout = origStdout })

		errPrint := printJSON(map[string]int{"count": 3})
		_ = w.Close()
		os.Stdout = origStdout

		out, errRead := io.ReadAll(r)
		if errRead != nil {
			t.Fatalf("could not read the output: %v", errRead)
		}
		if errPrint != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", errPrint)
		}
		if want := "{\n  \"count\": 3\n}\n"; string(out) != want {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", want, string(out))
		}
	})
}
