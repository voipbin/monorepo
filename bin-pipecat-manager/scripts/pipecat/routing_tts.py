from loguru import logger

from pipecat.frames.frames import CancelFrame, EndFrame, Frame, StartFrame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor, FrameProcessorSetup


class RoutingTTSService(FrameProcessor):
    """Routes TTS processing to the appropriate provider based on active member."""

    def __init__(self, member_services: dict[str, any]):
        """Initialize with a dict mapping member_id -> TTS service instance."""
        super().__init__()
        self._services = member_services
        self._active_id = None

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
            # Member has no TTS configured — keep using the previous member's TTS service.
            # In a voice call, this is safer than having no TTS (silence).
            logger.warning(f"Member {member_id} has no TTS service, keeping previous member's TTS")
            return
        self._active_id = member_id
        logger.info(f"TTS routing switched to member: {member_id}")

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
