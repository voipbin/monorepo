import asyncio
import json
import sys
from typing import Optional, List
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from loguru import logger
from dotenv import load_dotenv

from run import run_pipeline
from task import task_manager

load_dotenv(override=True)

app = FastAPI(title="Python Pipeline API")

class ToolCall(BaseModel):
    id: str
    type: Optional[str] = "function"
    function: Optional[dict] = None

class Message(BaseModel):
    role: Optional[str] = None
    content: Optional[str] = None
    tool_calls: Optional[List[ToolCall]] = None   # for assistant -> tool call
    tool_call_id: Optional[str] = None            # for tool -> assistant response

    class Config:
        extra = "ignore"

class PipelineRequest(BaseModel):
    id: Optional[str] = None
    llm_type: Optional[str] = None
    llm_key: Optional[str] = None
    tts: Optional[str] = None
    stt: Optional[str] = None
    voice_id: Optional[str] = None
    messages: Optional[List[Message]] = Field(default_factory=list)

async def run_pipeline_wrapper(req: PipelineRequest):
    try:
        logger.info(f"Pipeline started: id={req.id}")
        await run_pipeline(
            req.id,
            req.llm_type,
            req.llm_key,
            req.tts,
            req.stt,
            req.voice_id,
            [m.model_dump() for m in req.messages],
        )
        logger.info(f"Pipeline finished successfully: id={req.id}")
    except Exception as e:
        logger.exception(f"Pipeline failed (id={req.id}): {e}")


@app.post("/run")
async def run_pipeline_endpoint(req: PipelineRequest):
    try:
        msg_count = len(req.messages or [])
        logger.info(json.dumps({
            "event": "run_request",
            "id": req.id,
            "llm_type": req.llm_type,
            "tts": req.tts,
            "stt": req.stt,
            "voice_id": req.voice_id,
            "message_count": msg_count
        }))
        
        asyncio.create_task(run_pipeline_wrapper(req))
        await asyncio.sleep(0)

        return {"status": "ok", "message": "Pipeline executed successfully"}

    except Exception as e:
        logger.exception(f"Pipeline execution failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/stop")
async def stop_pipeline(id: str):
    logger.info(f"Stop request received for pipeline task id='{id}'")    
    cancelled = await task_manager.stop(id)

    if not cancelled:
        raise HTTPException(404, f"No running pipeline with id '{id}' found")

    logger.info(f"Pipeline task id='{id}' stopped successfully")
    return {"status": "stopped", "id": id}


# --- logger setup ---
logger.remove()
logger.add(
    sys.stdout,
    serialize=True,
)

# --- run server ---
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
