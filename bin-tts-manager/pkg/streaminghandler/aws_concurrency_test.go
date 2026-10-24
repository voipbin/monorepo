package streaminghandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/message"

	"go.uber.org/mock/gomock"
)

// newLocalPCMServer returns a local Polly stand-in that answers every request
// with the given amount of PCM audio.
func newLocalPCMServer(t *testing.T, size int) *httptest.Server {
	t.Helper()

	pcm := make([]byte, size)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write(pcm)
	}))
	t.Cleanup(srv.Close)

	return srv
}

// callRecovered runs fn in its own goroutine and reports whether it panicked,
// so a failing assertion does not take the whole test binary down.
func callRecovered(fn func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	fn()

	return false
}

func Test_awsHandler_SayFinish_calledTwice(t *testing.T) {
	cf := newTestAWSConfig(nil)
	h := &awsHandler{}

	for i := 0; i < 2; i++ {
		var err error
		if callRecovered(func() { err = h.SayFinish(cf) }) {
			t.Fatalf("SayFinish call %d must not panic", i+1)
		}
		if err != nil {
			t.Fatalf("SayFinish call %d returned an error: %v", i+1, err)
		}
	}

	if _, ok := <-cf.audioCh; ok {
		t.Error("the audio channel must be closed")
	}
	if !cf.Message.Finish {
		t.Error("Finish flag must be set")
	}
}

func Test_awsHandler_SayAdd_afterSayFinish(t *testing.T) {
	srv := newLocalPCMServer(t, 3000)
	cf := newTestAWSConfig(newLocalPollyClient(srv.URL))
	h := &awsHandler{}

	if err := h.SayFinish(cf); err != nil {
		t.Fatalf("SayFinish returned an error: %v", err)
	}

	// repeat the call to cover every select outcome
	for i := 0; i < 50; i++ {
		var err error
		if callRecovered(func() { err = h.SayAdd(cf, "late text") }) {
			t.Fatalf("SayAdd after SayFinish must not panic. iteration: %d", i)
		}
		if err == nil {
			t.Fatalf("SayAdd after SayFinish must return an error. iteration: %d", i)
		}
	}

	if cf.Message.TotalCount != 0 || cf.Message.TotalMessage != "" {
		t.Errorf("a rejected SayAdd must not be counted. count: %d, message: %q", cf.Message.TotalCount, cf.Message.TotalMessage)
	}
}

func Test_AWSConfig_send_afterClose(t *testing.T) {
	cf := newTestAWSConfig(nil)

	// the session context stays active; only the channel state is closed
	cf.mu.Lock()
	cf.closed = true
	close(cf.audioCh)
	cf.mu.Unlock()

	for i := 0; i < 50; i++ {
		var err error
		if callRecovered(func() { err = cf.send([]byte{1}) }) {
			t.Fatalf("send on a closed session must not panic. iteration: %d", i)
		}
		if err == nil {
			t.Fatalf("send on a closed session must return an error. iteration: %d", i)
		}
	}
}

func Test_AWSConfig_send_afterCancel(t *testing.T) {
	cf := newTestAWSConfig(nil)
	cf.Cancel()

	for i := 0; i < 50; i++ {
		if err := cf.send([]byte{1}); err == nil {
			t.Fatalf("send on a cancelled session must return an error. iteration: %d", i)
		}
	}
	if got := len(cf.audioCh); got != 0 {
		t.Errorf("nothing may be queued on a cancelled session. queued: %d", got)
	}
}

func Test_AWSConfig_send_concurrentWithSayFinish(t *testing.T) {
	cf := newTestAWSConfig(nil)
	h := &awsHandler{}

	// drain so that senders keep running until the session ends
	go func() {
		for range cf.audioCh {
		}
	}()

	const senders = 8
	var wg sync.WaitGroup
	panics := make(chan struct{}, senders+1)

	for i := 0; i < senders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if callRecovered(func() {
				for j := 0; j < 200; j++ {
					_ = cf.send([]byte{1})
				}
			}) {
				panics <- struct{}{}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if callRecovered(func() { _ = h.SayFinish(cf) }) {
			panics <- struct{}{}
		}
	}()

	waitDone := make(chan struct{})
	go func() { wg.Wait(); close(waitDone) }()
	select {
	case <-waitDone:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent send and SayFinish did not finish")
	}

	if len(panics) != 0 {
		t.Errorf("concurrent send and SayFinish panicked %d time(s)", len(panics))
	}
	if err := cf.send([]byte{1}); err == nil {
		t.Error("send after the session finished must return an error")
	}
}

func Test_awsHandler_SayFlush_afterSayFinish(t *testing.T) {
	cf := newTestAWSConfig(nil)
	h := &awsHandler{}

	cf.audioCh <- []byte{1}
	if err := h.SayFinish(cf); err != nil {
		t.Fatalf("SayFinish returned an error: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- h.SayFlush(cf) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("SayFlush returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SayFlush did not return on a finished session")
	}
}

func Test_awsHandler_SayFlush_emptyClosedChannel(t *testing.T) {
	cf := newTestAWSConfig(nil)
	h := &awsHandler{}

	// the channel is closed and empty: only the receive status tells it apart
	// from an open, empty channel. Keep the context active to rule it out.
	cf.mu.Lock()
	cf.closed = true
	close(cf.audioCh)
	cf.mu.Unlock()

	done := make(chan error, 1)
	go func() { done <- h.SayFlush(cf) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("SayFlush returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SayFlush did not return on a closed channel")
	}
}

func Test_awsHandler_SayFlush_drainsPendingAudio(t *testing.T) {
	cf := newTestAWSConfig(nil)
	h := &awsHandler{}

	cf.audioCh <- []byte{1}
	cf.audioCh <- []byte{2}

	if err := h.SayFlush(cf); err != nil {
		t.Fatalf("SayFlush returned an error: %v", err)
	}
	if got := len(cf.audioCh); got != 0 {
		t.Errorf("the audio channel must be drained. remaining: %d", got)
	}
}

func Test_awsHandler_SayAdd_blockedOnFullBuffer_releasedBySessionEnd(t *testing.T) {
	tests := []struct {
		name string
		end  func(h *awsHandler, cf *AWSConfig) error
	}{
		{
			name: "released by SayFinish",
			end:  func(h *awsHandler, cf *AWSConfig) error { return h.SayFinish(cf) },
		},
		{
			name: "released by SayStop",
			end:  func(h *awsHandler, cf *AWSConfig) error { return h.SayStop(cf) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newLocalPCMServer(t, 100)
			cf := newTestAWSConfig(newLocalPollyClient(srv.URL))
			h := &awsHandler{}

			// fill the buffer so the next chunk blocks the sender
			for i := 0; i < defaultAWSAudioChBuffer; i++ {
				cf.audioCh <- []byte{1}
			}

			addDone := make(chan error, 2)
			for i := 0; i < 2; i++ {
				go func() { addDone <- h.SayAdd(cf, "text") }()
			}

			// let at least one sender reach the blocking send
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if !cf.mu.TryLock() {
					break
				}
				cf.mu.Unlock()
				runtime.Gosched()
			}

			endDone := make(chan error, 1)
			go func() { endDone <- tt.end(h, cf) }()

			select {
			case err := <-endDone:
				if err != nil {
					t.Fatalf("ending the session returned an error: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("ending the session did not return while a sender was blocked")
			}

			for i := 0; i < 2; i++ {
				select {
				case err := <-addDone:
					if err == nil {
						t.Error("a blocked SayAdd must return an error once the session ended")
					}
				case <-time.After(5 * time.Second):
					t.Fatal("a blocked SayAdd was not released")
				}
			}
		})
	}
}

func Test_awsHandler_SayAdd_concurrentWithSayFinish(t *testing.T) {
	srv := newLocalPCMServer(t, 9000)
	cf := newTestAWSConfig(newLocalPollyClient(srv.URL))
	h := &awsHandler{}

	// keep the consumer side alive so senders make progress until the end
	go func() {
		for range cf.audioCh {
		}
	}()

	const senders = 8
	var wg sync.WaitGroup
	panics := make(chan struct{}, senders+1)

	for i := 0; i < senders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if callRecovered(func() { _ = h.SayAdd(cf, "x") }) {
				panics <- struct{}{}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if callRecovered(func() { _ = h.SayFinish(cf) }) {
			panics <- struct{}{}
		}
	}()

	waitDone := make(chan struct{})
	go func() { wg.Wait(); close(waitDone) }()
	select {
	case <-waitDone:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent SayAdd and SayFinish did not finish")
	}

	if len(panics) != 0 {
		t.Errorf("concurrent SayAdd and SayFinish panicked %d time(s)", len(panics))
	}
}

func Test_awsHandler_SayAdd_countsEveryCall(t *testing.T) {
	srv := newLocalPCMServer(t, 100)
	cf := newTestAWSConfig(newLocalPollyClient(srv.URL))
	h := &awsHandler{}

	go func() {
		for range cf.audioCh {
		}
	}()
	t.Cleanup(func() { _ = h.SayFinish(cf) })

	const calls = 20
	var wg sync.WaitGroup
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.SayAdd(cf, "ab"); err != nil {
				t.Errorf("SayAdd returned an error: %v", err)
			}
		}()
	}
	wg.Wait()

	cf.muMsg.Lock()
	defer cf.muMsg.Unlock()
	if cf.Message.TotalCount != calls {
		t.Errorf("Wrong TotalCount. expected: %d, got: %d", calls, cf.Message.TotalCount)
	}
	if len(cf.Message.TotalMessage) != calls*2 {
		t.Errorf("Wrong TotalMessage length. expected: %d, got: %d", calls*2, len(cf.Message.TotalMessage))
	}
}

func Test_awsHandler_runProcess_publishesMessageSnapshot(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	cf := newTestAWSConfig(nil)
	t.Cleanup(cf.Cancel)

	started := make(chan *message.Message, 1)
	finished := make(chan *message.Message, 1)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypePlayStarted, gomock.Any()).Do(
		func(ctx context.Context, eventType string, data any) {
			m, _ := data.(*message.Message)
			started <- m
		},
	)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypePlayFinished, gomock.Any()).Do(
		func(ctx context.Context, eventType string, data any) {
			m, _ := data.(*message.Message)
			finished <- m
		},
	)
	h := &awsHandler{notifyHandler: mockNotify}

	cf.Message.TotalCount = 3
	if err := h.SayFinish(cf); err != nil {
		t.Fatalf("SayFinish returned an error: %v", err)
	}

	go h.runProcess(cf)

	var gotStarted, gotFinished *message.Message
	for i := 0; i < 2; i++ {
		select {
		case gotStarted = <-started:
		case gotFinished = <-finished:
		case <-time.After(5 * time.Second):
			t.Fatal("runProcess did not publish its events in time")
		}
	}

	if gotStarted == nil || gotFinished == nil {
		t.Fatal("the published data must be a *message.Message")
	}
	if gotStarted == cf.Message || gotFinished == cf.Message {
		t.Error("the published message must be a copy of the session message")
	}
	if !gotFinished.Finish || gotFinished.TotalCount != 3 {
		t.Errorf("the finished event must reflect the session state. finish: %v, count: %d", gotFinished.Finish, gotFinished.TotalCount)
	}

	// changing the session message afterwards must not alter what was published
	cf.muMsg.Lock()
	cf.Message.TotalCount = 99
	cf.muMsg.Unlock()
	if gotFinished.TotalCount != 3 {
		t.Errorf("the published message changed afterwards. count: %d", gotFinished.TotalCount)
	}
}
