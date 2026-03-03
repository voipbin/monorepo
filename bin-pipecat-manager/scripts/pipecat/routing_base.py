from loguru import logger

from pipecat.frames.frames import EndFrame, Frame, StartFrame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor


class RoutingServiceBase(FrameProcessor):
    """Base class for routing services that delegate to per-member child services.

    Child services are standalone objects not in the pipeline. This class
    propagates setup() to give them a TaskManager, forwards StartFrame lazily
    (only when a service becomes active), forwards EndFrame to all started
    children on shutdown, and suppresses duplicate StartFrame/EndFrame
    downstream pushes.
    """

    def __init__(self, member_services: dict[str, any]):
        super().__init__()
        self._services = member_services
        self._active_id = None
        self._started_ids: set[str] = set()
        self._start_frame = None
        self._start_direction = None
        self._suppress_propagation = False

        for member_id, svc in self._services.items():
            svc.push_frame = self._create_routing_push()

    def _create_routing_push(self):
        async def routing_push(frame: Frame, direction: FrameDirection = FrameDirection.DOWNSTREAM):
            if self._suppress_propagation and isinstance(frame, (StartFrame, EndFrame)):
                return
            await self.push_frame(frame, direction)
        return routing_push

    async def setup(self, setup):
        await super().setup(setup)
        for svc in self._services.values():
            await svc.setup(setup)

    def _check_started(self, frame: Frame):
        # Override pipecat's started check. The base FrameProcessor sets __started
        # inside process_frame() when it sees StartFrame, but we override
        # process_frame to route frames to children. Calling super().process_frame()
        # is not feasible because __start() requires full pipeline internals
        # (observer, clock, TaskManager tasks). Since we handle StartFrame
        # explicitly and manage our own lifecycle, push_frame is always safe.
        return True

    async def _ensure_started(self, member_id: str):
        """Lazily forward stored StartFrame to a child that hasn't been started yet."""
        if member_id in self._started_ids or not self._start_frame:
            return
        self._suppress_propagation = True
        try:
            await self._services[member_id].process_frame(self._start_frame, self._start_direction)
        finally:
            self._suppress_propagation = False
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
            if self._started_ids:
                self._suppress_propagation = True
                try:
                    for mid in list(self._started_ids):
                        if mid in self._services:
                            await self._services[mid].process_frame(frame, direction)
                finally:
                    self._suppress_propagation = False
            await self.push_frame(frame, direction)
            return

        if self._active_id and self._active_id in self._services:
            await self._ensure_started(self._active_id)
            await self._services[self._active_id].process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)
