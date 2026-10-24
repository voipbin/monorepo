package cachehandler

import (
	"context"
	"errors"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/models/recordingpeak"
)

// fakeRedis is an in-memory redis hook. It answers get/set/del/ping without any
// network connection and records every command it receives.
type fakeRedis struct {
	mu   sync.Mutex
	data map[string][]byte

	calls [][]any

	failErr error // returned for every command when set
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{data: map[string][]byte{}}
}

func (f *fakeRedis) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, errors.New("the fake redis must not dial")
	}
}

func (f *fakeRedis) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		return errors.New("the fake redis does not support pipelines")
	}
}

func (f *fakeRedis) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		f.mu.Lock()
		defer f.mu.Unlock()

		f.calls = append(f.calls, cmd.Args())
		if f.failErr != nil {
			return f.failErr
		}

		switch cmd.Name() {
		case "get":
			key, _ := cmd.Args()[1].(string)
			v, ok := f.data[key]
			if !ok {
				return redis.Nil
			}
			cmd.(*redis.StringCmd).SetVal(string(v))

		case "set":
			key, _ := cmd.Args()[1].(string)
			v, _ := cmd.Args()[2].([]byte)
			f.data[key] = v
			cmd.(*redis.StatusCmd).SetVal("OK")

		case "del":
			key, _ := cmd.Args()[1].(string)
			_, ok := f.data[key]
			delete(f.data, key)
			if ok {
				cmd.(*redis.IntCmd).SetVal(1)
			}

		case "ping":
			cmd.(*redis.StatusCmd).SetVal("PONG")

		default:
			return errors.New("unexpected command: " + cmd.Name())
		}

		return nil
	}
}

func newTestHandler(t *testing.T) (*handler, *fakeRedis) {
	t.Helper()

	f := newFakeRedis()
	c := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	c.AddHook(f)
	t.Cleanup(func() { _ = c.Close() })

	return &handler{Cache: c}, f
}

func Test_FileSetGetDel(t *testing.T) {
	ctx := context.Background()
	h, f := newTestHandler(t)

	tmExpire := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	in := &file.File{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002"),
		},
		AccountID:        uuid.FromStringOrNil("11111111-0000-0000-0000-000000000003"),
		ReferenceType:    file.ReferenceTypeRecording,
		ReferenceID:      uuid.FromStringOrNil("11111111-0000-0000-0000-000000000004"),
		Name:             "name",
		Detail:           "detail",
		BucketName:       "bucket",
		Filename:         "a.wav",
		Filepath:         "recording/a.wav",
		Filesize:         1234,
		URIDownload:      "https://example.invalid/a.wav",
		TMDownloadExpire: &tmExpire,
	}

	if err := h.FileSet(ctx, in); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the key and the TTL must be exactly what other readers of the cache expect.
	if len(f.calls) != 1 {
		t.Fatalf("Wrong match. expect: 1 command, got: %d", len(f.calls))
	}
	call := f.calls[0]
	if len(call) != 5 || call[0] != "set" || call[1] != "storage:file:11111111-0000-0000-0000-000000000001" || call[3] != "ex" || call[4] != int64(86400) {
		t.Errorf("Wrong match. got: %v", call)
	}

	res, err := h.FileGet(ctx, in.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, in) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", in, res)
	}

	if err := h.FileDel(ctx, in.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if _, err := h.FileGet(ctx, in.ID); !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
	}
}

func Test_AccountSetGet(t *testing.T) {
	ctx := context.Background()
	h, f := newTestHandler(t)

	in := &account.Account{
		ID:             uuid.FromStringOrNil("22222222-0000-0000-0000-000000000001"),
		CustomerID:     uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
		TotalFileCount: 7,
		TotalFileSize:  4096,
	}

	if err := h.AccountSet(ctx, in); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got := f.calls[0][1]; got != "storage:account:22222222-0000-0000-0000-000000000001" {
		t.Errorf("Wrong match. got key: %v", got)
	}
	if got := f.calls[0][4]; got != int64(86400) {
		t.Errorf("Wrong match. got ttl: %v", got)
	}

	res, err := h.AccountGet(ctx, in.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, in) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", in, res)
	}

	// an account that was never cached is a miss, not an empty account.
	_, err = h.AccountGet(ctx, uuid.FromStringOrNil("22222222-0000-0000-0000-0000000000ff"))
	if !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
	}
}

func Test_RecordingPeaksSetGet(t *testing.T) {
	ctx := context.Background()
	h, f := newTestHandler(t)

	id := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001")
	in := map[string]recordingpeak.RecordingFilePeak{
		"call_in.wav":  {Peaks: []float64{0.1, 0.5, 1}, Duration: 12.5},
		"call_out.wav": {Peaks: []float64{}, Duration: 0},
	}

	if err := h.RecordingPeaksSet(ctx, id, in); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got := f.calls[0][1]; got != "storage:recording_peaks:33333333-0000-0000-0000-000000000001" {
		t.Errorf("Wrong match. got key: %v", got)
	}
	if got := f.calls[0][4]; got != int64(86400) {
		t.Errorf("Wrong match. got ttl: %v", got)
	}

	res, err := h.RecordingPeaksGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, in) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", in, res)
	}

	res, err = h.RecordingPeaksGet(ctx, uuid.FromStringOrNil("33333333-0000-0000-0000-0000000000ff"))
	if !errors.Is(err, redis.Nil) || res != nil {
		t.Errorf("Wrong match. expect: nil/redis.Nil, got: %v/%v", res, err)
	}
}

func Test_serialize_errors(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000001")
	errBroken := errors.New("simulated cache failure")

	tests := []struct {
		name string
		run  func(h *handler) error
	}{
		{"FileSet", func(h *handler) error { return h.FileSet(ctx, &file.File{Identity: commonidentity.Identity{ID: id}}) }},
		{"FileGet", func(h *handler) error { _, err := h.FileGet(ctx, id); return err }},
		{"FileDel", func(h *handler) error { return h.FileDel(ctx, id) }},
		{"AccountSet", func(h *handler) error { return h.AccountSet(ctx, &account.Account{ID: id}) }},
		{"AccountGet", func(h *handler) error { _, err := h.AccountGet(ctx, id); return err }},
		{"RecordingPeaksSet", func(h *handler) error {
			return h.RecordingPeaksSet(ctx, id, map[string]recordingpeak.RecordingFilePeak{})
		}},
		{"RecordingPeaksGet", func(h *handler) error { _, err := h.RecordingPeaksGet(ctx, id); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newTestHandler(t)
			f.failErr = errBroken

			if err := tt.run(h); !errors.Is(err, errBroken) {
				t.Errorf("Wrong match. expect: %v, got: %v", errBroken, err)
			}
		})
	}
}

func Test_getSerialize_invalidJSON(t *testing.T) {
	ctx := context.Background()
	h, f := newTestHandler(t)

	id := uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001")
	f.data["storage:file:"+id.String()] = []byte("{not json")

	res, err := h.FileGet(ctx, id)
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: an unmarshal error, got: %v/%v", res, err)
	}
}

func Test_setSerialize_unmarshalable(t *testing.T) {
	h, f := newTestHandler(t)

	// a channel cannot be marshaled; nothing must reach the cache.
	if err := h.setSerialize(context.Background(), "key", make(chan int)); err == nil {
		t.Errorf("Wrong match. expect: a marshal error, got: nil")
	}
	if len(f.calls) != 0 {
		t.Errorf("Wrong match. expect: no command, got: %v", f.calls)
	}
}
