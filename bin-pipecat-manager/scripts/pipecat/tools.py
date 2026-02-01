import common
import json
from loguru import logger
import aiohttp
import asyncio
from typing import List, Dict, Any

from pipecat.services.llm_service import FunctionCallParams
from pipecat.frames.frames import FunctionCallResultProperties


def convert_to_openai_format(tools_data: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """
    Convert tools from ai-manager format to OpenAI function calling format.

    ai-manager format:
    {
        "name": "connect_call",
        "description": "...",
        "parameters": {...}
    }

    OpenAI format:
    {
        "type": "function",
        "function": {
            "name": "connect_call",
            "description": "...",
            "parameters": {...}
        }
    }
    """
    if not tools_data:
        return []

    openai_tools = []
    for tool in tools_data:
        openai_tool = {
            "type": "function",
            "function": {
                "name": tool.get("name", ""),
                "description": tool.get("description", ""),
                "parameters": tool.get("parameters", {"type": "object", "properties": {}, "required": []}),
            }
        }
        openai_tools.append(openai_tool)

    return openai_tools


def get_tool_names(tools_data: List[Dict[str, Any]]) -> List[str]:
    """Extract tool names from tools data."""
    if not tools_data:
        return []
    return [tool.get("name", "") for tool in tools_data if tool.get("name")]


def tool_register(llm_service, pipecatcall_id: str, tool_names: List[str]):
    """Register tool functions with the LLM service."""
    def create_wrapper(tool_name, pipecatcall_id):
        async def wrapper(params: FunctionCallParams):
            return await tool_execute(tool_name, params, pipecatcall_id)
        return wrapper

    for tool_name in tool_names:
        wrapper = create_wrapper(tool_name, pipecatcall_id)
        llm_service.register_function(tool_name, wrapper)
    logger.info(f"Registered {len(tool_names)} tools for pipecatcall {pipecatcall_id}")


def tool_unregister(llm_service, tool_names: List[str]):
    """Unregisters tools from the LLM service."""
    for tool_name in tool_names:
        try:
            llm_service.unregister_function(tool_name)
        except KeyError:
            logger.debug(f"Tool '{tool_name}' was not registered or already removed.")
        except Exception as e:
            logger.warning(f"Error while unregistering tool '{tool_name}': {e}")


async def tool_execute(tool_name: str, params: FunctionCallParams, pipecatcall_id: str):
    """Generic executor for tool calls (connect, message_send, etc)."""
    
    args = params.arguments if isinstance(params.arguments, dict) else {}
    logger.info(f"[{tool_name}] Executing. Args: {json.dumps(args, ensure_ascii=False)}")

    should_run_llm = args.pop("run_llm", False)

    http_url = f"{common.PIPECATCALL_HTTP_URL}/{pipecatcall_id}/tools"
    http_body = {
        "id": params.tool_call_id,
        "type": "function",
        "function": {
            "name": tool_name,
            "arguments": json.dumps(args, ensure_ascii=False),
        },
    }
    logger.debug(f"[{tool_name}] POST {http_url} with body: {json.dumps(http_body, ensure_ascii=False)}")

    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(http_url, json=http_body, timeout=aiohttp.ClientTimeout(total=10)) as response:
                status = response.status
                content_type = response.headers.get("Content-Type", "")
                text = await response.text()

                if content_type.startswith("application/json"):
                    try:
                        data = json.loads(text)
                    except json.JSONDecodeError:
                        data = {"raw": text}
                else:
                    data = {"raw": text}

                if status >= 400:
                    logger.warning(f"[{tool_name}] HTTP {status} Error: {text[:500]}")
                    await params.result_callback({
                        "status": "error",
                        "error": f"HTTP {status}: {text}",
                    })
                    return

                logger.info(f"[{tool_name}] Success: {status}")
                properties = FunctionCallResultProperties(
                    run_llm=should_run_llm,
                )

                await params.result_callback(
                    {
                        "status": "ok",
                        "data": data,
                    },
                    properties=properties,
                )

    except asyncio.TimeoutError:
        logger.error(f"[{tool_name}] Request timed out after 10s")
        await params.result_callback({
            "status": "error",
            "error": "Request timed out",
        })

    except aiohttp.ClientError as e:
        logger.exception(f"[{tool_name}] Client error: {e}")
        await params.result_callback({
            "status": "error",
            "error": str(e),
        })

    except Exception as e:
        logger.exception(f"[{tool_name}] Unexpected error: {e}")
        await params.result_callback({
            "status": "error",
            "error": f"Unexpected error: {e}",
        })
