import asyncio
import json
from typing import Optional, List
from fastapi import FastAPI, HTTPException, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
from loguru import logger
from dotenv import load_dotenv

import log_config
log_config.setup()  # before run is imported: its import time log lines follow the policy

import custom_endpoint_net as ce_net
from run import init_pipeline, execute_pipeline
from task import task_manager

load_dotenv(override=True)

app = FastAPI(title="Python Pipeline API")

# Fixed texts (custom endpoint, VOIP-1599). The default 422 body of FastAPI repeats the request
# (the endpoint key included), and pythonrunner.go puts non-200 bodies into its error text.
_INVALID_REQUEST_DETAIL = "invalid request"
_INIT_FAILED_DETAIL = "pipeline initialization failed"


@app.exception_handler(RequestValidationError)
async def request_validation_handler(request: Request, exc: RequestValidationError):
    # Only the field path and the error type are logged: "input" would repeat the request.
    problems = [f"{'.'.join(str(p) for p in e.get('loc', ()))}:{e.get('type', '')}" for e in exc.errors()]
    logger.error(f"Request validation failed: {', '.join(problems[:10])}")
    return JSONResponse(status_code=422, content={"detail": _INVALID_REQUEST_DETAIL})


class ToolCall(BaseModel):
    # Optional: history written by a model server that omits the tool call id has none (VOIP-1633);
    # filter_valid_messages repairs or drops those rows before they reach a provider.
    id: Optional[str] = None
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
    run_llm: Optional[bool] = False

    class Config:
        extra = "ignore"

class ResolvedAI(BaseModel):
    engine_model: str
    # engine_key is the legacy key: never forwarded to a customer chosen host.
    engine_key: str = Field(repr=False)
    # Customer specified endpoint (customendpoint.): the URL and the dedicated key.
    llm_base_url: Optional[str] = Field(default=None, repr=False)
    llm_base_key: Optional[str] = Field(default=None, repr=False)
    # None: older Go that does not send the field. "": Go rejected the member.
    llm_type: Optional[str] = None
    init_prompt: Optional[str] = None
    parameter: Optional[dict] = None
    tts_type: Optional[str] = None
    tts_voice_id: Optional[str] = None
    stt_type: Optional[str] = None
    vad_config: Optional[dict] = None
    smart_turn_enabled: Optional[bool] = False

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
    llm_key: Optional[str] = Field(default=None, repr=False)
    # Customer specified endpoint (customendpoint.). customer_id is not a secret.
    customer_id: Optional[str] = None
    llm_base_url: Optional[str] = Field(default=None, repr=False)
    llm_base_key: Optional[str] = Field(default=None, repr=False)
    llm_messages: Optional[List[Message]] = Field(default_factory=list)
    stt_type: Optional[str] = None
    stt_language: Optional[str] = None
    tts_type: Optional[str] = None
    tts_language: Optional[str] = None
    tts_voice_id: Optional[str] = None
    tools: Optional[List[Tool]] = Field(default_factory=list)
    resolved_team: Optional[ResolvedTeam] = None
    vad_config: Optional[dict] = None
    smart_turn_enabled: Optional[bool] = False

def request_has_custom_endpoint(req: PipelineRequest) -> bool:
    """True when the request or any team member uses the customendpoint service."""
    if ce_net.is_custom_endpoint_type(req.llm_type):
        return True
    if req.resolved_team:
        for member in req.resolved_team.members:
            if ce_net.is_custom_endpoint_type(member.ai.llm_type) or ce_net.is_custom_endpoint_type(member.ai.engine_model):
                return True
    return False


def _log_llm_type(llm_type):
    return "customendpoint.<redacted>" if ce_net.is_custom_endpoint_type(llm_type) else llm_type


async def execute_pipeline_wrapper(id: str, ctx: dict):
    """Background task wrapper for execute_pipeline."""
    try:
        await execute_pipeline(id, ctx)
        logger.info(f"Pipeline finished successfully: id={id}")
    except Exception as e:
        if ctx.get("custom_endpoint"):
            # no raw error text, no traceback: both can carry the URL, the key or a response body
            logger.error(f"Pipeline execution failed (id={id}): custom endpoint session error (details withheld)")
        else:
            logger.exception(f"Pipeline execution failed (id={id}): {e}")


@app.post("/run")
async def run_pipeline_endpoint(req: PipelineRequest):
    msg_count = len(req.llm_messages or [])
    logger.info(json.dumps({
        "event": "run_request",
        "id": req.id,
        "llm_type": _log_llm_type(req.llm_type),
        "llm_message_count": msg_count,
        "stt_type": req.stt_type,
        "stt_language": req.stt_language,
        "tts_type": req.tts_type,
        "tts_language": req.tts_language,
        "tts_voice_id": req.tts_voice_id,
        "has_resolved_team": req.resolved_team is not None,
    }))

    custom = request_has_custom_endpoint(req)
    if custom and not ce_net.reserve_pipeline(req.id):
        # A refused request registers nothing and releases nothing: the pipeline that is
        # running under this id keeps its slots.
        raise HTTPException(status_code=409, detail=ce_net.FIXED_ERROR)

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
            vad_config=req.vad_config,
            smart_turn_enabled=req.smart_turn_enabled,
            customer_id=req.customer_id,
            llm_base_url=req.llm_base_url,
            llm_base_key=req.llm_base_key,
        )
    except BaseException as e:
        if custom:
            # Give back the slots of this request (a built LLM service whose STT or TTS sibling
            # failed, earlier team members whose later member failed). Also on cancellation.
            await ce_net.release_pipeline(req.id)
        if isinstance(e, ValueError):
            if custom:
                logger.error(f"Pipeline validation failed (id={req.id}): custom endpoint request (details withheld)")
                raise HTTPException(status_code=400, detail=ce_net.FIXED_ERROR) from None
            logger.error(f"Pipeline validation failed (id={req.id}): {e}")
            raise HTTPException(status_code=400, detail=str(e))
        if isinstance(e, Exception):
            if custom:
                logger.error(f"Pipeline initialization failed (id={req.id}): custom endpoint request (details withheld)")
                raise HTTPException(status_code=500, detail=_INIT_FAILED_DETAIL) from None
            logger.exception(f"Pipeline initialization failed (id={req.id}): {e}")
            raise HTTPException(status_code=500, detail=str(e))
        raise

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


# --- run server ---
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
