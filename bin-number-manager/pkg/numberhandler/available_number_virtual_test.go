package numberhandler

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/pkg/dbhandler"
)

func Test_GetAvailableVirtualNumbers(t *testing.T) {

	tests := []struct {
		name string

		limit uint
		// takenCount is how many of the received candidates the db reports as taken (from the front)
		takenCount int

		expectCandidates int
		expectCount      int
	}{
		{
			name: "limit 0 falls back to 10",

			limit:      0,
			takenCount: 0,

			expectCandidates: 30,
			expectCount:      10,
		},
		{
			name: "limit is applied",

			limit:      4,
			takenCount: 0,

			expectCandidates: 12,
			expectCount:      4,
		},
		{
			name: "taken candidates are skipped but the limit is still reached",

			limit:      10,
			takenCount: 20,

			expectCandidates: 30,
			expectCount:      10,
		},
		{
			name: "not enough free candidates",

			limit:      10,
			takenCount: 25,

			expectCandidates: 30,
			expectCount:      5,
		},
		{
			name: "all candidates taken",

			limit:      2,
			takenCount: 6,

			expectCandidates: 6,
			expectCount:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &numberHandler{db: mockDB}
			ctx := context.Background()

			var candidates []string
			mockDB.EXPECT().NumberGetExistingNumbers(ctx, gomock.Any()).DoAndReturn(
				func(_ context.Context, nums []string) ([]string, error) {
					candidates = append([]string{}, nums...)
					return append([]string{}, nums[:tt.takenCount]...), nil
				},
			)

			res, err := h.GetAvailableVirtualNumbers(ctx, tt.limit)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if len(candidates) != tt.expectCandidates {
				t.Fatalf("Wrong candidate count. expect: %d, got: %d", tt.expectCandidates, len(candidates))
			}

			seen := map[string]bool{}
			for _, c := range candidates {
				if !strings.HasPrefix(c, "+899") || len(c) != 13 {
					t.Errorf("Wrong candidate format: %s", c)
				}
				if strings.HasPrefix(c, "+899000") {
					t.Errorf("Candidate is in the reserved range: %s", c)
				}
				if seen[c] {
					t.Errorf("Duplicated candidate: %s", c)
				}
				seen[c] = true
			}

			if len(res) != tt.expectCount {
				t.Fatalf("Wrong result count. expect: %d, got: %d", tt.expectCount, len(res))
			}

			// the results are the first free candidates, in candidate order
			for i, r := range res {
				expect := &availablenumber.AvailableNumber{
					Number:       candidates[tt.takenCount+i],
					ProviderName: "",
					Country:      "899",
					Features:     []availablenumber.Feature{"voice"},
				}
				if !reflect.DeepEqual(expect, r) {
					t.Errorf("Wrong match at %d.\nexpect: %v\ngot: %v", i, expect, r)
				}
			}
		})
	}
}

func Test_GetAvailableVirtualNumbers_db_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &numberHandler{db: mockDB}
	ctx := context.Background()

	dbErr := fmt.Errorf("db failure")
	mockDB.EXPECT().NumberGetExistingNumbers(ctx, gomock.Any()).Return(nil, dbErr)

	res, err := h.GetAvailableVirtualNumbers(ctx, 3)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if !strings.Contains(err.Error(), "db failure") {
		t.Errorf("Wrong match. expect the db error to be wrapped, got: %v", err)
	}
}
