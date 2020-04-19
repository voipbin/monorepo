package dbhandler

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"database/sql"

	log "github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3"
	"github.com/smotes/purse"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

var dbTest *sql.DB = nil // database for test

func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite3", `file::memory:?cache=shared`)
	if err != nil {
		log.Errorf("err: %v", err)
	}
	db.SetMaxOpenConns(1)

	// Load all SQL files from specified directory into a map
	ps, err := purse.New(filepath.Join("../../scripts/database_scripts"))
	if err != nil {
		log.Infof("Err. err: %v", err)
	}
	log.Infof("Script loaded. scripts: %v", ps)

	// Get a file's contents
	contents, ok := ps.Get("table_channels.sql")
	if !ok {
		log.Info("SQL file not loaded")
	}

	ret, err := db.Exec(contents)
	if err != nil {
		log.Errorf("Could not execute the sql. err: %v", err)
	}
	log.Infof("executed sql file. ret: %v", ret)

	dbTest = db
	defer dbTest.Close()

	os.Exit(m.Run())
}

func TestChannelCreate(t *testing.T) {
	type test struct {
		name string

		channel       channel.Channel
		expectChannel channel.Channel
	}

	tests := []test{
		{
			"test normal",
			channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
		{
			"test normal has state",
			channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "fd4ed562-823f-11ea-a6b2-bbfcd3647952",
				State:      "Up",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "fd4ed562-823f-11ea-a6b2-bbfcd3647952",
				State:      "Up",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			resChannel, err := h.ChannelGet(context.Background(), tt.channel.AsteriskID, tt.channel.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, *resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelGet(t *testing.T) {
	type test struct {
		name string

		queryChannel  channel.Channel
		expectChannel channel.Channel
	}

	tests := []test{
		{
			"test normal",
			channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "edcf72a4-8230-11ea-9f7f-ff89da373481",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "edcf72a4-8230-11ea-9f7f-ff89da373481",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			if err := h.ChannelCreate(context.Background(), tt.queryChannel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			resChannel, err := h.ChannelGet(context.Background(), tt.expectChannel.AsteriskID, tt.expectChannel.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok , got: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, *resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelEnd(t *testing.T) {
	type test struct {
		name string

		channel   channel.Channel
		hangup    int
		timestamp string

		expectChannel channel.Channel
	}

	tests := []test{
		{
			"test normal",
			channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "810a31da-8245-11ea-881e-df4110bf6754",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			16,
			"2020-04-18T03:23:20.995000",
			channel.Channel{
				AsteriskID:  "3e:50:6b:43:bb:30",
				ID:          "810a31da-8245-11ea-881e-df4110bf6754",
				HangupCause: 16,
				TMCreate:    "2020-04-18T03:22:17.995000",
				TMUpdate:    "2020-04-18T03:23:20.995000",
				TMEnd:       "2020-04-18T03:23:20.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			// prepare
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.ChannelEnd(context.Background(), tt.channel.AsteriskID, tt.channel.ID, tt.timestamp, tt.hangup); err != nil {
				t.Errorf("Wrong match. expect: ok , got: %v", err)
			}

			resChannel, err := h.ChannelGet(context.Background(), tt.channel.AsteriskID, tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, *resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}
