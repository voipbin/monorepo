package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/smotes/purse"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-route-manager/internal/config"
	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/cachehandler"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

// The tests in this file replace process wide state (the connect function, the broker factory, the config
// fields, the prometheus registerer, the standard output), so none of them runs in parallel.

var dbCounter atomic.Int64

const (
	testCustomerID = "b9d2e1a4-5c3f-4a8e-9f10-2d7c6e5b4a39"
	testProviderID = "6a1f3c52-8d47-4e0b-a2c9-71e5b8d09f34"
	testRouteID    = "c3e7a9f1-2b64-4d58-8e1a-5f0c9b7d3a26"
	testUnknownID  = "0f8e6d4c-1a2b-4c3d-9e5f-7a8b9c0d1e2f"
)

// cliEnv is the environment of one command: an sqlite database, a cache and a broker that does not exist.
type cliEnv struct {
	db    *sql.DB
	store dbhandler.DBHandler

	// events are the event types published through the broker
	events []string
}

// newCLIEnv installs the seams for one test. initHandler is expected to run once when expectInit is true.
func newCLIEnv(t *testing.T, expectInit bool) *cliEnv {
	t.Helper()

	// the request handler registers its collectors without a guard, so every test needs its own registry
	origReg := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = origReg })

	mr := miniredis.RunT(t)
	cfg := config.Get()
	origAddr := cfg.RedisAddress
	t.Cleanup(func() { cfg.RedisAddress = origAddr })
	cfg.RedisAddress = mr.Addr()

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:routecontrol%d?mode=memory&cache=shared", dbCounter.Add(1)))
	if err != nil {
		t.Fatalf("could not open the database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	ps, err := purse.New(filepath.Join("../../scripts/database_scripts"))
	if err != nil {
		t.Fatalf("could not load the schema. err: %v", err)
	}
	for _, f := range ps.Files() {
		contents, ok := ps.Get(f)
		if !ok {
			t.Fatalf("could not get the schema file. file: %s", f)
		}
		if _, errExec := db.Exec(contents); errExec != nil {
			t.Fatalf("could not apply the schema. file: %s, err: %v", f, errExec)
		}
	}

	cache := cachehandler.NewHandler(mr.Addr(), cfg.RedisPassword, cfg.RedisDatabase)
	res := &cliEnv{
		db:    db,
		store: dbhandler.NewHandler(db, cache),
	}

	t.Cleanup(databasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return db, nil }))

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	mockSock := sockhandler.NewMockSockHandler(mc)
	if expectInit {
		mockSock.EXPECT().Connect().Times(1)
		mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	}
	mockSock.EXPECT().EventPublish(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ string, _ string, evt *sock.Event) error {
			res.events = append(res.events, string(evt.Type))
			return nil
		}).AnyTimes()
	t.Cleanup(sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock }))

	return res
}

// seedRoute stores a route through the database handler.
func (e *cliEnv) seedRoute(t *testing.T, id, customerID, target string, priority int) {
	t.Helper()

	err := e.store.RouteCreate(context.Background(), &route.Route{
		ID:         uuid.FromStringOrNil(id),
		CustomerID: uuid.FromStringOrNil(customerID),
		Name:       "seed " + id[:8],
		Detail:     "seeded route",
		ProviderID: uuid.FromStringOrNil(testProviderID),
		Priority:   priority,
		Target:     target,
	})
	if err != nil {
		t.Fatalf("could not seed the route. err: %v", err)
	}
}

// dropRoutes removes the table so that the next query of the handler fails.
func (e *cliEnv) dropRoutes(t *testing.T) {
	t.Helper()

	if _, err := e.db.Exec("drop table route_routes"); err != nil {
		t.Fatalf("could not drop the table. err: %v", err)
	}
}

// runCLI runs the cli and returns what it printed and the execution error.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var err error
	out := captureStdout(t, func() { err = executeCommand(t, args...) })
	return out, err
}

func parseRoute(t *testing.T, out string) *route.Route {
	t.Helper()

	res := &route.Route{}
	if err := json.Unmarshal([]byte(out), res); err != nil {
		t.Fatalf("could not parse the output. output: %q, err: %v", out, err)
	}
	return res
}

func parseRoutes(t *testing.T, out string) []*route.Route {
	t.Helper()

	var res []*route.Route
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("could not parse the output. output: %q, err: %v", out, err)
	}
	return res
}

func expectEvents(t *testing.T, got []string, expect ...string) {
	t.Helper()

	if strings.Join(got, ",") != strings.Join(expect, ",") {
		t.Errorf("Wrong match. expect events: %v, got: %v", expect, got)
	}
}

func Test_initHandler_success(t *testing.T) {
	newCLIEnv(t, true)

	res, err := initHandler()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil {
		t.Errorf("Wrong match. expect: handler, got: nil")
	}
}

func Test_initHandler_cacheUnreachable(t *testing.T) {
	newCLIEnv(t, false)

	// a closed cache address refuses the connection
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	cfg := config.Get()
	orig := cfg.RedisAddress
	t.Cleanup(func() { cfg.RedisAddress = orig })
	cfg.RedisAddress = addr

	res, err := initHandler()
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if err == nil || !strings.Contains(err.Error(), "could not initialize the cache") {
		t.Errorf("Wrong match. expect: could not initialize the cache, got: %v", err)
	}
}

func Test_routeCreate(t *testing.T) {
	t.Run("creates the route and publishes the event", func(t *testing.T) {
		env := newCLIEnv(t, true)

		out, err := runCLI(t, "route", "create",
			"--customer-id", testCustomerID, "--name", "route a", "--detail", "detail a",
			"--provider-id", testProviderID, "--priority", "3", "--target", "82")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := parseRoute(t, out)
		if res.ID == uuid.Nil || res.TMCreate == nil {
			t.Errorf("Wrong match. expect: id and tm_create set, got: %v", res)
		}
		if res.CustomerID.String() != testCustomerID || res.Name != "route a" || res.Detail != "detail a" ||
			res.ProviderID.String() != testProviderID || res.Priority != 3 || res.Target != "82" {
			t.Errorf("Wrong match. got: %v", res)
		}

		stored, errGet := env.store.RouteGet(context.Background(), res.ID)
		if errGet != nil || stored.Name != "route a" || stored.Target != "82" {
			t.Errorf("Wrong match. expect: the route stored, got: %v, err: %v", stored, errGet)
		}
		expectEvents(t, env.events, route.EventTypeRouteCreated)
	})

	tests := []struct {
		name string
		args []string

		expectError string
	}{
		{
			name:        "customer id is missing",
			args:        []string{"--name", "n", "--provider-id", testProviderID, "--target", "1"},
			expectError: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:        "customer id is invalid",
			args:        []string{"--customer-id", "x", "--name", "n", "--provider-id", testProviderID, "--target", "1"},
			expectError: "failed to resolve customer ID: invalid format for Customer ID",
		},
		{
			name:        "name is missing",
			args:        []string{"--customer-id", testCustomerID, "--provider-id", testProviderID, "--target", "1"},
			expectError: "name is required",
		},
		{
			name:        "provider id is missing",
			args:        []string{"--customer-id", testCustomerID, "--name", "n", "--target", "1"},
			expectError: "failed to resolve provider ID: Provider ID is required",
		},
		{
			name:        "target is missing",
			args:        []string{"--customer-id", testCustomerID, "--name", "n", "--provider-id", testProviderID},
			expectError: "target is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newCLIEnv(t, true)

			out, err := runCLI(t, append([]string{"route", "create"}, tt.args...)...)
			if err == nil || !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectError, err)
			}
			if out != "" {
				t.Errorf("Wrong match. expect: no output, got: %q", out)
			}
			expectEvents(t, env.events)
		})
	}

	t.Run("a store failure is reported and nothing is published", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.dropRoutes(t)

		_, err := runCLI(t, "route", "create",
			"--customer-id", testCustomerID, "--name", "n", "--provider-id", testProviderID, "--target", "1")
		if err == nil || !strings.Contains(err.Error(), "failed to create route") {
			t.Errorf("Wrong match. expect: failed to create route, got: %v", err)
		}
		expectEvents(t, env.events)
	})
}

func Test_routeGet(t *testing.T) {
	t.Run("prints the stored route", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.seedRoute(t, testRouteID, testCustomerID, "82", 2)

		out, err := runCLI(t, "route", "get", "--id", testRouteID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := parseRoute(t, out)
		if res.ID.String() != testRouteID || res.CustomerID.String() != testCustomerID || res.Target != "82" || res.Priority != 2 {
			t.Errorf("Wrong match. got: %v", res)
		}
	})

	tests := []struct {
		name string
		args []string

		expectError string
	}{
		{"id is missing", []string{"route", "get"}, "failed to resolve route ID: Route ID is required"},
		{"id is invalid", []string{"route", "get", "--id", "x"}, "failed to resolve route ID: invalid format for Route ID"},
		{"unknown route", []string{"route", "get", "--id", testUnknownID}, "failed to retrieve route"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newCLIEnv(t, true)

			out, err := runCLI(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectError, err)
			}
			if out != "" {
				t.Errorf("Wrong match. expect: no output, got: %q", out)
			}
		})
	}
}

func Test_routeList(t *testing.T) {
	t.Run("prints the routes of the customer only", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.seedRoute(t, testRouteID, testCustomerID, "82", 1)
		env.seedRoute(t, "d4f8b0a2-3c75-4e69-9f2b-6a1d0c8e4b37", testCustomerID, "all", 2)
		env.seedRoute(t, "e5a9c1b3-4d86-4f7a-a03c-7b2e1d9f5c48", "f6bad2c4-5e97-4a8b-b14d-8c3f2ea06d59", "82", 1)

		out, err := runCLI(t, "route", "list", "--customer-id", testCustomerID, "--limit", "10")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := parseRoutes(t, out)
		if len(res) != 2 {
			t.Fatalf("Wrong match. expect: 2 routes, got: %d", len(res))
		}
		for _, r := range res {
			if r.CustomerID.String() != testCustomerID {
				t.Errorf("Wrong match. expect: customer %s only, got: %v", testCustomerID, r)
			}
		}
	})

	t.Run("the limit applies", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.seedRoute(t, testRouteID, testCustomerID, "82", 1)
		env.seedRoute(t, "d4f8b0a2-3c75-4e69-9f2b-6a1d0c8e4b37", testCustomerID, "all", 2)

		out, err := runCLI(t, "route", "list", "--customer-id", testCustomerID, "--limit", "1")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res := parseRoutes(t, out); len(res) != 1 {
			t.Errorf("Wrong match. expect: 1 route, got: %d", len(res))
		}
	})

	t.Run("customer id is missing", func(t *testing.T) {
		newCLIEnv(t, true)

		_, err := runCLI(t, "route", "list")
		if err == nil || !strings.Contains(err.Error(), "failed to resolve customer ID: Customer ID is required") {
			t.Errorf("Wrong match. got: %v", err)
		}
	})

	t.Run("a store failure is reported", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.dropRoutes(t)

		_, err := runCLI(t, "route", "list", "--customer-id", testCustomerID)
		if err == nil || !strings.Contains(err.Error(), "failed to retrieve routes") {
			t.Errorf("Wrong match. got: %v", err)
		}
	})
}

func Test_routeListByTarget(t *testing.T) {
	t.Run("returns the target routes and the all routes of other providers", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.seedRoute(t, testRouteID, testCustomerID, "82", 1)
		env.seedRoute(t, "d4f8b0a2-3c75-4e69-9f2b-6a1d0c8e4b37", testCustomerID, "1", 2)

		out, err := runCLI(t, "route", "list-by-target", "--customer-id", testCustomerID, "--target", "82")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := parseRoutes(t, out)
		if len(res) != 1 || res[0].ID.String() != testRouteID {
			t.Errorf("Wrong match. expect: only the route of target 82, got: %v", res)
		}
	})

	tests := []struct {
		name string
		args []string
		drop bool

		expectError string
	}{
		{"customer id is missing", []string{"route", "list-by-target", "--target", "1"}, false, "failed to resolve customer ID: Customer ID is required"},
		{"target is missing", []string{"route", "list-by-target", "--customer-id", testCustomerID}, false, "target is required"},
		{"a store failure is reported", []string{"route", "list-by-target", "--customer-id", testCustomerID, "--target", "1"}, true, "failed to retrieve routes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newCLIEnv(t, true)
			if tt.drop {
				env.dropRoutes(t)
			}

			_, err := runCLI(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectError, err)
			}
		})
	}
}

func Test_routeUpdate(t *testing.T) {
	t.Run("updates every field and publishes the event", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.seedRoute(t, testRouteID, testCustomerID, "82", 1)

		newProvider := "7b2a4d63-9e58-4f1c-b3da-82f6c9e1a045"
		out, err := runCLI(t, "route", "update", "--id", testRouteID, "--name", "renamed", "--detail", "new detail",
			"--provider-id", newProvider, "--priority", "7", "--target", "1")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := parseRoute(t, out)
		if res.Name != "renamed" || res.Detail != "new detail" || res.ProviderID.String() != newProvider || res.Priority != 7 || res.Target != "1" {
			t.Errorf("Wrong match. got: %v", res)
		}
		if res.CustomerID.String() != testCustomerID {
			t.Errorf("Wrong match. expect: the customer is untouched, got: %s", res.CustomerID)
		}

		stored, errGet := env.store.RouteGet(context.Background(), uuid.FromStringOrNil(testRouteID))
		if errGet != nil || stored.Name != "renamed" || stored.Priority != 7 {
			t.Errorf("Wrong match. expect: the update stored, got: %v, err: %v", stored, errGet)
		}
		expectEvents(t, env.events, route.EventTypeRouteUpdated)
	})

	tests := []struct {
		name string
		args []string

		expectError string
	}{
		{"id is missing", []string{"--name", "n", "--provider-id", testProviderID, "--target", "1"}, "failed to resolve route ID: Route ID is required"},
		{"name is missing", []string{"--id", testRouteID, "--provider-id", testProviderID, "--target", "1"}, "name is required"},
		{"provider id is missing", []string{"--id", testRouteID, "--name", "n", "--target", "1"}, "failed to resolve provider ID: Provider ID is required"},
		{"target is missing", []string{"--id", testRouteID, "--name", "n", "--provider-id", testProviderID}, "target is required"},
		{"unknown route", []string{"--id", testUnknownID, "--name", "n", "--provider-id", testProviderID, "--target", "1"}, "failed to update route"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newCLIEnv(t, true)

			out, err := runCLI(t, append([]string{"route", "update"}, tt.args...)...)
			if err == nil || !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectError, err)
			}
			if out != "" {
				t.Errorf("Wrong match. expect: no output, got: %q", out)
			}
			expectEvents(t, env.events)
		})
	}
}

func Test_routeDelete(t *testing.T) {
	t.Run("deletes the route and publishes the event", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.seedRoute(t, testRouteID, testCustomerID, "82", 1)

		out, err := runCLI(t, "route", "delete", "--id", testRouteID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := parseRoute(t, out)
		if res.ID.String() != testRouteID || res.TMDelete == nil {
			t.Errorf("Wrong match. expect: the deleted route with tm_delete, got: %v", res)
		}
		expectEvents(t, env.events, route.EventTypeRouteDeleted)

		// a deleted route is not listed any more
		listed, errList := env.store.RouteList(context.Background(), "", 10, map[route.Field]any{route.FieldCustomerID: uuid.FromStringOrNil(testCustomerID)})
		if errList != nil || len(listed) != 0 {
			t.Errorf("Wrong match. expect: no listed route, got: %v, err: %v", listed, errList)
		}
	})

	tests := []struct {
		name string
		args []string

		expectError string
	}{
		{"id is missing", []string{"route", "delete"}, "failed to resolve route ID: Route ID is required"},
		{"id is invalid", []string{"route", "delete", "--id", "x"}, "failed to resolve route ID: invalid format for Route ID"},
		{"unknown route", []string{"route", "delete", "--id", testUnknownID}, "failed to delete route"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newCLIEnv(t, true)

			_, err := runCLI(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectError, err)
			}
			expectEvents(t, env.events)
		})
	}
}

func Test_dialrouteList(t *testing.T) {
	t.Run("merges the routes of the customer and the default routes", func(t *testing.T) {
		env := newCLIEnv(t, true)
		env.seedRoute(t, testRouteID, testCustomerID, "82", 1)
		env.seedRoute(t, "d4f8b0a2-3c75-4e69-9f2b-6a1d0c8e4b37", route.CustomerIDBasicRoute.String(), "82", 1)

		out, err := runCLI(t, "route", "dialroute-list", "--customer-id", testCustomerID, "--target", "82")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		// both routes use the same provider, so the customer route wins and the default route is dropped
		res := parseRoutes(t, out)
		if len(res) != 1 || res[0].ID.String() != testRouteID {
			t.Errorf("Wrong match. expect: the customer route only, got: %v", res)
		}
	})

	tests := []struct {
		name string
		args []string
		drop bool

		expectError string
	}{
		{"customer id is missing", []string{"route", "dialroute-list", "--target", "1"}, false, "failed to resolve customer ID: Customer ID is required"},
		{"target is missing", []string{"route", "dialroute-list", "--customer-id", testCustomerID}, false, "target is required"},
		{"a store failure is reported", []string{"route", "dialroute-list", "--customer-id", testCustomerID, "--target", "1"}, true, "failed to retrieve dialroutes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newCLIEnv(t, true)
			if tt.drop {
				env.dropRoutes(t)
			}

			_, err := runCLI(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectError, err)
			}
		})
	}
}
