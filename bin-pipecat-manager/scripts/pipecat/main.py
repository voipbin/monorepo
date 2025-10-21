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


{"ws_server_url":"ws://localhost:35747/ws","llm":"openai.gpt-4-turbo","tts":"cartesia","stt":"deepgram","voice_id":"71a7ad14-091c-4e8e-a314-022ece01c121","messages":[{"content":"\nRole:\nYou are an AI assistant integrated with voipbin. \nYour role is to follow the user's system or custom prompt strictly, provide natural responses, and call external tools when necessary.\n\nContext:\n- Users will set their own instructions (persona, style, context).\n- You must adapt to those instructions consistently.\n- If user requests or situation requires, use available tools to gather data or perform actions.\n\nInput Values:\n- User-provided system/custom prompt\n- User query\n- Available tools list\n\nInstructions:\n- Always prioritize the user's provided prompt instructions.\n- Generate a helpful, coherent, and contextually appropriate response.\n- If tools are available and required, call them responsibly and return results clearly.\n- **Do not mention tool names or the fact that a tool is being used in the user-facing response.**\n- Maintain consistency with the user-defined tone and role.\n- If ambiguity exists, ask clarifying questions before answering.\n- Before giving the final answer, outline a short execution plan (2–4 steps), then provide a concise summary (1–2 sentences) and the final answer.  \n- For each Input Value, ask clarifying questions **one at a time in sequence**. Wait for the user's answer before moving to the next question.  \n\nConstraints:\n- Avoid hallucination; use tools for factual queries.  \n- Keep answers aligned with user's persona and tone.  \n- Respect conversation history and continuity.  \n\t","role":"system"},{"content":null,"role":"system"}]}

@app.post("/run")
async def run_pipeline_endpoint(req: PipelineRequest):
    try:
        logger.info("=== Received /run request ===")
        logger.info(f"ws_server_url: {req.ws_server_url}")
        logger.info(f"llm: {req.llm}")
        logger.info(f"tts: {req.tts}")
        logger.info(f"stt: {req.stt}")
        logger.info(f"voice_id: {req.voice_id}")
        logger.info(f"messages: {json.dumps([m.dict() for m in req.messages], indent=2)}")

        await run_pipeline(
            req.ws_server_url,
            req.llm,
            req.tts,
            req.stt,
            req.voice_id,
            [m.dict() for m in req.messages],
            )

        return {"status": "ok", "message": "Pipeline executed successfully"}

    except Exception as e:
        logger.exception(f"Pipeline execution failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# --- 서버 실행 ---
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
