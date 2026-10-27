package main

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"monorepo/bin-conference-manager/models/conference"
)

var (
	testCustomerID = uuid.FromStringOrNil("5f1c8a6e-0d2b-4c3a-9e47-1b6a2d3c4e5f")
	testConfID     = uuid.FromStringOrNil("7b0d1c52-3f4e-4a7b-9c1d-2e8f6a5b4c3d")
	testBridgeID   = uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-00000000b001")
	testMissingID  = "c9d8e7f6-0000-4000-8000-00000000f001"
)

func progressingConference() *conference.Conference {
	return &conference.Conference{
		Identity:     commonidentity.Identity{ID: testConfID, CustomerID: testCustomerID},
		ConfbridgeID: testBridgeID,
		Type:         conference.TypeConference,
		Status:       conference.StatusProgressing,
		Name:         "weekly sync",
		Detail:       "synthetic conference",
	}
}

func Test_initHandler(t *testing.T) {
	t.Run("handlers are built when the database and the cache are reachable", func(t *testing.T) {
		newEnv(t)

		h, err := initHandler()
		if err != nil || h == nil {
			t.Fatalf("Wrong match. expect: handler, got: %v, err: %v", h, err)
		}
	})

	t.Run("a cache that does not answer stops the initialization", func(t *testing.T) {
		newEnv(t)

		testRedis.Close()
		t.Cleanup(func() {
			if err := testRedis.Restart(); err != nil {
				t.Fatalf("could not restart the stand-in cache server: %v", err)
			}
		})

		h, err := initHandler()
		if err == nil || !strings.Contains(err.Error(), "could not initialize the cache") {
			t.Fatalf("Wrong match. expect: cache error, got: %v", err)
		}
		if h != nil {
			t.Errorf("Wrong match. expect: nil handler, got: %v", h)
		}
	})
}

func Test_get(t *testing.T) {
	t.Run("prints the stored conference", func(t *testing.T) {
		e := newEnv(t)
		cf := e.seed(progressingConference())

		out, err := e.exec("conference", "get", "--id", cf.ID.String())
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := decodeConference(t, out)
		if res.ID != cf.ID || res.Name != "weekly sync" || res.Status != conference.StatusProgressing {
			t.Errorf("Wrong match. expect: id %s, name weekly sync, status progressing, got: %+v", cf.ID, res)
		}
	})

	t.Run("a missing conference is reported", func(t *testing.T) {
		e := newEnv(t)

		out, err := e.exec("conference", "get", "--id", testMissingID)
		if err == nil || !strings.Contains(err.Error(), "failed to retrieve conference") {
			t.Fatalf("Wrong match. expect: retrieve error, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect: no output, got: %s", out)
		}
	})
}

func Test_getByConfbridge(t *testing.T) {
	t.Run("prints the conference of the confbridge", func(t *testing.T) {
		e := newEnv(t)
		cf := e.seed(progressingConference())

		out, err := e.exec("conference", "get-by-confbridge", "--confbridge-id", testBridgeID.String())
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res := decodeConference(t, out); res.ID != cf.ID {
			t.Errorf("Wrong match. expect: %s, got: %s", cf.ID, res.ID)
		}
	})

	t.Run("an unknown confbridge is reported", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.exec("conference", "get-by-confbridge", "--confbridge-id", testMissingID)
		if err == nil || !strings.Contains(err.Error(), "failed to retrieve conference") {
			t.Fatalf("Wrong match. expect: retrieve error, got: %v", err)
		}
	})
}

func Test_list(t *testing.T) {
	t.Run("lists only the conferences of the customer", func(t *testing.T) {
		e := newEnv(t)
		e.seed(progressingConference())

		other := progressingConference()
		other.ID = uuid.Nil
		other.CustomerID = uuid.FromStringOrNil("6a1c8a6e-0d2b-4c3a-9e47-1b6a2d3c4e60")
		other.ConfbridgeID = uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-00000000b002")
		e.seed(other)

		out, err := e.exec("conference", "list", "--customer-id", testCustomerID.String(), "--limit", "10")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if strings.Count(out, `"id":`) != 1 || !strings.Contains(out, testConfID.String()) {
			t.Errorf("Wrong match. expect: only the conference %s, got: %s", testConfID, out)
		}
	})

	t.Run("the limit bounds the number of conferences", func(t *testing.T) {
		e := newEnv(t)
		for i := 0; i < 3; i++ {
			cf := progressingConference()
			cf.ID = uuid.Nil
			cf.ConfbridgeID = uuid.Must(uuid.NewV4())
			e.seed(cf)
		}

		out, err := e.exec("conference", "list", "--customer-id", testCustomerID.String(), "--limit", "2")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if got := strings.Count(out, `"id":`); got != 2 {
			t.Errorf("Wrong match. expect: 2 conferences, got: %d", got)
		}
	})

	t.Run("a database failure is reported", func(t *testing.T) {
		e := newEnv(t)
		_ = e.db.Close()

		_, err := e.exec("conference", "list", "--customer-id", testCustomerID.String())
		if err == nil || !strings.Contains(err.Error(), "failed to retrieve conferences") {
			t.Fatalf("Wrong match. expect: retrieve error, got: %v", err)
		}
	})
}

func Test_update(t *testing.T) {
	t.Run("changes only the given fields", func(t *testing.T) {
		e := newEnv(t)
		e.seed(progressingConference())
		preFlow := "11111111-0000-4000-8000-0000000000a1"
		postFlow := "22222222-0000-4000-8000-0000000000a2"

		out, err := e.exec("conference", "update", "--id", testConfID.String(),
			"--name", "renamed", "--detail", "new detail", "--data", `{"k":"v"}`, "--timeout", "120",
			"--pre-flow-id", preFlow, "--post-flow-id", postFlow)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := decodeConference(t, out)
		if res.Name != "renamed" || res.Detail != "new detail" || res.Timeout != 120 ||
			res.Data["k"] != "v" || res.PreFlowID.String() != preFlow || res.PostFlowID.String() != postFlow {
			t.Errorf("Wrong match. expect: all fields changed, got: %+v", res)
		}
		if res.Status != conference.StatusProgressing || res.ConfbridgeID != testBridgeID {
			t.Errorf("Wrong match. expect: untouched status and confbridge, got: %+v", res)
		}
	})

	t.Run("fields that are not given stay unchanged", func(t *testing.T) {
		e := newEnv(t)
		e.seed(progressingConference())

		out, err := e.exec("conference", "update", "--id", testConfID.String(), "--name", "only name")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res := decodeConference(t, out); res.Name != "only name" || res.Detail != "synthetic conference" {
			t.Errorf("Wrong match. expect: name changed and detail kept, got: %+v", res)
		}
	})

	tests := []struct {
		name string

		args []string

		expectErr string
	}{
		{"id is required", []string{"conference", "update"}, "failed to resolve conference ID: Conference ID is required"},
		{"data must be a json object", []string{"conference", "update", "--id", testConfID.String(), "--data", "{bad"}, "invalid JSON for --data"},
		{"pre flow id must be a uuid", []string{"conference", "update", "--id", testConfID.String(), "--pre-flow-id", "bad"}, "invalid format for Pre-flow ID: 'bad' is not a valid UUID"},
		{"post flow id must be a uuid", []string{"conference", "update", "--id", testConfID.String(), "--post-flow-id", "bad"}, "invalid format for Post-flow ID: 'bad' is not a valid UUID"},
		{"a missing conference is reported", []string{"conference", "update", "--id", testMissingID, "--name", "x"}, "failed to update conference"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.seed(progressingConference())

			_, err := e.exec(tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_terminating(t *testing.T) {
	t.Run("moves the conference to terminating", func(t *testing.T) {
		e := newEnv(t)
		e.seed(progressingConference())

		out, err := e.exec("conference", "terminating", "--id", testConfID.String())
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res := decodeConference(t, out); res.Status != conference.StatusTerminating {
			t.Errorf("Wrong match. expect: terminating, got: %s", res.Status)
		}
	})

	t.Run("a missing conference is reported", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.exec("conference", "terminating", "--id", testMissingID)
		if err == nil || !strings.Contains(err.Error(), "failed to terminate conference") {
			t.Fatalf("Wrong match. expect: terminate error, got: %v", err)
		}
	})
}

func Test_delete(t *testing.T) {
	t.Run("terminates and removes the conference and its direct hash", func(t *testing.T) {
		e := newEnv(t)
		cf := progressingConference()
		cf.DirectID = uuid.FromStringOrNil("33333333-0000-4000-8000-0000000000d1")
		e.seed(cf)
		e.on("/v1/directs/", 200, &dmdirect.Direct{Identity: commonidentity.Identity{ID: cf.DirectID}})

		out, err := e.exec("conference", "delete", "--id", testConfID.String())
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := decodeConference(t, out)
		if res.ID != testConfID || res.TMDelete == nil {
			t.Errorf("Wrong match. expect: the deleted conference with tm_delete, got: %+v", res)
		}
		if len(e.requests()) == 0 || !strings.HasPrefix(e.requests()[0], "DELETE /v1/directs/") {
			t.Errorf("Wrong match. expect: the direct delete request first, got: %v", e.requests())
		}
	})

	t.Run("a missing conference is reported", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.exec("conference", "delete", "--id", testMissingID)
		if err == nil || !strings.Contains(err.Error(), "failed to delete conference") {
			t.Fatalf("Wrong match. expect: delete error, got: %v", err)
		}
	})
}

func Test_create(t *testing.T) {
	newCreateEnv := func(t *testing.T, valid bool) *testEnv {
		e := newEnv(t)
		e.on("is_valid_resource_limit", 200, map[string]any{"valid": valid})
		e.on("/v1/confbridges", 200, &cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: testBridgeID}})
		e.on("/v1/directs", 200, &dmdirect.Direct{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("44444444-0000-4000-8000-0000000000d2")},
			Hash:     "synthetic-hash",
		})
		return e
	}

	t.Run("creates the conference with the given id", func(t *testing.T) {
		e := newCreateEnv(t, true)

		out, err := e.exec("conference", "create", "--id", testConfID.String(), "--customer-id", testCustomerID.String(),
			"--type", "connect", "--name", "created", "--timeout", "90")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := decodeConference(t, out)
		if res.ID != testConfID || res.CustomerID != testCustomerID || res.Type != conference.TypeConnect ||
			res.Name != "created" || res.ConfbridgeID != testBridgeID || res.DirectHash != "synthetic-hash" || res.Timeout != 90 {
			t.Errorf("Wrong match. expect: the created conference, got: %+v", res)
		}

		stored, errGet := e.dbh.ConferenceGet(context.Background(), testConfID)
		if errGet != nil || stored.Name != "created" {
			t.Errorf("Wrong match. expect: stored conference, got: %v, err: %v", stored, errGet)
		}
	})

	t.Run("creates the conference with a generated id and flows", func(t *testing.T) {
		e := newCreateEnv(t, true)

		out, err := e.exec("conference", "create", "--customer-id", testCustomerID.String(),
			"--pre-flow-id", "11111111-0000-4000-8000-0000000000a1", "--post-flow-id", "22222222-0000-4000-8000-0000000000a2")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res := decodeConference(t, out)
		if res.ID == uuid.Nil || res.Type != conference.TypeConference ||
			res.PreFlowID.String() != "11111111-0000-4000-8000-0000000000a1" || res.PostFlowID.String() != "22222222-0000-4000-8000-0000000000a2" {
			t.Errorf("Wrong match. expect: generated id and both flows, got: %+v", res)
		}
	})

	t.Run("an exhausted resource limit stops the creation", func(t *testing.T) {
		e := newCreateEnv(t, false)

		_, err := e.exec("conference", "create", "--customer-id", testCustomerID.String())
		if err == nil || !strings.Contains(err.Error(), "failed to create conference: resource limit exceeded") {
			t.Fatalf("Wrong match. expect: limit error, got: %v", err)
		}
		for _, call := range e.requests() {
			if strings.Contains(call, "/v1/confbridges") {
				t.Errorf("Wrong match. expect: no confbridge request after the limit check, got: %v", e.requests())
			}
		}
	})
}

func Test_recording(t *testing.T) {
	recordingID := uuid.FromStringOrNil("55555555-0000-4000-8000-0000000000e1")

	t.Run("start requests the recording and stores its id", func(t *testing.T) {
		e := newEnv(t)
		e.seed(progressingConference())
		e.on("/v1/recordings", 200, &cmrecording.Recording{Identity: commonidentity.Identity{ID: recordingID}})

		out, err := e.exec("conference", "recording-start", "--id", testConfID.String(), "--duration", "60",
			"--activeflow-id", "66666666-0000-4000-8000-0000000000f1", "--on-end-flow-id", "77777777-0000-4000-8000-0000000000f2")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res := decodeConference(t, out); res.RecordingID != recordingID {
			t.Errorf("Wrong match. expect: recording %s, got: %s", recordingID, res.RecordingID)
		}
	})

	t.Run("stop ends a running recording", func(t *testing.T) {
		e := newEnv(t)
		cf := progressingConference()
		cf.RecordingID = recordingID
		e.seed(cf)
		e.on("/v1/recordings/", 200, &cmrecording.Recording{Identity: commonidentity.Identity{ID: recordingID}})

		out, err := e.exec("conference", "recording-stop", "--id", testConfID.String())
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res := decodeConference(t, out); res.RecordingID != uuid.Nil {
			t.Errorf("Wrong match. expect: no current recording, got: %s", res.RecordingID)
		}
	})

	t.Run("update-recording-id stores the recording id", func(t *testing.T) {
		e := newEnv(t)
		e.seed(progressingConference())

		out, err := e.exec("conference", "update-recording-id", "--id", testConfID.String(), "--recording-id", recordingID.String())
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		res := decodeConference(t, out)
		if res.RecordingID != recordingID || len(res.RecordingIDs) != 1 || res.RecordingIDs[0] != recordingID {
			t.Errorf("Wrong match. expect: recording %s stored and listed, got: %+v", recordingID, res)
		}
	})

	tests := []struct {
		name string

		args []string
		seed func(*conference.Conference)

		expectErr string
	}{
		{"start: id is required", []string{"conference", "recording-start"}, nil, "failed to resolve conference ID: Conference ID is required"},
		{"start: activeflow id must be a uuid", []string{"conference", "recording-start", "--id", testConfID.String(), "--activeflow-id", "bad"}, nil, "invalid format for Active flow ID: 'bad' is not a valid UUID"},
		{"start: on end flow id must be a uuid", []string{"conference", "recording-start", "--id", testConfID.String(), "--on-end-flow-id", "bad"}, nil, "invalid format for On-end flow ID: 'bad' is not a valid UUID"},
		{"start: only wav is accepted", []string{"conference", "recording-start", "--id", testConfID.String(), "--format", "mp3"}, nil, "invalid recording format: must be 'wav'"},
		{"start: a running recording is not started twice", []string{"conference", "recording-start", "--id", testConfID.String()}, func(c *conference.Conference) {
			c.RecordingID = uuid.FromStringOrNil("55555555-0000-4000-8000-0000000000e1")
		}, "failed to start recording"},
		{"start: the conference must be progressing", []string{"conference", "recording-start", "--id", testConfID.String()}, func(c *conference.Conference) { c.Status = conference.StatusTerminating }, "failed to start recording"},
		{"stop: id is required", []string{"conference", "recording-stop"}, nil, "failed to resolve conference ID: Conference ID is required"},
		{"stop: nothing is recording", []string{"conference", "recording-stop", "--id", testConfID.String()}, nil, "failed to stop recording"},
		{"update-recording-id: id is required", []string{"conference", "update-recording-id", "--recording-id", testConfID.String()}, nil, "failed to resolve conference ID: Conference ID is required"},
		{"update-recording-id: recording id is required", []string{"conference", "update-recording-id", "--id", testConfID.String()}, nil, "failed to resolve recording ID: Recording ID is required"},
		{"update-recording-id: a missing conference is reported", []string{"conference", "update-recording-id", "--id", testMissingID, "--recording-id", testConfID.String()}, nil, "failed to update recording ID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			cf := progressingConference()
			if tt.seed != nil {
				tt.seed(cf)
			}
			e.seed(cf)

			_, err := e.exec(tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
			}
			if len(e.requests()) != 0 {
				t.Errorf("Wrong match. expect: no request to other services, got: %v", e.requests())
			}
		})
	}
}

func Test_transcribe(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("88888888-0000-4000-8000-0000000000e2")
	hostID := uuid.FromStringOrNil("99999999-0000-4000-8000-0000000000e3")

	t.Run("start requests the transcription and stores its id", func(t *testing.T) {
		e := newEnv(t)
		e.seed(progressingConference())
		e.on("/v1/transcribes", 200, &tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: transcribeID}})

		out, err := e.exec("conference", "transcribe-start", "--id", testConfID.String(), "--lang", "ko-KR")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		res := decodeConference(t, out)
		if res.TranscribeID != transcribeID || len(res.TranscribeIDs) != 1 {
			t.Errorf("Wrong match. expect: transcribe %s stored, got: %+v", transcribeID, res)
		}
	})

	t.Run("stop ends a running transcription", func(t *testing.T) {
		e := newEnv(t)
		cf := progressingConference()
		cf.TranscribeID = transcribeID
		e.seed(cf)
		e.on("/v1/transcribes/", 200, &tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: transcribeID}, HostID: hostID})

		out, err := e.exec("conference", "transcribe-stop", "--id", testConfID.String())
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res := decodeConference(t, out); res.TranscribeID != uuid.Nil {
			t.Errorf("Wrong match. expect: no current transcribe, got: %s", res.TranscribeID)
		}
	})

	tests := []struct {
		name string

		args []string
		seed func(*conference.Conference)

		expectErr string
	}{
		{"start: id is required", []string{"conference", "transcribe-start"}, nil, "failed to resolve conference ID: Conference ID is required"},
		{"start: the conference must be progressing", []string{"conference", "transcribe-start", "--id", testConfID.String()}, func(c *conference.Conference) { c.Status = conference.StatusTerminating }, "failed to start transcription"},
		{"stop: id is required", []string{"conference", "transcribe-stop"}, nil, "failed to resolve conference ID: Conference ID is required"},
		{"stop: nothing is transcribing", []string{"conference", "transcribe-stop", "--id", testConfID.String()}, nil, "failed to stop transcription"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			cf := progressingConference()
			if tt.seed != nil {
				tt.seed(cf)
			}
			e.seed(cf)

			_, err := e.exec(tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
			}
			if len(e.requests()) != 0 {
				t.Errorf("Wrong match. expect: no request to other services, got: %v", e.requests())
			}
		})
	}
}

func Test_idIsRequired(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"get", []string{"conference", "get"}, "failed to resolve conference ID: Conference ID is required"},
		{"get-by-confbridge", []string{"conference", "get-by-confbridge"}, "failed to resolve confbridge ID: Confbridge ID is required"},
		{"list", []string{"conference", "list"}, "failed to resolve customer ID: Customer ID is required"},
		{"delete", []string{"conference", "delete"}, "failed to resolve conference ID: Conference ID is required"},
		{"terminating", []string{"conference", "terminating"}, "failed to resolve conference ID: Conference ID is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			_, err := e.exec(tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Wrong match. expect: %s, got: %v", tt.want, err)
			}
		})
	}
}
