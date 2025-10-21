import asyncio
import json
from typing import Optional, List
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from loguru import logger
from dotenv import load_dotenv

from run import run_pipeline

load_dotenv(override=True)

app = FastAPI(title="Python Pipeline API")

class Message(BaseModel):
    role: Optional[str] = None
    content: Optional[str] = None

class PipelineRequest(BaseModel):
    ws_server_url: Optional[str] = None
    llm: Optional[str] = None
    tts: Optional[str] = None
    stt: Optional[str] = None
    voice_id: Optional[str] = None
    messages: Optional[List[Message]] = None

@app.post("/run")
async def run_pipeline_endpoint(req: PipelineRequest):
    try:
        logger.info("=== Received /run request ===")
        logger.info(f"ws_server_url: {req.ws_server_url}")
        logger.info(f"llm: {req.llm}")
        logger.info(f"tts: {req.tts}")
        logger.info(f"stt: {req.stt}")
        logger.info(f"voice_id: {req.voice_id}")
        logger.info(f"messages_length: {len(req.messages) if req.messages else 0}")

        await run_pipeline(
            req.ws_server_url,
            req.llm,
            req.tts,
            req.stt,
            req.voice_id,
            [m.model_dump() for m in req.messages],
            )

        return {"status": "ok", "message": "Pipeline executed successfully"}

    except Exception as e:
        logger.exception(f"Pipeline execution failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# --- 서버 실행 ---
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
