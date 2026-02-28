from loguru import logger

from pipecat.frames.frames import Frame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor


class RoutingLLMService(FrameProcessor):
    """Routes LLM processing to the appropriate provider based on active member.

    Wraps multiple LLM service instances and delegates process_frame to the
    active member's service. Output from wrapped services is routed through
    this processor via push_frame interception.
    """

    def __init__(self, member_services: dict[str, any]):
        """Initialize with a dict mapping member_id -> LLM service instance."""
        super().__init__()
        self._services = member_services
        self._active_id = None

        # Override each service's push_frame to route output through us
        for member_id, svc in self._services.items():
            svc.push_frame = self._create_routing_push(svc)

    def _create_routing_push(self, svc):
        async def routing_push(frame: Frame, direction: FrameDirection = FrameDirection.DOWNSTREAM):
            await self.push_frame(frame, direction)
        return routing_push

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            raise ValueError(f"Unknown member_id for LLM routing: {member_id}")
        self._active_id = member_id
        logger.info(f"LLM routing switched to member: {member_id}")

    async def process_frame(self, frame: Frame, direction: FrameDirection):
        if self._active_id and self._active_id in self._services:
            await self._services[self._active_id].process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)

    # Delegate FlowManager-facing methods to all services so transitions work
    def register_function(self, name, handler):
        for svc in self._services.values():
            svc.register_function(name, handler)

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
