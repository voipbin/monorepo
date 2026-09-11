import common
import json
from loguru import logger
import aiohttp
import asyncio
from typing import List, Dict, Any

from pipecat.services.llm_service import FunctionCallParams
from pipecat.frames.frames import FunctionCallResultProperties


def register_missing_tool_logging(llm_service, pipecatcall_id: str):
    """Log an ERROR when the LLM calls a tool not registered on this service (VOIP-1512).

    pipecat's ``on_function_calls_started`` event fires just before dispatch, for
    every function call the LLM emits -- including calls to tools that were never
    registered. When a called tool is absent from ``service._functions``, pipecat
    silently routes it to its terminal missing-function handler and only emits its
    own WARNING; nothing reaches the agent or an operator. This promotes that case
    to a structured ERROR we control (fixed ``[missing_tool]`` prefix), which an
    Alloy ``stage.metrics`` stage turns into a Prometheus counter for alerting.

    Anchored on the real pipecat ``LLMService`` instance (single-AI: the one
    llm_service; team: each member service), NOT on ``tool_register`` -- the team
    path never calls ``tool_register``, so anchoring there would leave team sessions
    blind.

    Membership is decided against ``service._functions`` -- the exact dict pipecat's
    own dispatch checks (llm_service.py: ``function_name in self._functions.keys()``).
    This runner never registers a ``None`` catch-all handler, so the absence of a
    name from ``_functions`` is a genuine missing tool with no false positives.
    """

    @llm_service.event_handler("on_function_calls_started")
    async def _on_function_calls_started(service, function_calls):
        for fc in function_calls:
            if fc.function_name not in service._functions:
                logger.error(
                    f"[missing_tool] LLM called unadvertised tool. "
                    f"pipecatcall_id={pipecatcall_id} tool_name={fc.function_name}"
                )


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


def _build_run_llm_defaults(tools_data: List[Dict[str, Any]]) -> Dict[str, bool]:
    """Build per-tool run_llm defaults from tool metadata."""
    defaults = {}
    if tools_data:
        for tool in tools_data:
            name = tool.get("name", "")
            if name:
                defaults[name] = tool.get("run_llm", False)
    return defaults


def tool_register(llm_service, pipecatcall_id: str, tool_names: List[str], tools_data: List[Dict[str, Any]] = None):
    """Register tool functions with the LLM service."""
    run_llm_defaults = _build_run_llm_defaults(tools_data)

    def create_wrapper(tool_name, pipecatcall_id, default_run_llm):
        async def wrapper(params: FunctionCallParams):
            return await tool_execute(tool_name, params, pipecatcall_id, default_run_llm=default_run_llm)
        return wrapper

    for tool_name in tool_names:
        default_run_llm = run_llm_defaults.get(tool_name, False)
        wrapper = create_wrapper(tool_name, pipecatcall_id, default_run_llm)
        # Single-AI registers directly on the raw pipecat LLMService (positional
        # name + handler), bypassing RoutingLLMService — single-AI has no router.
        # Team pipelines register via RoutingLLMService.register_function instead.
        llm_service.register_function(tool_name, wrapper)
        # Per-tool line (VOIP-1510): the aggregate summary line below names every
        # tool too (run_llm_defaults is keyed by name), but an EXPLICIT per-tool
        # confirmation is what a Loki grep for one specific tool_name needs to
        # answer "was this tool actually registered on this pipecatcall's LLM
        # service" without parsing the aggregate dict out of a longer message.
        logger.debug(f"[{tool_name}] Registered on LLM service for pipecatcall {pipecatcall_id}.")
    logger.info(f"Registered {len(tool_names)} tools for pipecatcall {pipecatcall_id}. run_llm_defaults: {run_llm_defaults}")


def tool_unregister(llm_service, tool_names: List[str]):
    """Unregisters tools from the LLM service."""
    for tool_name in tool_names:
        try:
            llm_service.unregister_function(tool_name)
        except KeyError:
            logger.debug(f"Tool '{tool_name}' was not registered or already removed.")
        except Exception as e:
            logger.warning(f"Error while unregistering tool '{tool_name}': {e}")


async def tool_execute(tool_name: str, params: FunctionCallParams, pipecatcall_id: str, default_run_llm: bool = False):
    """Generic executor for tool calls (connect, message_send, etc)."""

    args = params.arguments if isinstance(params.arguments, dict) else {}
    # tool_call_id included (VOIP-1510): this is the ONLY field that correlates
    # this log line with the RTVI llm-function-call-{started,in-progress,stopped}
    # frames bin-pipecat-manager's Go side already logs (receiveMessageFrameMessage
    # in pkg/pipecatcallhandler/runner.go) — those carry tool_call_id but never the
    # function name, so without this line a failure that never reaches this point
    # (i.e. never gets logged at all) is indistinguishable, from Loki alone, from
    # one whose tool_call_id we simply can't tie back to a specific tool_name.
    logger.info(f"[{tool_name}] Executing. tool_call_id={params.tool_call_id}, args: {json.dumps(args, ensure_ascii=False)}")

    should_run_llm = args.pop("run_llm", default_run_llm)

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
