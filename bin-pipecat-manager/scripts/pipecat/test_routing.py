"""Tests for routing services: setup propagation, StartFrame/EndFrame, lazy-start."""

import asyncio
from unittest.mock import AsyncMock, MagicMock

import pytest

from conftest import _StartFrame, _EndFrame, _Frame, _FrameDirection

from routing_base import RoutingServiceBase
from routing_stt import RoutingSTTService
from routing_tts import RoutingTTSService
from routing_llm import RoutingLLMService


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_child_service():
    """Create a mock child service with async process_frame and setup."""
    svc = MagicMock()
    svc.process_frame = AsyncMock()
    svc.setup = AsyncMock()
    svc.register_function = MagicMock()
    svc.unregister_function = MagicMock()
    return svc


def _make_router(cls, member_ids=("m1", "m2")):
    """Create a routing service with mock child services and a captured push log."""
    services = {mid: _make_child_service() for mid in member_ids}
    router = cls(services)
    pushed = []

    async def capture_push(frame, direction=_FrameDirection.DOWNSTREAM):
        pushed.append((frame, direction))

    router.push_frame = capture_push
    return router, services, pushed


DOWN = _FrameDirection.DOWNSTREAM


# ---------------------------------------------------------------------------
# setup() propagation
# ---------------------------------------------------------------------------

class TestSetupPropagation:
    @pytest.mark.asyncio
    async def test_setup_propagates_to_all_children(self):
        router, services, _ = _make_router(RoutingSTTService)
        setup_obj = MagicMock()

        await router.setup(setup_obj)

        for svc in services.values():
            svc.setup.assert_awaited_once_with(setup_obj)

    @pytest.mark.asyncio
    async def test_setup_propagates_for_tts(self):
        router, services, _ = _make_router(RoutingTTSService)
        setup_obj = MagicMock()

        await router.setup(setup_obj)

        for svc in services.values():
            svc.setup.assert_awaited_once_with(setup_obj)

    @pytest.mark.asyncio
    async def test_setup_propagates_for_llm(self):
        router, services, _ = _make_router(RoutingLLMService)
        setup_obj = MagicMock()

        await router.setup(setup_obj)

        for svc in services.values():
            svc.setup.assert_awaited_once_with(setup_obj)


# ---------------------------------------------------------------------------
# StartFrame handling
# ---------------------------------------------------------------------------

class TestStartFrame:
    @pytest.mark.asyncio
    async def test_startframe_forwarded_to_active_child(self):
        router, services, pushed = _make_router(RoutingSTTService)
        router.set_active_member("m1")
        frame = _StartFrame()

        await router.process_frame(frame, DOWN)

        services["m1"].process_frame.assert_awaited_once_with(frame, DOWN)
        services["m2"].process_frame.assert_not_awaited()
        assert len(pushed) == 0  # child's push_frame handles downstream

    @pytest.mark.asyncio
    async def test_startframe_pushed_downstream_when_no_active(self):
        router, services, pushed = _make_router(RoutingSTTService)
        frame = _StartFrame()

        await router.process_frame(frame, DOWN)

        for svc in services.values():
            svc.process_frame.assert_not_awaited()
        assert len(pushed) == 1
        assert pushed[0][0] is frame

    @pytest.mark.asyncio
    async def test_startframe_stored_for_later_replay(self):
        router, _, _ = _make_router(RoutingSTTService)
        router.set_active_member("m1")
        frame = _StartFrame()

        await router.process_frame(frame, DOWN)

        assert router._start_frame is frame
        assert router._start_direction is DOWN
        assert "m1" in router._started_ids


# ---------------------------------------------------------------------------
# EndFrame handling
# ---------------------------------------------------------------------------

class TestEndFrame:
    @pytest.mark.asyncio
    async def test_endframe_forwarded_to_all_started_children(self):
        router, services, pushed = _make_router(RoutingSTTService)
        router.set_active_member("m1")
        start = _StartFrame()
        await router.process_frame(start, DOWN)

        # Lazy-start m2
        router.set_active_member("m2")
        audio = _Frame()
        await router.process_frame(audio, DOWN)

        # Both m1 and m2 are now in _started_ids
        assert "m1" in router._started_ids
        assert "m2" in router._started_ids

        # Reset mocks for EndFrame assertions
        for svc in services.values():
            svc.process_frame.reset_mock()
        pushed.clear()

        end = _EndFrame()
        await router.process_frame(end, DOWN)

        # Both started children receive EndFrame
        services["m1"].process_frame.assert_awaited_once_with(end, DOWN)
        services["m2"].process_frame.assert_awaited_once_with(end, DOWN)
        # Exactly one EndFrame pushed downstream
        assert len(pushed) == 1
        assert isinstance(pushed[0][0], _EndFrame)

    @pytest.mark.asyncio
    async def test_endframe_pushed_when_no_started_children(self):
        router, services, pushed = _make_router(RoutingSTTService)
        end = _EndFrame()

        await router.process_frame(end, DOWN)

        for svc in services.values():
            svc.process_frame.assert_not_awaited()
        assert len(pushed) == 1
        assert isinstance(pushed[0][0], _EndFrame)

    @pytest.mark.asyncio
    async def test_endframe_suppressed_in_child_push(self):
        """When EndFrame is forwarded to children, their monkey-patched push_frame
        must NOT propagate EndFrame downstream (suppression flag active)."""
        router, services, pushed = _make_router(RoutingSTTService)
        router.set_active_member("m1")
        start = _StartFrame()
        await router.process_frame(start, DOWN)

        # Simulate child pushing EndFrame during its own EndFrame processing
        async def child_pushes_endframe(frame, direction):
            if isinstance(frame, _EndFrame):
                # Call the monkey-patched push_frame (should be suppressed)
                await services["m1"].push_frame(_EndFrame(), DOWN)

        services["m1"].process_frame = child_pushes_endframe
        pushed.clear()

        end = _EndFrame()
        await router.process_frame(end, DOWN)

        # Only one EndFrame downstream (from router itself), child's push suppressed
        assert len(pushed) == 1


# ---------------------------------------------------------------------------
# Lazy-start (_ensure_started)
# ---------------------------------------------------------------------------

class TestLazyStart:
    @pytest.mark.asyncio
    async def test_lazy_start_on_member_switch(self):
        router, services, pushed = _make_router(RoutingSTTService)
        router.set_active_member("m1")

        start = _StartFrame()
        await router.process_frame(start, DOWN)

        # Switch to m2 and send a regular frame
        router.set_active_member("m2")
        services["m2"].process_frame.reset_mock()

        audio = _Frame()
        await router.process_frame(audio, DOWN)

        # m2 should have received StartFrame (lazy-start) + audio frame
        calls = services["m2"].process_frame.await_args_list
        assert len(calls) == 2
        assert calls[0].args[0] is start  # lazy StartFrame
        assert calls[1].args[0] is audio  # regular frame
        assert "m2" in router._started_ids

    @pytest.mark.asyncio
    async def test_no_double_start_on_already_started(self):
        router, services, _ = _make_router(RoutingSTTService)
        router.set_active_member("m1")

        start = _StartFrame()
        await router.process_frame(start, DOWN)

        services["m1"].process_frame.reset_mock()

        # Send regular frame to already-started m1
        audio = _Frame()
        await router.process_frame(audio, DOWN)

        # Only the audio frame, no duplicate StartFrame
        calls = services["m1"].process_frame.await_args_list
        assert len(calls) == 1
        assert calls[0].args[0] is audio

    @pytest.mark.asyncio
    async def test_suppression_prevents_startframe_downstream(self):
        """During lazy-start, child's push_frame must suppress StartFrame."""
        router, services, pushed = _make_router(RoutingSTTService)
        router.set_active_member("m1")

        start = _StartFrame()
        await router.process_frame(start, DOWN)
        pushed.clear()

        # Switch to m2 — simulate child pushing StartFrame during lazy-start
        async def child_pushes_start(frame, direction):
            if isinstance(frame, _StartFrame):
                await services["m2"].push_frame(_StartFrame(), DOWN)

        services["m2"].process_frame = child_pushes_start
        router.set_active_member("m2")

        audio = _Frame()
        await router.process_frame(audio, DOWN)

        # No StartFrame should have reached downstream
        start_frames = [f for f, d in pushed if isinstance(f, _StartFrame)]
        assert len(start_frames) == 0

    @pytest.mark.asyncio
    async def test_suppression_flag_reset_on_exception(self):
        router, services, _ = _make_router(RoutingSTTService)
        router.set_active_member("m1")

        start = _StartFrame()
        await router.process_frame(start, DOWN)

        # Make m2's process_frame raise
        services["m2"].process_frame = AsyncMock(side_effect=RuntimeError("boom"))
        router.set_active_member("m2")

        audio = _Frame()
        with pytest.raises(RuntimeError, match="boom"):
            await router.process_frame(audio, DOWN)

        # Flag must be reset despite exception
        assert router._suppress_propagation is False


# ---------------------------------------------------------------------------
# _check_started override
# ---------------------------------------------------------------------------

class TestCheckStarted:
    def test_base_stub_returns_false(self):
        """The FrameProcessor stub's _check_started returns False by default."""
        from conftest import _FrameProcessor, _Frame
        fp = _FrameProcessor()
        assert fp._check_started(_Frame()) is False

    def test_routing_service_overrides_to_true(self):
        """RoutingServiceBase overrides _check_started to always return True."""
        router, _, _ = _make_router(RoutingSTTService)
        assert router._check_started(_Frame()) is True


# ---------------------------------------------------------------------------
# set_active_member behavior
# ---------------------------------------------------------------------------

class TestSetActiveMember:
    def test_stt_ignores_unknown_member(self):
        router, _, _ = _make_router(RoutingSTTService)
        router.set_active_member("m1")
        router.set_active_member("unknown")
        assert router._active_id == "m1"

    def test_tts_ignores_unknown_member(self):
        router, _, _ = _make_router(RoutingTTSService)
        router.set_active_member("m1")
        router.set_active_member("unknown")
        assert router._active_id == "m1"

    def test_llm_raises_on_unknown_member(self):
        router, _, _ = _make_router(RoutingLLMService)
        with pytest.raises(ValueError, match="Unknown member_id"):
            router.set_active_member("unknown")


# ---------------------------------------------------------------------------
# LLM-specific methods
# ---------------------------------------------------------------------------

class TestLLMSpecific:
    def test_register_function_delegates_to_all(self):
        router, services, _ = _make_router(RoutingLLMService)
        handler = MagicMock()

        router.register_function(function_name="test_fn", handler=handler)

        for svc in services.values():
            svc.register_function.assert_called_once_with(
                function_name="test_fn",
                handler=handler,
                start_callback=None,
                cancel_on_interruption=True,
            )

    def test_unregister_function_delegates_to_all(self):
        router, services, _ = _make_router(RoutingLLMService)

        router.unregister_function("test_fn")

        for svc in services.values():
            svc.unregister_function.assert_called_once_with("test_fn")

    def test_active_service_returns_current(self):
        router, services, _ = _make_router(RoutingLLMService)
        router.set_active_member("m2")
        assert router.active_service is services["m2"]

    def test_active_service_returns_none_when_not_set(self):
        router, _, _ = _make_router(RoutingLLMService)
        assert router.active_service is None


# ---------------------------------------------------------------------------
# Regular frame routing
# ---------------------------------------------------------------------------

class TestRegularFrameRouting:
    @pytest.mark.asyncio
    async def test_regular_frame_delegated_to_active(self):
        router, services, pushed = _make_router(RoutingSTTService)
        router.set_active_member("m1")
        # Simulate start so _ensure_started doesn't fire
        router._started_ids.add("m1")

        audio = _Frame()
        await router.process_frame(audio, DOWN)

        services["m1"].process_frame.assert_awaited_once_with(audio, DOWN)
        services["m2"].process_frame.assert_not_awaited()
        assert len(pushed) == 0

    @pytest.mark.asyncio
    async def test_regular_frame_pushed_when_no_active(self):
        router, services, pushed = _make_router(RoutingSTTService)

        audio = _Frame()
        await router.process_frame(audio, DOWN)

        for svc in services.values():
            svc.process_frame.assert_not_awaited()
        assert len(pushed) == 1
        assert pushed[0][0] is audio
