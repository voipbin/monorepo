import asyncio
import json
import sys
from typing import Optional, List
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from loguru import logger
from dotenv import load_dotenv

from run import init_pipeline, execute_pipeline
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

class Tool(BaseModel):
    name: str
    description: str
    parameters: Optional[dict] = None

    class Config:
        extra = "ignore"

class ResolvedAI(BaseModel):
    engine_model: str
    engine_key: str
    init_prompt: Optional[str] = None
    parameter: Optional[dict] = None
    tts_type: Optional[str] = None
    tts_voice_id: Optional[str] = None
    stt_type: Optional[str] = None

    class Config:
        extra = "ignore"

class TeamTransition(BaseModel):
    function_name: str
    description: str
    next_member_id: str

    class Config:
        extra = "ignore"

class ResolvedMember(BaseModel):
    id: str
    name: str
    ai: ResolvedAI
    tools: Optional[List[Tool]] = Field(default_factory=list)
    transitions: Optional[List[TeamTransition]] = Field(default_factory=list)

    class Config:
        extra = "ignore"

class ResolvedTeam(BaseModel):
    id: str
    start_member_id: str
    members: List[ResolvedMember]

    class Config:
        extra = "ignore"

class PipelineRequest(BaseModel):
    id: Optional[str] = None
    llm_type: Optional[str] = None
    llm_key: Optional[str] = None
    llm_messages: Optional[List[Message]] = Field(default_factory=list)
    stt_type: Optional[str] = None
    stt_language: Optional[str] = None
    tts_type: Optional[str] = None
    tts_language: Optional[str] = None
    tts_voice_id: Optional[str] = None
    tools: Optional[List[Tool]] = Field(default_factory=list)
    resolved_team: Optional[ResolvedTeam] = None
    vad_stop_secs: float = 0.5

async def execute_pipeline_wrapper(id: str, ctx: dict):
    """Background task wrapper for execute_pipeline."""
    try:
        await execute_pipeline(id, ctx)
        logger.info(f"Pipeline finished successfully: id={id}")
    except Exception as e:
        logger.exception(f"Pipeline execution failed (id={id}): {e}")


@app.post("/run")
async def run_pipeline_endpoint(req: PipelineRequest):
    msg_count = len(req.llm_messages or [])
    logger.info(json.dumps({
        "event": "run_request",
        "id": req.id,
        "llm_type": req.llm_type,
        "llm_message_count": msg_count,
        "stt_type": req.stt_type,
        "stt_language": req.stt_language,
        "tts_type": req.tts_type,
        "tts_language": req.tts_language,
        "tts_voice_id": req.tts_voice_id,
        "has_resolved_team": req.resolved_team is not None,
    }))

    try:
        tools_data = [t.model_dump() for t in req.tools] if req.tools else []
        resolved_team_data = req.resolved_team.model_dump() if req.resolved_team else None

        ctx = await init_pipeline(
            req.id,
            req.llm_type,
            req.llm_key,
            [m.model_dump() for m in req.llm_messages],
            req.stt_type,
            req.stt_language,
            req.tts_type,
            req.tts_language,
            req.tts_voice_id,
            tools_data,
            resolved_team=resolved_team_data,
            vad_stop_secs=req.vad_stop_secs,
        )
    except ValueError as e:
        logger.error(f"Pipeline validation failed (id={req.id}): {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.exception(f"Pipeline initialization failed (id={req.id}): {e}")
        raise HTTPException(status_code=500, detail=str(e))

    asyncio.create_task(execute_pipeline_wrapper(req.id, ctx))

    return {"status": "ok", "message": "Pipeline initialized and started"}

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
