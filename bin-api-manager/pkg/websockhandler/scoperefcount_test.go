package websockhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestScopeRefCount_BindOnFirstSubscribe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)

	rc := newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic")

	mockSock.EXPECT().QueueBind("pod-queue-1", "customer_id.abc.#", "bin-manager.webhook-manager.event.topic", false, nil).Return(nil)

	err := rc.Acquire("customer_id.abc.#")
	require.NoError(t, err)
}

func TestScopeRefCount_NoRebindOnSecondSubscribe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)

	rc := newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic")

	// QueueBind should only be called once, on the first Acquire.
	mockSock.EXPECT().QueueBind("pod-queue-1", "customer_id.abc.#", "bin-manager.webhook-manager.event.topic", false, nil).Return(nil).Times(1)

	err := rc.Acquire("customer_id.abc.#")
	require.NoError(t, err)

	err = rc.Acquire("customer_id.abc.#")
	require.NoError(t, err)
}

func TestScopeRefCount_UnbindOnLastRelease(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)

	rc := newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic")

	mockSock.EXPECT().QueueBind("pod-queue-1", "customer_id.abc.#", "bin-manager.webhook-manager.event.topic", false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind("pod-queue-1", "customer_id.abc.#", "bin-manager.webhook-manager.event.topic", nil).Return(nil)

	err := rc.Acquire("customer_id.abc.#")
	require.NoError(t, err)

	err = rc.Acquire("customer_id.abc.#")
	require.NoError(t, err)

	err = rc.Release("customer_id.abc.#")
	require.NoError(t, err)

	err = rc.Release("customer_id.abc.#")
	require.NoError(t, err)
}

func TestScopeRefCount_NoUnbindWhileRefsRemain(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)

	rc := newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic")

	// QueueUnbind must NOT be called since only one of the two acquired refs is released.
	mockSock.EXPECT().QueueBind("pod-queue-1", "customer_id.abc.#", "bin-manager.webhook-manager.event.topic", false, nil).Return(nil).Times(1)

	err := rc.Acquire("customer_id.abc.#")
	require.NoError(t, err)

	err = rc.Acquire("customer_id.abc.#")
	require.NoError(t, err)

	err = rc.Release("customer_id.abc.#")
	require.NoError(t, err)
}
