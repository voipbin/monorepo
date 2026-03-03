from loguru import logger

from pipecat.frames.frames import CancelFrame, EndFrame, Frame, StartFrame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor, FrameProcessorSetup


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

    async def setup(self, setup: FrameProcessorSetup):
        await super().setup(setup)
        for svc in self._services.values():
            await svc.setup(setup)

    async def cleanup(self):
        await super().cleanup()
        for svc in self._services.values():
            await svc.cleanup()

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            raise ValueError(f"Unknown member_id for LLM routing: {member_id}")
        self._active_id = member_id
        logger.info(f"LLM routing switched to member: {member_id}")

    async def process_frame(self, frame: Frame, direction: FrameDirection):
        # Lifecycle frames must initialize the router itself and propagate to all inner services.
        if isinstance(frame, (StartFrame, CancelFrame, EndFrame)):
            await super().process_frame(frame, direction)
            for svc in self._services.values():
                await svc.process_frame(frame, direction)
            return

        if self._active_id and self._active_id in self._services:
            await self._services[self._active_id].process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)

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
