from loguru import logger

from pipecat.frames.frames import EndFrame, Frame, StartFrame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor


class RoutingSTTService(FrameProcessor):
    """Routes STT processing to the appropriate provider based on active member.

    Child services are standalone objects not in the pipeline. This class
    propagates setup() to give them a TaskManager, forwards StartFrame lazily
    (only when a service becomes active), and suppresses duplicate StartFrame
    downstream pushes during member switches.
    """

    def __init__(self, member_services: dict[str, any]):
        """Initialize with a dict mapping member_id -> STT service instance."""
        super().__init__()
        self._services = member_services
        self._active_id = None
        self._started_ids: set[str] = set()
        self._start_frame = None
        self._start_direction = None
        self._suppress_start_propagation = False

        for member_id, svc in self._services.items():
            svc.push_frame = self._create_routing_push(svc)

    def _create_routing_push(self, svc):
        async def routing_push(frame: Frame, direction: FrameDirection = FrameDirection.DOWNSTREAM):
            if self._suppress_start_propagation and isinstance(frame, StartFrame):
                return
            await self.push_frame(frame, direction)
        return routing_push

    async def setup(self, setup):
        await super().setup(setup)
        for svc in self._services.values():
            await svc.setup(setup)

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            # Member has no STT configured — keep using the previous member's STT service.
            # In a voice call, this is safer than having no STT (deaf).
            logger.warning(f"Member {member_id} has no STT service, keeping previous member's STT")
            return
        self._active_id = member_id
        logger.info(f"STT routing switched to member: {member_id}")

    async def _ensure_started(self, member_id: str):
        """Lazily forward stored StartFrame to a child that hasn't been started yet."""
        if member_id in self._started_ids or not self._start_frame:
            return
        self._suppress_start_propagation = True
        try:
            await self._services[member_id].process_frame(self._start_frame, self._start_direction)
        finally:
            self._suppress_start_propagation = False
        self._started_ids.add(member_id)

    async def process_frame(self, frame: Frame, direction: FrameDirection):
        if isinstance(frame, StartFrame):
            self._start_frame = frame
            self._start_direction = direction
            if self._active_id and self._active_id in self._services:
                await self._services[self._active_id].process_frame(frame, direction)
                self._started_ids.add(self._active_id)
            else:
                await self.push_frame(frame, direction)
            return

        if isinstance(frame, EndFrame):
            if self._active_id and self._active_id in self._services:
                await self._services[self._active_id].process_frame(frame, direction)
            else:
                await self.push_frame(frame, direction)
            return

        if self._active_id and self._active_id in self._services:
            await self._ensure_started(self._active_id)
            await self._services[self._active_id].process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)
