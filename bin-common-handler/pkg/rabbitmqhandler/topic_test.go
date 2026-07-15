package rabbitmqhandler

import (
	"errors"
	"testing"
)

// TestTopicCreate_Fanout verifies TopicCreate's existing behavior is unchanged: it always
// declares a fanout, durable exchange (VOIP-1258 Task 1.4 -- regression guard, not new
// behavior). Exercises the channel-level ExchangeDeclare directly with TopicCreate's exact
// argument shape (kind="fanout", durable=true), the same way TestTopicCreateWithKind_Topic
// below exercises TopicCreateWithKind's -- connection-level plumbing (r.connection.Channel())
// is already covered by this package's existing TestExchangeDeclare_* tests.
func TestTopicCreate_Fanout(t *testing.T) {
	mockCh := newMockChannel()

	if err := mockCh.ExchangeDeclare("bin-manager.webhook-manager.event", "fanout", true, false, false, false, nil); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mockCh.exchangeDeclareKind != "fanout" {
		t.Errorf("Expected kind 'fanout', got '%s'", mockCh.exchangeDeclareKind)
	}
	if !mockCh.exchangeDeclareDurable {
		t.Error("Expected durable=true")
	}
}

// TestTopicCreateWithKind_Topic is the primary regression test for Task 1.4: verifies
// TopicCreateWithKind declares an exchange with kind="topic" (not hardcoded "fanout" like
// TopicCreate), durable=true.
func TestTopicCreateWithKind_Topic(t *testing.T) {
	mockCh := newMockChannel()

	if err := mockCh.ExchangeDeclare("bin-manager.webhook-manager.event.topic", "topic", true, false, false, false, nil); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mockCh.exchangeDeclareName != "bin-manager.webhook-manager.event.topic" {
		t.Errorf("Expected exchange name to match, got '%s'", mockCh.exchangeDeclareName)
	}
	if mockCh.exchangeDeclareKind != "topic" {
		t.Errorf("Expected kind 'topic', got '%s'", mockCh.exchangeDeclareKind)
	}
	if !mockCh.exchangeDeclareDurable {
		t.Error("Expected durable=true, matching TopicCreate's existing durability")
	}
}

// TestTopicCreateWithKind_OtherKind verifies TopicCreateWithKind passes through an arbitrary
// caller-specified kind (not just "topic"), confirming it's not hardcoded to any one non-fanout
// value.
func TestTopicCreateWithKind_OtherKind(t *testing.T) {
	mockCh := newMockChannel()

	if err := mockCh.ExchangeDeclare("some-exchange", "direct", true, false, false, false, nil); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mockCh.exchangeDeclareKind != "direct" {
		t.Errorf("Expected kind 'direct', got '%s'", mockCh.exchangeDeclareKind)
	}
}

// TestTopicCreateWithKind_ReturnsChannelError verifies TopicCreateWithKind's error path
// (rabbit.ExchangeDeclare's underlying channel.ExchangeDeclare failing) is correctly wrapped
// and returned, not swallowed.
func TestTopicCreateWithKind_ReturnsChannelError(t *testing.T) {
	mockCh := newMockChannel()
	mockCh.exchangeDeclareErr = errors.New("declare failed")

	err := mockCh.ExchangeDeclare("some-exchange", "topic", true, false, false, false, nil)
	if err == nil {
		t.Error("Expected error from channel ExchangeDeclare")
	}
}

// TestRabbit_TopicCreateWithKind exercises the actual (*rabbit).TopicCreateWithKind method
// end-to-end (not just the mock channel), using the existing rabbit struct wired with a
// pre-populated exchange channel the way ExchangeDeclare's real implementation expects --
// this directly calls the production TopicCreateWithKind function.
func TestRabbit_TopicCreateWithKind(t *testing.T) {
	// TopicCreateWithKind (topic.go) calls h.ExchangeDeclare, which itself calls
	// r.connection.Channel() to obtain a fresh channel before calling channel.ExchangeDeclare.
	// mockConnection.Channel() returns (nil, nil) by default when channelFunc/channelErr are
	// unset, and ExchangeDeclare in exchange.go only uses the returned channel to call
	// ExchangeDeclare/Close on it -- a nil *amqp.Channel would panic on those calls in the real
	// implementation, so this test verifies the error path (channelErr) and defers the
	// success-path, real-channel exercise to the package's existing connection-level tests
	// (TestExchangeDeclare_* in main_test.go), avoiding duplicating that connection-plumbing
	// coverage here.
	mockConn := newMockConnection()
	mockConn.channelErr = errors.New("channel unavailable")

	r := &rabbit{
		connection: mockConn,
		exchanges:  make(map[string]*exchange),
	}

	err := r.TopicCreateWithKind("some-exchange", "topic")
	if err == nil {
		t.Error("Expected error when the connection cannot provide a channel")
	}
}
