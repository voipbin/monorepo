import asyncio
from typing import Dict

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
            
            task.cancel()
            self._tasks.pop(id, None)
            return True

    async def list_ids(self):
        async with self._lock:
            return list(self._tasks.keys())


task_manager = TaskManager()
