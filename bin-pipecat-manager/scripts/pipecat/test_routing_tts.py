"""Tests for RoutingTTSService.

Covers lifecycle frame propagation, active member routing, fallback behavior,
and push_frame interception.
"""
import sys
import enum
import pytest
from unittest.mock import MagicMock, AsyncMock


class _FrameDirection(enum.Enum):
    DOWNSTREAM = "downstream"
    UPSTREAM = "upstream"


class _StubFrameProcessor:
    """Minimal stub replacing pipecat's FrameProcessor for unit tests."""
    def __init__(self, **kwargs):
        pass

    async def setup(self, setup):
        pass

    async def cleanup(self):
        pass

    async def process_frame(self, frame, direction):
        pass

    async def push_frame(self, frame, direction=None):
        pass


_fp_mod = sys.modules["pipecat.processors.frame_processor"]
_fp_mod.FrameProcessor = _StubFrameProcessor
_fp_mod.FrameDirection = _FrameDirection

if "routing_tts" in sys.modules:
    del sys.modules["routing_tts"]
from routing_tts import RoutingTTSService


def _make_services():
    """Create two mock TTS services."""
    svc_a = MagicMock()
    svc_a.process_frame = AsyncMock()
    svc_a.setup = AsyncMock()
    svc_a.cleanup = AsyncMock()

    svc_b = MagicMock()
    svc_b.process_frame = AsyncMock()
    svc_b.setup = AsyncMock()
    svc_b.cleanup = AsyncMock()

    return {"member-a": svc_a, "member-b": svc_b}


class TestSetActiveMember:
    def test_set_valid_member(self):
        services = _make_services()
        routing = RoutingTTSService(services)
        routing.set_active_member("member-a")
        assert routing._active_id == "member-a"

    def test_unknown_member_keeps_previous(self):
        """Unknown member preserves the previous active TTS (safer than silence)."""
        services = _make_services()
        routing = RoutingTTSService(services)
        routing.set_active_member("member-a")
        routing.set_active_member("nonexistent")
        assert routing._active_id == "member-a"

    def test_switch_between_members(self):
        services = _make_services()
        routing = RoutingTTSService(services)
        routing.set_active_member("member-a")
        routing.set_active_member("member-b")
        assert routing._active_id == "member-b"


class TestProcessFrame:
    @pytest.mark.asyncio
    async def test_routes_to_active_service(self):
        services = _make_services()
        routing = RoutingTTSService(services)
        routing.set_active_member("member-a")
        frame = MagicMock()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)

        services["member-a"].process_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)
        services["member-b"].process_frame.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_passthrough_when_no_active(self):
        services = _make_services()
        routing = RoutingTTSService(services)
        frame = MagicMock()
        routing.push_frame = AsyncMock()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)
        routing.push_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)


class TestLifecycleFramePropagation:
    @pytest.mark.asyncio
    async def test_start_frame_propagates_to_all_services(self):
        _frames_mod = sys.modules["pipecat.frames.frames"]
        services = _make_services()
        routing = RoutingTTSService(services)
        routing.set_active_member("member-a")
        frame = _frames_mod.StartFrame()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)

        for svc in services.values():
            svc.process_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)

    @pytest.mark.asyncio
    async def test_cancel_frame_propagates_to_all_services(self):
        _frames_mod = sys.modules["pipecat.frames.frames"]
        services = _make_services()
        routing = RoutingTTSService(services)
        frame = _frames_mod.CancelFrame()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)

        for svc in services.values():
            svc.process_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)

    @pytest.mark.asyncio
    async def test_end_frame_propagates_to_all_services(self):
        _frames_mod = sys.modules["pipecat.frames.frames"]
        services = _make_services()
        routing = RoutingTTSService(services)
        frame = _frames_mod.EndFrame()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)

        for svc in services.values():
            svc.process_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)

    @pytest.mark.asyncio
    async def test_lifecycle_frame_without_active_member(self):
        _frames_mod = sys.modules["pipecat.frames.frames"]
        services = _make_services()
        routing = RoutingTTSService(services)
        frame = _frames_mod.StartFrame()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)

        for svc in services.values():
            svc.process_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)

    @pytest.mark.asyncio
    async def test_non_lifecycle_frame_not_broadcast(self):
        services = _make_services()
        routing = RoutingTTSService(services)
        routing.set_active_member("member-a")
        frame = MagicMock()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)

        services["member-a"].process_frame.assert_awaited_once()
        services["member-b"].process_frame.assert_not_awaited()


class TestPushFrameRouting:
    def test_push_frame_overridden_on_init(self):
        services = _make_services()
        original_push_a = services["member-a"].push_frame
        RoutingTTSService(services)
        assert services["member-a"].push_frame is not original_push_a
        assert callable(services["member-a"].push_frame)


class TestSetupCleanupPropagation:
    @pytest.mark.asyncio
    async def test_setup_propagates_to_all_services(self):
        services = _make_services()
        routing = RoutingTTSService(services)
        setup_obj = MagicMock()
        await routing.setup(setup_obj)

        for svc in services.values():
            svc.setup.assert_awaited_once_with(setup_obj)

    @pytest.mark.asyncio
    async def test_cleanup_propagates_to_all_services(self):
        services = _make_services()
        routing = RoutingTTSService(services)
        await routing.cleanup()

        for svc in services.values():
            svc.cleanup.assert_awaited_once()
