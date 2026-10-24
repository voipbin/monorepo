package main

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_countNonDeletedAgents(t *testing.T) {
	query := regexp.QuoteMeta("SELECT COUNT(*) FROM agent_agents WHERE tm_delete IS NULL")

	t.Run("returns the counted rows", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("could not create sqlmock: %v", err)
		}
		defer func() { _ = db.Close() }()

		mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

		n, err := countNonDeletedAgents(context.Background(), db)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if n != 7 {
			t.Errorf("Wrong match. expect: 7, got: %d", n)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unmet expectations: %v", errMet)
		}
	})

	t.Run("scan failure is reported", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("could not create sqlmock: %v", err)
		}
		defer func() { _ = db.Close() }()

		mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow("not-a-number"))

		n, err := countNonDeletedAgents(context.Background(), db)
		if err == nil || !strings.HasPrefix(err.Error(), "could not scan count") {
			t.Errorf("Wrong match. expect a scan error, got: %v", err)
		}
		if n != 0 {
			t.Errorf("Wrong match. expect 0 on error, got: %d", n)
		}
	})

	t.Run("query failure is reported", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("could not create sqlmock: %v", err)
		}
		defer func() { _ = db.Close() }()

		mock.ExpectQuery(query).WillReturnError(errors.New("simulated query failure"))

		_, err = countNonDeletedAgents(context.Background(), db)
		if err == nil || !strings.Contains(err.Error(), "simulated query failure") {
			t.Errorf("Wrong match. expect the query error, got: %v", err)
		}
	})
}

func Test_scanAgents(t *testing.T) {
	customerA := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	customerB := uuid.FromStringOrNil("9129ad1a-2fd5-11ef-af80-1f74bf8dbf2b")
	idA := uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6")
	idB := uuid.FromStringOrNil("8e3b890a-2fd7-11ef-b442-133f59be8b36")
	idC := uuid.FromStringOrNil("0a5fa8c2-8f45-11ef-93f0-0fd5de1b7c11")

	tel := func(target string) commonaddress.Address {
		return commonaddress.Address{Type: commonaddress.TypeTel, Target: target}
	}
	newAgent := func(id uuid.UUID, customerID uuid.UUID, tmCreate *time.Time, addrs ...commonaddress.Address) *agent.Agent {
		return &agent.Agent{
			Identity:  commonidentity.Identity{ID: id, CustomerID: customerID},
			Addresses: addrs,
			TMCreate:  tmCreate,
		}
	}
	at := func(tm time.Time) *time.Time { return &tm }

	// the premise of the collision cases: both raw targets share one canonical form.
	if mustNorm(commonaddress.TypeTel, "+1-555-0100") != mustNorm(commonaddress.TypeTel, "+1 (555) 0100") {
		t.Fatalf("test premise broken: the two raw targets must share one canonical form")
	}

	wantFilters := map[agent.Field]any{agent.FieldDeleted: false}
	plus9 := time.FixedZone("plus9", 9*3600)

	type listCall struct {
		expectToken string
		res         []*agent.Agent
		err         error
	}

	tests := []struct {
		name string

		pageSize uint64
		calls    []listCall

		expectErr        string
		expectVisited    int
		expectChangedIDs []uuid.UUID
		expectCollisions map[string][]uuid.UUID
		expectStdout     string
	}{
		{
			name:     "single short page ends after one call",
			pageSize: 5,
			calls: []listCall{
				{expectToken: "", res: []*agent.Agent{
					newAgent(idA, customerA, at(time.Now()), tel("+1-555-0100")),
					newAgent(idB, customerA, at(time.Now()), tel(mustNorm(commonaddress.TypeTel, "+1-555-0200"))),
				}},
			},
			expectVisited:    2,
			expectChangedIDs: []uuid.UUID{idA},
			expectCollisions: map[string][]uuid.UUID{},
		},
		{
			name:     "full page advances the cursor to the oldest tm_create in UTC",
			pageSize: 2,
			calls: []listCall{
				{expectToken: "", res: []*agent.Agent{
					newAgent(idA, customerA, at(time.Date(2026, 1, 2, 13, 0, 0, 0, plus9))),
					newAgent(idB, customerA, at(time.Date(2026, 1, 2, 12, 0, 0, 123456000, plus9))),
				}},
				{expectToken: "2026-01-02T03:00:00.123456Z", res: []*agent.Agent{
					newAgent(idC, customerA, at(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))),
				}},
			},
			expectVisited:    3,
			expectChangedIDs: []uuid.UUID{},
			expectCollisions: map[string][]uuid.UUID{},
		},
		{
			name:     "full page followed by an empty page stops on the empty page",
			pageSize: 2,
			calls: []listCall{
				{expectToken: "", res: []*agent.Agent{
					newAgent(idA, customerA, at(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))),
					newAgent(idB, customerA, at(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))),
				}},
				{expectToken: "2026-01-01T00:00:00.000000Z", res: []*agent.Agent{}},
			},
			expectVisited:    2,
			expectChangedIDs: []uuid.UUID{},
			expectCollisions: map[string][]uuid.UUID{},
		},
		{
			name:     "nil tm_create on a full page stops the scan with a warning",
			pageSize: 1,
			calls: []listCall{
				{expectToken: "", res: []*agent.Agent{newAgent(idA, customerA, nil)}},
			},
			expectVisited:    1,
			expectChangedIDs: []uuid.UUID{},
			expectCollisions: map[string][]uuid.UUID{},
			expectStdout:     "warning: agent " + idA.String() + " has a nil tm_create; stopping scan early (completeness check will flag it)\n",
		},
		{
			name:     "list failure is wrapped",
			pageSize: 5,
			calls: []listCall{
				{expectToken: "", err: errors.New("simulated list failure")},
			},
			expectErr: "could not list agents: simulated list failure",
		},
		{
			name:     "two agents that share a canonical target collide",
			pageSize: 5,
			calls: []listCall{
				{expectToken: "", res: []*agent.Agent{
					newAgent(idA, customerA, at(time.Now()), tel("+1-555-0100")),
					newAgent(idB, customerA, at(time.Now()), tel("+1 (555) 0100")),
				}},
			},
			expectVisited:    2,
			expectChangedIDs: []uuid.UUID{idA, idB},
			expectCollisions: map[string][]uuid.UUID{
				collisionKey(customerA, commonaddress.Address{Type: commonaddress.TypeTel, Target: mustNorm(commonaddress.TypeTel, "+1-555-0100")}): {idA, idB},
			},
		},
		{
			name:     "one agent owning the same canonical target twice is not a collision",
			pageSize: 5,
			calls: []listCall{
				{expectToken: "", res: []*agent.Agent{
					newAgent(idA, customerA, at(time.Now()), tel("+1-555-0100"), tel("+1 (555) 0100")),
				}},
			},
			expectVisited:    1,
			expectChangedIDs: []uuid.UUID{idA},
			expectCollisions: map[string][]uuid.UUID{},
		},
		{
			name:     "the same canonical target on different customers is not a collision",
			pageSize: 5,
			calls: []listCall{
				{expectToken: "", res: []*agent.Agent{
					newAgent(idA, customerA, at(time.Now()), tel("+1-555-0100")),
					newAgent(idB, customerB, at(time.Now()), tel("+1-555-0100")),
				}},
			},
			expectVisited:    2,
			expectChangedIDs: []uuid.UUID{idA, idB},
			expectCollisions: map[string][]uuid.UUID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			var prev *gomock.Call
			for _, c := range tt.calls {
				call := mockDB.EXPECT().AgentList(gomock.Any(), tt.pageSize, c.expectToken, wantFilters).Return(c.res, c.err).Times(1)
				if prev != nil {
					call.After(prev)
				}
				prev = call
			}

			var changes []addressChange
			var collisions map[string][]uuid.UUID
			var visited int
			var err error
			stdout := captureStdout(t, func() {
				changes, collisions, visited, err = scanAgents(context.Background(), mockDB, tt.pageSize)
			})

			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Fatalf("Wrong match. expect error: %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect ok, got: %v", err)
			}

			if visited != tt.expectVisited {
				t.Errorf("Wrong match. expect visited: %d, got: %d", tt.expectVisited, visited)
			}

			changedIDs := []uuid.UUID{}
			for _, c := range changes {
				changedIDs = append(changedIDs, c.agentID)
			}
			if !reflect.DeepEqual(changedIDs, tt.expectChangedIDs) {
				t.Errorf("Wrong match. expect changed: %v, got: %v", tt.expectChangedIDs, changedIDs)
			}

			if len(collisions) != len(tt.expectCollisions) {
				t.Fatalf("Wrong match. expect collisions: %v, got: %v", tt.expectCollisions, collisions)
			}
			for k, want := range tt.expectCollisions {
				got := collisions[k]
				if !reflect.DeepEqual(got, want) {
					t.Errorf("Wrong match. collision %q expect owners: %v, got: %v", k, want, got)
				}
			}

			if stdout != tt.expectStdout {
				t.Errorf("Wrong match. expect stdout: %q, got: %q", tt.expectStdout, stdout)
			}
		})
	}
}

func Test_addressesDiffer_lengthMismatch(t *testing.T) {
	a := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}

	if !addressesDiffer([]commonaddress.Address{a}, []commonaddress.Address{a, a}) {
		t.Errorf("Wrong match. expect a longer list to differ")
	}
	if !addressesDiffer([]commonaddress.Address{a, a}, []commonaddress.Address{a}) {
		t.Errorf("Wrong match. expect a shorter list to differ")
	}
	if addressesDiffer(nil, []commonaddress.Address{}) {
		t.Errorf("Wrong match. expect two empty lists not to differ")
	}
}

func Test_dedupeUUIDs(t *testing.T) {
	idA := uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6")
	idB := uuid.FromStringOrNil("8e3b890a-2fd7-11ef-b442-133f59be8b36")

	tests := []struct {
		name   string
		ids    []uuid.UUID
		expect []uuid.UUID
	}{
		{"empty", nil, []uuid.UUID{}},
		{"duplicates are dropped and the first-seen order is kept", []uuid.UUID{idB, idA, idB, idA}, []uuid.UUID{idB, idA}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupeUUIDs(tt.ids)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_reportChanges(t *testing.T) {
	idA := uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6")
	canonical := mustNorm(commonaddress.TypeTel, "+1-555-0100")

	t.Run("no changes prints nothing", func(t *testing.T) {
		out := captureStdout(t, func() { reportChanges(nil) })
		if out != "" {
			t.Errorf("Wrong match. expect no output, got: %q", out)
		}
	})

	t.Run("unchanged addresses of a changed agent are skipped", func(t *testing.T) {
		unchanged := commonaddress.Address{Type: commonaddress.TypeExtension, Target: "49b41028-2d8d-11ef-b38d-27dd55f2bb71"}
		changes := []addressChange{{
			agentID:  idA,
			original: []commonaddress.Address{unchanged, {Type: commonaddress.TypeTel, Target: "+1-555-0100"}},
			normalized: []commonaddress.Address{
				unchanged, {Type: commonaddress.TypeTel, Target: canonical},
			},
		}}

		out := captureStdout(t, func() { reportChanges(changes) })

		expect := "\nplanned changes (agent_id: type raw -> canonical):\n" +
			"  " + idA.String() + ": tel \"+1-555-0100\" -> \"" + canonical + "\"\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})
}

func Test_reportCollisions(t *testing.T) {
	customerA := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	idA := uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6")
	idB := uuid.FromStringOrNil("8e3b890a-2fd7-11ef-b442-133f59be8b36")
	idC := uuid.FromStringOrNil("0a5fa8c2-8f45-11ef-93f0-0fd5de1b7c11")

	t.Run("none", func(t *testing.T) {
		out := captureStdout(t, func() { reportCollisions(map[string][]uuid.UUID{}) })
		if out != "\ncollisions: none\n" {
			t.Errorf("Wrong match. got: %q", out)
		}
	})

	t.Run("collisions are listed in sorted key order", func(t *testing.T) {
		collisions := map[string][]uuid.UUID{
			collisionKey(customerA, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550200"}): {idC, idA},
			collisionKey(customerA, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}): {idA, idB},
		}

		out := captureStdout(t, func() { reportCollisions(collisions) })

		expect := "\ncollisions: 2 (a canonical (customer,type,target) owned by >1 agent)\n" +
			"  customer=" + customerA.String() + " type=tel target=\"+15550100\" owners=[" + idA.String() + ", " + idB.String() + "]\n" +
			"  customer=" + customerA.String() + " type=tel target=\"+15550200\" owners=[" + idC.String() + ", " + idA.String() + "]\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})
}

func Test_formatAddresses(t *testing.T) {
	tests := []struct {
		name      string
		addresses []commonaddress.Address
		expect    string
	}{
		{"empty", nil, "[]"},
		{
			"two addresses",
			[]commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+15550100"},
				{Type: commonaddress.TypeSIP, Target: "alice@example.com"},
			},
			"[{tel:+15550100},{sip:alice@example.com}]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAddresses(tt.addresses); got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_dryRunLabel(t *testing.T) {
	if got := dryRunLabel(true); got != "dry-run (preview only)" {
		t.Errorf("Wrong match. got: %q", got)
	}
	if got := dryRunLabel(false); got != "apply" {
		t.Errorf("Wrong match. got: %q", got)
	}
}
