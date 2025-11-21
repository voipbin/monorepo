import asyncio
from typing import Dict

from tools import tool_unregister

class TaskManager:
    def __init__(self):
        self._tasks: Dict[str, object] = {}
        self._lock = asyncio.Lock()

    async def add(self, id: str, task):
        async with self._lock:
            self._tasks[id] = task

    async def get(self, id: str):
        async with self._lock:
            return self._tasks.get(id)

    async def remove(self, id: str):
        async with self._lock:
            return self._tasks.pop(id, None)

    async def stop(self, id: str):
        async with self._lock:
            task = self._tasks.get(id)
            if not task:
                return False
            
            # await task.cancel()
            # self._tasks.pop(id, None)
            # return True

        try:
            await task.cancel()

            # 2️⃣ close WebSocket transports explicitly if exists
            if hasattr(task, "pipeline"):
                for stage in getattr(task.pipeline, "stages", []):
                    if hasattr(stage, "close"):
                        await stage.close()

            # 3️⃣ unregister LLM tool functions
            if hasattr(task, "llm_service") and task.llm_service:
                llm_service = task.llm_service
                
                tool_unregister(llm_service)
            # 4️⃣ remove from task manager
            async with self._lock:
                self._tasks.pop(id, None)

            return True

        except Exception as e:
            return False


    async def list_ids(self):
        async with self._lock:
            return list(self._tasks.keys())


task_manager = TaskManager()
