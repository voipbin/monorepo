"""Tests for RoutingLLMService.

Covers register_function signature compatibility with pipecat's LLMService,
delegation to all wrapped services, active_service property, and error handling.
"""
import sys
import enum
import pytest
from unittest.mock import MagicMock, AsyncMock


# FrameDirection needs to be a real enum for process_frame tests.
class _FrameDirection(enum.Enum):
    DOWNSTREAM = "downstream"
    UPSTREAM = "upstream"


class _StubFrameProcessor:
    """Minimal stub replacing pipecat's FrameProcessor for unit tests."""
    def __init__(self, **kwargs):
        pass

    async def push_frame(self, frame, direction=None):
        pass


# Replace the MagicMock FrameProcessor from conftest with a real class
# so RoutingLLMService can properly inherit from it.
_fp_mod = sys.modules["pipecat.processors.frame_processor"]
_fp_mod.FrameProcessor = _StubFrameProcessor
_fp_mod.FrameDirection = _FrameDirection

# Force reimport routing_llm with the stub FrameProcessor
if "routing_llm" in sys.modules:
    del sys.modules["routing_llm"]
from routing_llm import RoutingLLMService


def _make_services():
    """Create two mock LLM services."""
    svc_a = MagicMock()
    svc_a.process_frame = AsyncMock()
    svc_a.register_function = MagicMock()
    svc_a.unregister_function = MagicMock()

    svc_b = MagicMock()
    svc_b.process_frame = AsyncMock()
    svc_b.register_function = MagicMock()
    svc_b.unregister_function = MagicMock()

    return {"member-a": svc_a, "member-b": svc_b}


class TestSetActiveMember:
    def test_set_valid_member(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        routing.set_active_member("member-a")
        assert routing._active_id == "member-a"

    def test_set_another_valid_member(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        routing.set_active_member("member-b")
        assert routing._active_id == "member-b"

    def test_set_unknown_member_raises(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        with pytest.raises(ValueError, match="Unknown member_id"):
            routing.set_active_member("nonexistent")

    def test_switch_between_members(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        routing.set_active_member("member-a")
        routing.set_active_member("member-b")
        assert routing._active_id == "member-b"


class TestActiveServiceProperty:
    def test_returns_none_when_no_active_member(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        assert routing.active_service is None

    def test_returns_correct_service_after_set(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        routing.set_active_member("member-a")
        assert routing.active_service is services["member-a"]

    def test_returns_correct_service_after_switch(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        routing.set_active_member("member-a")
        routing.set_active_member("member-b")
        assert routing.active_service is services["member-b"]


class TestRegisterFunction:
    """Tests for register_function pipecat LLMService API compatibility.

    FlowManager calls register_function with keyword args including
    cancel_on_interruption. The signature must accept all pipecat kwargs.
    """

    def test_delegates_to_all_services(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        handler = MagicMock()
        routing.register_function(function_name="test_fn", handler=handler)

        for svc in services.values():
            svc.register_function.assert_called_once_with(
                function_name="test_fn",
                handler=handler,
                start_callback=None,
                cancel_on_interruption=True,
            )

    def test_forwards_cancel_on_interruption_false(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        handler = MagicMock()
        routing.register_function(
            function_name="fn", handler=handler, cancel_on_interruption=False,
        )

        for svc in services.values():
            svc.register_function.assert_called_once_with(
                function_name="fn",
                handler=handler,
                start_callback=None,
                cancel_on_interruption=False,
            )

    def test_forwards_start_callback(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        handler = MagicMock()
        start_cb = MagicMock()
        routing.register_function(
            function_name="fn", handler=handler, start_callback=start_cb,
        )

        for svc in services.values():
            svc.register_function.assert_called_once_with(
                function_name="fn",
                handler=handler,
                start_callback=start_cb,
                cancel_on_interruption=True,
            )

    def test_accepts_keyword_only_cancel_on_interruption(self):
        """cancel_on_interruption must be keyword-only (after *)."""
        services = _make_services()
        routing = RoutingLLMService(services)
        handler = MagicMock()
        routing.register_function(function_name="fn", handler=handler, cancel_on_interruption=True)
        assert services["member-a"].register_function.called

    def test_accepts_extra_kwargs(self):
        """Signature includes **kwargs for forward compatibility."""
        services = _make_services()
        routing = RoutingLLMService(services)
        handler = MagicMock()
        # Should not raise even with unknown kwargs
        routing.register_function(function_name="fn", handler=handler, some_future_param="value")


class TestUnregisterFunction:
    def test_delegates_to_all_services(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        routing.unregister_function("test_fn")

        for svc in services.values():
            svc.unregister_function.assert_called_once_with("test_fn")

    def test_suppresses_key_error(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        services["member-a"].unregister_function.side_effect = KeyError("not found")
        routing.unregister_function("missing_fn")
        services["member-b"].unregister_function.assert_called_once()

    def test_suppresses_generic_exception(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        services["member-a"].unregister_function.side_effect = Exception("unexpected")
        routing.unregister_function("fn")
        services["member-b"].unregister_function.assert_called_once()

    def test_both_services_called_even_if_first_fails(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        services["member-a"].unregister_function.side_effect = KeyError("nope")
        routing.unregister_function("fn")
        services["member-a"].unregister_function.assert_called_once_with("fn")
        services["member-b"].unregister_function.assert_called_once_with("fn")


class TestProcessFrame:
    @pytest.mark.asyncio
    async def test_routes_to_active_service(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        routing.set_active_member("member-a")
        frame = MagicMock()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)

        services["member-a"].process_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)
        services["member-b"].process_frame.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_routes_to_switched_service(self):
        services = _make_services()
        routing = RoutingLLMService(services)
        routing.set_active_member("member-a")
        routing.set_active_member("member-b")
        frame = MagicMock()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)

        services["member-b"].process_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)
        services["member-a"].process_frame.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_passthrough_when_no_active(self):
        """When no active member, frames pass through via push_frame."""
        services = _make_services()
        routing = RoutingLLMService(services)
        frame = MagicMock()
        routing.push_frame = AsyncMock()
        await routing.process_frame(frame, _FrameDirection.DOWNSTREAM)
        routing.push_frame.assert_awaited_once_with(frame, _FrameDirection.DOWNSTREAM)

        for svc in services.values():
            svc.process_frame.assert_not_awaited()


class TestPushFrameRouting:
    def test_push_frame_overridden_on_init(self):
        """Init overrides each service's push_frame to route through the routing service."""
        services = _make_services()
        original_push_a = services["member-a"].push_frame
        original_push_b = services["member-b"].push_frame

        RoutingLLMService(services)

        assert services["member-a"].push_frame is not original_push_a
        assert services["member-b"].push_frame is not original_push_b
        assert callable(services["member-a"].push_frame)
        assert callable(services["member-b"].push_frame)
