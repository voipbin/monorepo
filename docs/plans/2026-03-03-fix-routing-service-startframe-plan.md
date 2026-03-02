# Fix Routing Service StartFrame Initialization — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix "TaskManager is still not initialized" errors in pipecat routing services by properly propagating `setup()` and lazily forwarding `StartFrame` to child services.

**Architecture:** Three routing services (`RoutingSTTService`, `RoutingTTSService`, `RoutingLLMService`) wrap per-member pipecat services but never call `setup()` or forward `StartFrame` to them. We add `setup()` propagation, `StartFrame`/`EndFrame` handling, and lazy-start on member switch to all three.

**Tech Stack:** Python 3.12, pipecat-ai >= 0.0.101, pipecat-ai-flows >= 0.0.10

**Design doc:** `docs/plans/2026-03-03-fix-routing-service-startframe-design.md`

---

### Task 1: Update `routing_stt.py`

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/routing_stt.py`

**Step 1: Replace routing_stt.py with the fixed version**

```python
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
```

**Step 2: Verify syntax**

Run from the worktree:
```bash
cd bin-pipecat-manager/scripts/pipecat && python3 -c "import ast; ast.parse(open('routing_stt.py').read()); print('OK')"
```
Expected: `OK`

---

### Task 2: Update `routing_tts.py`

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/routing_tts.py`

**Step 1: Replace routing_tts.py with the fixed version**

```python
from loguru import logger

from pipecat.frames.frames import EndFrame, Frame, StartFrame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor


class RoutingTTSService(FrameProcessor):
    """Routes TTS processing to the appropriate provider based on active member.

    Child services are standalone objects not in the pipeline. This class
    propagates setup() to give them a TaskManager, forwards StartFrame lazily
    (only when a service becomes active), and suppresses duplicate StartFrame
    downstream pushes during member switches.
    """

    def __init__(self, member_services: dict[str, any]):
        """Initialize with a dict mapping member_id -> TTS service instance."""
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
            # Member has no TTS configured — keep using the previous member's TTS service.
            # In a voice call, this is safer than having no TTS (silence).
            logger.warning(f"Member {member_id} has no TTS service, keeping previous member's TTS")
            return
        self._active_id = member_id
        logger.info(f"TTS routing switched to member: {member_id}")

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
```

**Step 2: Verify syntax**

```bash
cd bin-pipecat-manager/scripts/pipecat && python3 -c "import ast; ast.parse(open('routing_tts.py').read()); print('OK')"
```
Expected: `OK`

---

### Task 3: Update `routing_llm.py`

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/routing_llm.py`

**Step 1: Replace routing_llm.py with the fixed version**

This file has additional methods (`register_function`, `unregister_function`, `active_service`) that must be preserved.

```python
from loguru import logger

from pipecat.frames.frames import EndFrame, Frame, StartFrame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor


class RoutingLLMService(FrameProcessor):
    """Routes LLM processing to the appropriate provider based on active member.

    Wraps multiple LLM service instances and delegates process_frame to the
    active member's service. Output from wrapped services is routed through
    this processor via push_frame interception.

    Child services are standalone objects not in the pipeline. This class
    propagates setup() to give them a TaskManager, forwards StartFrame lazily
    (only when a service becomes active), and suppresses duplicate StartFrame
    downstream pushes during member switches.
    """

    def __init__(self, member_services: dict[str, any]):
        """Initialize with a dict mapping member_id -> LLM service instance."""
        super().__init__()
        self._services = member_services
        self._active_id = None
        self._started_ids: set[str] = set()
        self._start_frame = None
        self._start_direction = None
        self._suppress_start_propagation = False

        # Override each service's push_frame to route output through us
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
            raise ValueError(f"Unknown member_id for LLM routing: {member_id}")
        self._active_id = member_id
        logger.info(f"LLM routing switched to member: {member_id}")

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
```

**Step 2: Verify syntax**

```bash
cd bin-pipecat-manager/scripts/pipecat && python3 -c "import ast; ast.parse(open('routing_llm.py').read()); print('OK')"
```
Expected: `OK`

---

### Task 4: Build Docker image to verify imports

**Step 1: Build the Docker image**

The pipecat-ai package is only available inside Docker, so we verify by building:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-fix-routing-service-startframe/bin-pipecat-manager
docker build -t pipecat-manager-test -f Dockerfile ../ 2>&1 | tail -20
```

Expected: Build succeeds. If `StartFrame` or `EndFrame` import fails, the build will error during the Python layer.

**Step 2: Quick import check inside container**

```bash
docker run --rm pipecat-manager-test python3 -c "
from pipecat.frames.frames import StartFrame, EndFrame
from routing_stt import RoutingSTTService
from routing_tts import RoutingTTSService
from routing_llm import RoutingLLMService
print('All imports OK')
"
```

Expected: `All imports OK`

---

### Task 5: Commit

**Step 1: Stage and commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-fix-routing-service-startframe
git add \
  bin-pipecat-manager/scripts/pipecat/routing_stt.py \
  bin-pipecat-manager/scripts/pipecat/routing_tts.py \
  bin-pipecat-manager/scripts/pipecat/routing_llm.py \
  docs/plans/2026-03-03-fix-routing-service-startframe-design.md \
  docs/plans/2026-03-03-fix-routing-service-startframe-plan.md

git commit -m "NOJIRA-fix-routing-service-startframe

Fix 'TaskManager is still not initialized' errors in pipecat team pipeline by
properly initializing child services in routing processors.

- bin-pipecat-manager: Override setup() in routing services to propagate TaskManager to children
- bin-pipecat-manager: Handle StartFrame/EndFrame explicitly in routing services
- bin-pipecat-manager: Add lazy-start for child services on member switch
- bin-pipecat-manager: Suppress duplicate StartFrame downstream propagation during lazy-start
- docs: Add design doc for routing service StartFrame fix"
```

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-fix-routing-service-startframe
```
