import asyncio
from typing import Dict
from loguru import logger

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
        logger.info(f"Stopping pipeline task id='{id}'")
        try:
            await task.cancel()

            # close WebSocket transports explicitly if exists
            if hasattr(task, "pipeline"):
                logger.info(f"Closing pipeline stages for task id='{id}'")
                for stage in getattr(task.pipeline, "stages", []):
                    if hasattr(stage, "close"):
                        logger.info(f"Closing stage '{stage}' for task id='{id}'")
                        await stage.close()

            # # unregister LLM tool functions
            # if hasattr(task, "llm_service") and task.llm_service:
            #     logger.info(f"Unregistering tool functions for task id='{id}'")
            #     llm_service = task.llm_service
                
            #     tool_unregister(llm_service)
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
