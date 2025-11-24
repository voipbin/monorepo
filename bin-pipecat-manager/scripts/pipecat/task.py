import asyncio
from typing import Dict
from loguru import logger

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
            
        logger.info(f"Stopping pipeline task id='{id}'")
        try:
            await task.cancel()

            # remove from task manager
            async with self._lock:
                self._tasks.pop(id, None)

            return True

        except Exception as e:
            logger.error(f"Error stopping pipeline task id='{id}': {e}")
            return False


    async def list_ids(self):
        async with self._lock:
            return list(self._tasks.keys())


task_manager = TaskManager()
