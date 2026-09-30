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
        self._start_forwarded = False

        # Override each service's push_frame to route output through us
        for member_id, svc in self._services.items():
            svc.push_frame = self._create_routing_push(svc)

    def _create_routing_push(self, svc):
        async def routing_push(frame: Frame, direction: FrameDirection = FrameDirection.DOWNSTREAM):
            # Every member echoes the StartFrame; forward only the first one.
            if isinstance(frame, StartFrame):
                if self._start_forwarded:
                    return
                self._start_forwarded = True
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
            if isinstance(frame, StartFrame):
                # 1.12 broadcasts service metadata from AIService.push_frame,
                # which the per-instance routing_push bypasses.
                for svc in self._services.values():
                    if hasattr(svc, "broadcast_service_metadata"):
                        await svc.broadcast_service_metadata()
            return

        if self._active_id and self._active_id in self._services:
            await self._services[self._active_id].process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)

    # Fan-out helpers. Signature matches pipecat 1.12 LLMService.register_function
    # (name, handler, *, cancel_on_interruption=None, timeout_secs=None,
    # cancellable_by_llm=None). Built-in pipecat.flows no longer calls this: each
    # member LLM auto-registers advertised handlers on its own LLMContextFrame.
    # `**kwargs` is kept only to reject unknown args loudly (e.g. a
    # re-introduced start_callback) rather than silently swallow them.
    def register_function(
        self,
        name=None,
        handler=None,
        *,
        cancel_on_interruption=None,
        timeout_secs=None,
        cancellable_by_llm=None,
        **kwargs,
    ):
        if kwargs:
            raise TypeError(
                f"RoutingLLMService.register_function got unexpected kwargs: {list(kwargs)}"
            )
        for svc in self._services.values():
            svc.register_function(
                name,
                handler,
                cancel_on_interruption=cancel_on_interruption,
                timeout_secs=timeout_secs,
                cancellable_by_llm=cancellable_by_llm,
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
