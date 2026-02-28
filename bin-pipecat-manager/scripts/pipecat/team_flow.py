import json
import aiohttp
import asyncio
import common

from loguru import logger
from pipecat_flows import FlowManager, FlowArgs, FlowsFunctionSchema, NodeConfig


def build_team_flow(
    resolved_team: dict,
    pipecatcall_id: str,
    routing_llm,
    routing_tts,
    routing_stt,
) -> tuple[dict, NodeConfig]:
    """Build NodeConfig objects for all team members.

    Args:
        resolved_team: The resolved team dict from Go.
        pipecatcall_id: The pipecatcall ID for tool endpoint calls.
        routing_llm: RoutingLLMService instance.
        routing_tts: RoutingTTSService instance (may be None).
        routing_stt: RoutingSTTService instance (may be None).

    Returns:
        Tuple of (member_nodes dict mapping member_id -> NodeConfig, start_node NodeConfig).
    """
    member_nodes = {}

    for member in resolved_team["members"]:
        member_id = member["id"]
        member_name = member["name"]
        ai = member["ai"]

        # Build regular tool functions (call Go's HTTP endpoint)
        tool_functions = []
        for tool in member.get("tools", []):
            tool_functions.append(FlowsFunctionSchema(
                name=tool["name"],
                description=tool.get("description", ""),
                properties=tool.get("parameters", {}).get("properties", {}),
                required=tool.get("parameters", {}).get("required", []),
                handler=_create_tool_handler(tool["name"], pipecatcall_id),
            ))

        # Build transition functions
        for transition in member.get("transitions", []):
            tool_functions.append(FlowsFunctionSchema(
                name=transition["function_name"],
                description=transition["description"],
                properties={},
                required=[],
                handler=_create_transition_handler(
                    transition["next_member_id"],
                    member_nodes,
                    routing_llm,
                    routing_tts,
                    routing_stt,
                ),
            ))

        # Build NodeConfig (TypedDict)
        role_messages = []
        if ai.get("init_prompt"):
            role_messages.append({
                "role": "system",
                "content": ai["init_prompt"],
            })

        node: NodeConfig = {
            "name": member_name,
            "role_messages": role_messages,
            "task_messages": [],
            "functions": tool_functions,
        }
        member_nodes[member_id] = node

    start_node = member_nodes[resolved_team["start_member_id"]]
    return member_nodes, start_node


def _create_tool_handler(tool_name: str, pipecatcall_id: str):
    """Create a FlowsFunctionSchema handler that calls Go's tool endpoint."""
    async def handler(args: FlowArgs, flow_manager: FlowManager):
        result = await _call_go_tool_endpoint(tool_name, args, pipecatcall_id)
        return result, None  # None = stay on current node
    return handler


def _create_transition_handler(
    next_member_id: str,
    member_nodes: dict,
    routing_llm,
    routing_tts,
    routing_stt,
):
    """Create a FlowsFunctionSchema handler for member transitions."""
    async def handler(args: FlowArgs, flow_manager: FlowManager):
        # Save current active member for rollback
        prev_llm = routing_llm._active_id
        prev_tts = routing_tts._active_id if routing_tts else None
        prev_stt = routing_stt._active_id if routing_stt else None

        try:
            routing_llm.set_active_member(next_member_id)
            if routing_tts:
                routing_tts.set_active_member(next_member_id)
            if routing_stt:
                routing_stt.set_active_member(next_member_id)
        except Exception as e:
            logger.error(f"Transition to {next_member_id} failed, rolling back: {e}")
            # Rollback
            if prev_llm:
                routing_llm.set_active_member(prev_llm)
            if routing_tts and prev_tts:
                routing_tts.set_active_member(prev_tts)
            if routing_stt and prev_stt:
                routing_stt.set_active_member(prev_stt)
            return {"error": str(e)}, None

        next_node = member_nodes.get(next_member_id)
        if next_node is None:
            logger.error(f"No NodeConfig for member {next_member_id}")
            return {"error": f"unknown member {next_member_id}"}, None

        logger.info(f"Transition to member {next_member_id} successful")
        return {"status": "transferred"}, next_node
    return handler


async def _call_go_tool_endpoint(tool_name: str, args: dict, pipecatcall_id: str) -> dict:
    """Call Go's tool execution HTTP endpoint."""
    http_url = f"{common.PIPECATCALL_HTTP_URL}/{pipecatcall_id}/tools"
    http_body = {
        "id": f"team-tool-{tool_name}",
        "type": "function",
        "function": {
            "name": tool_name,
            "arguments": json.dumps(args if isinstance(args, dict) else {}, ensure_ascii=False),
        },
    }
    logger.debug(f"[team_flow][{tool_name}] POST {http_url}")

    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(http_url, json=http_body, timeout=aiohttp.ClientTimeout(total=10)) as response:
                text = await response.text()
                if response.status >= 400:
                    logger.warning(f"[team_flow][{tool_name}] HTTP {response.status}: {text[:500]}")
                    return {"status": "error", "error": f"HTTP {response.status}: {text}"}

                content_type = response.headers.get("Content-Type", "")
                if content_type.startswith("application/json"):
                    try:
                        return {"status": "ok", "data": json.loads(text)}
                    except json.JSONDecodeError:
                        return {"status": "ok", "data": {"raw": text}}
                return {"status": "ok", "data": {"raw": text}}

    except asyncio.TimeoutError:
        logger.error(f"[team_flow][{tool_name}] Request timed out")
        return {"status": "error", "error": "Request timed out"}
    except Exception as e:
        logger.exception(f"[team_flow][{tool_name}] Unexpected error: {e}")
        return {"status": "error", "error": str(e)}
