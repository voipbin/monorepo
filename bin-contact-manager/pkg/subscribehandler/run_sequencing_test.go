package subscribehandler

import (
	"fmt"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// The tests below pin the Run() migration sequence of VOIP-1406: queue create ->
// fanout subscribes -> topic exchange declare -> pattern binds -> fanout unbinds ->
// consume. The bind/unbind block MUST complete synchronously inside Run(), before
// the async ConsumeMessage goroutine starts: QueueBind/QueueUnbind and
// ConsumeMessage's internal channel.Consume() share the same underlying AMQP channel
// for a given queue, and racing them makes the broker close the channel with a 503
// (VOIP-1258 PR #1101 post-deploy incident, 2026-07-14).

func testSubscribeTargets() []string {
	return []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
}

// Test_Run_sequencing_success verifies the full happy path in strict order:
// QueueCreate -> each fanout QueueSubscribe -> TopicCreateWithKind -> QueueBind for
// every topic pattern -> QueueUnbind for every fanout target.
func Test_Run_sequencing_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameContactSubscribe)
	subscribeTargets := testSubscribeTargets()

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, target).Return(nil))
	}
	calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil))
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	for _, target := range fanoutUnbindTargets {
		calls = append(calls, mockSock.EXPECT().QueueUnbind(queueName, "", target, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run(); it may or may not have
	// been scheduled by the time Run() returns.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameContactManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, subscribeTargets, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

// Test_Run_sequencing_declareFailure verifies failure path (a): if the topic
// exchange declare fails, Run() performs ZERO topic binds and ZERO fanout unbinds
// (the service stays fully on fanout), and Run() still succeeds.
func Test_Run_sequencing_declareFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameContactSubscribe)
	subscribeTargets := testSubscribeTargets()

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, target).Return(nil))
	}
	calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(fmt.Errorf("declare failed")))
	gomock.InOrder(calls...)

	mockSock.EXPECT().QueueBind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockSock.EXPECT().QueueUnbind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameContactManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, subscribeTargets, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

// Test_Run_sequencing_bindFailure_noRollbackNeeded verifies failure path (b) for a
// single-pattern service: if binding the FIRST (and only) pattern fails, nothing was
// bound yet, so Run() performs ZERO rollback unbinds and ZERO fanout unbinds (the
// service stays fully on fanout), and Run() still succeeds.
func Test_Run_sequencing_bindFailure_noRollbackNeeded(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameContactSubscribe)
	subscribeTargets := testSubscribeTargets()

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, target).Return(nil))
	}
	calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil))
	calls = append(calls, mockSock.EXPECT().QueueBind(queueName, topicPatterns[0], string(commonoutline.QueueNameEvent), false, nil).Return(fmt.Errorf("bind failed")))
	gomock.InOrder(calls...)

	// the bound set is empty when the first bind fails: zero rollback unbinds and
	// zero fanout unbinds.
	mockSock.EXPECT().QueueUnbind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameContactManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, subscribeTargets, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

// Test_Run_sequencing_fanoutUnbindFailure_continues verifies failure path (c): if a
// fanout unbind fails after all patterns were bound, it is logged CRITICAL and Run()
// still succeeds (double delivery is tolerated; loss is not).
func Test_Run_sequencing_fanoutUnbindFailure_continues(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameContactSubscribe)
	subscribeTargets := testSubscribeTargets()

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, target).Return(nil))
	}
	calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil))
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	for _, target := range fanoutUnbindTargets {
		calls = append(calls, mockSock.EXPECT().QueueUnbind(queueName, "", target, nil).Return(fmt.Errorf("unbind failed")))
	}
	gomock.InOrder(calls...)

	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameContactManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, subscribeTargets, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}
