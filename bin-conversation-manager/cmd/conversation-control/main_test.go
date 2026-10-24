package main

import (
	"context"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-conversation-manager/internal/config"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

// resetState isolates a test from the process wide viper and config state. Tests in this
// package must not run in parallel.
func resetState(t *testing.T) {
	t.Helper()

	prev := *config.Get()
	viper.Reset()

	t.Cleanup(func() {
		viper.Reset()
		*config.Get() = prev
	})
}

// captureStdout returns what fn prints to the standard output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create a pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	// the pipe is drained while fn runs, so a large output cannot block the writer
	done := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()

	fn()

	_ = w.Close()
	os.Stdout = orig

	return <-done
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectRes uuid.UUID
		expectErr string
	}{
		{
			name:      "valid",
			value:     "a6b3cf48-2a4b-11ee-b574-2bad4f039ce5",
			expectRes: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
		},
		{
			name:      "empty",
			value:     "",
			expectRes: uuid.Nil,
			expectErr: "Test ID is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectRes: uuid.Nil,
			expectErr: "invalid format for Test ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "the all zero uuid is rejected as an invalid format",
			value:     "00000000-0000-0000-0000-000000000000",
			expectRes: uuid.Nil,
			expectErr: "invalid format for Test ID: '00000000-0000-0000-0000-000000000000' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetState(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Test ID")
			if res != tt.expectRes {
				t.Errorf("resolveUUID() = %v, want %v", res, tt.expectRes)
			}
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("resolveUUID() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("resolveUUID() error = %v, want %q", err, tt.expectErr)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("prints the indented JSON", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]string{"name": "test"})
		})
		if err != nil {
			t.Fatalf("printJSON() error = %v", err)
		}
		if want := "{\n  \"name\": \"test\"\n}\n"; out != want {
			t.Errorf("printJSON() wrote %q, want %q", out, want)
		}
	})

	t.Run("a value that cannot be marshaled is an error and prints nothing", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("printJSON() error = %v, want the marshal context", err)
		}
		if out != "" {
			t.Errorf("printJSON() wrote %q, want nothing", out)
		}
	})
}

func Test_initCommand(t *testing.T) {
	resetState(t)

	root := initCommand()

	if root.Use != "conversation-control" {
		t.Errorf("root Use = %q, want conversation-control", root.Use)
	}

	// the connection settings are persistent flags of the root
	for _, name := range []string{"database_dsn", "redis_address", "rabbitmq_address", "redis_database"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("root persistent flag %q is not registered", name)
		}
	}

	subs := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		subs[c.Name()] = c
	}
	for _, name := range []string{"conversation", "account", "message", "migrate"} {
		if subs[name] == nil {
			t.Fatalf("root command %q is not registered, got %v", name, subs)
		}
	}

	type op struct {
		parent string
		name   string
		runE   func(*cobra.Command, []string) error
		flags  map[string]string // flag name -> default
	}
	ops := []op{
		{"conversation", "get", runConversationGet, map[string]string{"id": ""}},
		{"conversation", "list", runConversationList, map[string]string{"limit": "100", "token": "", "customer-id": "", "type": ""}},
		{"account", "create", runAccountCreate, map[string]string{"customer-id": "", "type": "", "name": "", "detail": "", "secret": "", "token": "", "message-flow-id": ""}},
		{"account", "get", runAccountGet, map[string]string{"id": ""}},
		{"account", "list", runAccountList, map[string]string{"limit": "100", "token": "", "customer-id": "", "type": ""}},
		{"account", "update", runAccountUpdate, map[string]string{"id": "", "name": "", "detail": "", "secret": "", "token": ""}},
		{"account", "delete", runAccountDelete, map[string]string{"id": ""}},
		{"message", "get", runMessageGet, map[string]string{"id": ""}},
		{"message", "list", runMessageList, map[string]string{"limit": "100", "token": "", "customer-id": "", "conversation-id": "", "direction": "", "status": ""}},
		{"message", "delete", runMessageDelete, map[string]string{"id": ""}},
	}

	counts := map[string]int{}
	for _, o := range ops {
		counts[o.parent]++
	}
	for parent, want := range counts {
		if got := len(subs[parent].Commands()); got != want {
			t.Errorf("%s subcommands = %d, want %d", parent, got, want)
		}
	}

	for _, o := range ops {
		var c *cobra.Command
		for _, sub := range subs[o.parent].Commands() {
			if sub.Name() == o.name {
				c = sub
			}
		}
		if c == nil {
			t.Errorf("%s %s is not registered", o.parent, o.name)
			continue
		}

		// the command must run its own handler function
		if c.RunE == nil || reflect.ValueOf(c.RunE).Pointer() != reflect.ValueOf(o.runE).Pointer() {
			t.Errorf("%s %s RunE is not wired to its run function", o.parent, o.name)
		}

		n := 0
		c.Flags().VisitAll(func(*pflag.Flag) { n++ })
		if n != len(o.flags) {
			t.Errorf("%s %s has %d flags, want %d", o.parent, o.name, n, len(o.flags))
		}
		for name, def := range o.flags {
			f := c.Flags().Lookup(name)
			if f == nil {
				t.Errorf("%s %s flag %q is not registered", o.parent, o.name, name)
				continue
			}
			if f.DefValue != def {
				t.Errorf("%s %s flag %q default = %q, want %q", o.parent, o.name, name, f.DefValue, def)
			}
		}
	}
}

func Test_initCache(t *testing.T) {
	t.Run("connects to a running cache", func(t *testing.T) {
		resetState(t)
		mr := miniredis.RunT(t)
		*config.Get() = config.Config{RedisAddress: mr.Addr()}

		res, err := initCache()
		if err != nil || res == nil {
			t.Errorf("initCache() = %v, %v, want a connected handler", res, err)
		}
	})

	t.Run("the configured password and database index are used", func(t *testing.T) {
		resetState(t)
		mr := miniredis.RunT(t)
		mr.RequireAuth(strings.Repeat("r", 12))
		*config.Get() = config.Config{RedisAddress: mr.Addr(), RedisPassword: strings.Repeat("r", 12), RedisDatabase: 3}

		res, err := initCache()
		if err != nil || res == nil {
			t.Fatalf("initCache() = %v, %v, want a connected handler", res, err)
		}

		id := uuid.FromStringOrNil("c1c2c3c4-0000-4000-8000-000000000001")
		if errSet := res.MediaSet(context.Background(), &media.Media{Identity: commonidentity.Identity{ID: id}}); errSet != nil {
			t.Fatalf("MediaSet() error = %v", errSet)
		}

		key := "conversation.media:" + id.String()
		mr.Select(3)
		if !mr.Exists(key) {
			t.Errorf("key %q is not stored in database 3", key)
		}
		mr.Select(0)
		if mr.Exists(key) {
			t.Errorf("key %q must not be stored in database 0", key)
		}
	})

	t.Run("wrong password is reported with the context", func(t *testing.T) {
		resetState(t)
		mr := miniredis.RunT(t)
		mr.RequireAuth(strings.Repeat("r", 12))
		*config.Get() = config.Config{RedisAddress: mr.Addr(), RedisPassword: strings.Repeat("w", 12)}

		res, err := initCache()
		if res != nil {
			t.Errorf("initCache() = %v, want nil", res)
		}
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("initCache() error = %v, want the connect context", err)
		}
	})

	t.Run("stopped cache is reported with the context", func(t *testing.T) {
		resetState(t)
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()
		*config.Get() = config.Config{RedisAddress: addr}

		res, err := initCache()
		if res != nil || err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("initCache() = %v, %v, want nil and the connect context", res, err)
		}
	})
}

func Test_initConversationHandlers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	conversationHandler, accountHandler, messageHandler, err := initConversationHandlers(nil, mockCache, mockReq, mockNotify)
	if err != nil {
		t.Fatalf("initConversationHandlers() error = %v", err)
	}
	if conversationHandler == nil || accountHandler == nil || messageHandler == nil {
		t.Fatalf("initConversationHandlers() = %v, %v, %v, want three handlers", conversationHandler, accountHandler, messageHandler)
	}

	// the account handler reads through the injected cache, so a cached account needs no database
	id := uuid.FromStringOrNil("b1c2d3e4-0000-4000-8000-000000000001")
	cached := &account.Account{
		Identity: commonidentity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("b1c2d3e4-0000-4000-8000-000000000002")},
		Type:     account.TypeLine,
		Name:     "cached account",
	}
	mockCache.EXPECT().AccountGet(gomock.Any(), id).Return(cached, nil).Times(1)

	res, err := accountHandler.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !reflect.DeepEqual(res, cached) {
		t.Errorf("Get() = %+v, want %+v", res, cached)
	}
}
