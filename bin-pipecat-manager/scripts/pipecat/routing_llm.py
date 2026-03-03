from loguru import logger

from routing_base import RoutingServiceBase


class RoutingLLMService(RoutingServiceBase):
    """Routes LLM processing to the appropriate provider based on active member.

    Wraps multiple LLM service instances and delegates process_frame to the
    active member's service. Output from wrapped services is routed through
    this processor via push_frame interception.
    """

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            raise ValueError(f"Unknown member_id for LLM routing: {member_id}")
        self._active_id = member_id
        logger.info(f"LLM routing switched to member: {member_id}")

    # Delegate FlowManager-facing methods to all services so transitions work.
    # Signature matches pipecat's LLMService.register_function so FlowManager
    # can call us with cancel_on_interruption and other kwargs.
    def register_function(self, function_name=None, handler=None, start_callback=None, *, cancel_on_interruption=True, **kwargs):
        for svc in self._services.values():
            svc.register_function(
                function_name=function_name,
                handler=handler,
                start_callback=start_callback,
                cancel_on_interruption=cancel_on_interruption,
            )

    def unregister_function(self, name):
        # Unregister from ALL services since function may have been registered
        # on a previously active service
        for svc in self._services.values():
            try:
                svc.unregister_function(name)
            except (KeyError, Exception):
                pass

    @property
    def active_service(self):
        """Return the currently active LLM service instance."""
        if self._active_id:
            return self._services.get(self._active_id)
        return None
