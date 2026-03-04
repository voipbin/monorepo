import json
import re
import aiohttp
import asyncio
import common

from loguru import logger
from pipecat_flows import FlowManager, FlowArgs, FlowsFunctionSchema, NodeConfig

# Google Gemini requires function names: start with letter/underscore,
# alphanumeric + _.-: only, max 64 chars.
_MAX_FUNCTION_NAME_LENGTH = 64
_INVALID_CHARS_RE = re.compile(r'[^a-zA-Z0-9_.\-:]')


def _sanitize_function_name(name: str) -> str:
    """Sanitize a function name for LLM provider compatibility (Gemini 64-char limit)."""
    sanitized = _INVALID_CHARS_RE.sub('_', name)
    if sanitized and not sanitized[0].isalpha() and sanitized[0] != '_':
        sanitized = '_' + sanitized
    if len(sanitized) > _MAX_FUNCTION_NAME_LENGTH:
        logger.warning(
            f"Function name truncated from {len(sanitized)} to {_MAX_FUNCTION_NAME_LENGTH} chars: "
            f"'{sanitized}' -> '{sanitized[:_MAX_FUNCTION_NAME_LENGTH]}'"
        )
        sanitized = sanitized[:_MAX_FUNCTION_NAME_LENGTH]
    return sanitized


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
    current_state = {"active_member_id": resolved_team["start_member_id"]}

    for member in resolved_team["members"]:
        member_id = member["id"]
        member_name = member["name"]
        ai = member["ai"]

        # Build regular tool functions (call Go's HTTP endpoint)
        tool_functions = []
        for tool in member.get("tools", []):
            tool_functions.append(FlowsFunctionSchema(
                name=_sanitize_function_name(tool["name"]),
                description=tool.get("description", ""),
                properties=tool.get("parameters", {}).get("properties", {}),
                required=tool.get("parameters", {}).get("required", []),
                handler=_create_tool_handler(tool["name"], pipecatcall_id),
            ))

        # Build transition functions
        for transition in member.get("transitions", []):
            tool_functions.append(FlowsFunctionSchema(
                name=_sanitize_function_name(transition["function_name"]),
                description=transition["description"],
                properties={},
                required=[],
                handler=_create_transition_handler(
                    transition["next_member_id"],
                    member_nodes,
                    routing_llm,
                    routing_tts,
                    routing_stt,
                    current_state,
                    pipecatcall_id,
                    resolved_team,
                    transition["function_name"],
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

    start_node = member_nodes.get(resolved_team["start_member_id"])
    if start_node is None:
        raise ValueError(
            f"start_member_id {resolved_team['start_member_id']} not found in member_nodes. "
            f"Known members: {list(member_nodes.keys())}"
        )
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
    current_state: dict,
    pipecatcall_id: str,
    resolved_team: dict,
    function_name: str,
):
    """Create a FlowsFunctionSchema handler for member transitions."""
    async def handler(args: FlowArgs, flow_manager: FlowManager):
        # Validate target member exists before switching any services
        next_node = member_nodes.get(next_member_id)
        if next_node is None:
            logger.error(f"No NodeConfig for member {next_member_id}")
            return {"error": f"unknown member {next_member_id}"}, None

        from_member_id = current_state["active_member_id"]

        routing_llm.set_active_member(next_member_id)
        if routing_tts:
            routing_tts.set_active_member(next_member_id)
        if routing_stt:
            routing_stt.set_active_member(next_member_id)

        current_state["active_member_id"] = next_member_id

        # Fire-and-forget notification to Go
        asyncio.create_task(_notify_member_switched(
            pipecatcall_id, from_member_id, next_member_id,
            function_name, resolved_team,
        ))

        logger.info(f"Transition to member {next_member_id} successful")
        return {"status": "transferred"}, next_node
    return handler


def _find_member(resolved_team: dict, member_id: str) -> dict | None:
    """Find a member dict by ID in the resolved team."""
    for member in resolved_team.get("members", []):
        if member["id"] == member_id:
            return member
    return None


def _build_member_info(member: dict) -> dict:
    """Build a MemberInfo dict from a resolved member, excluding sensitive fields."""
    ai = member.get("ai", {})
    return {
        "id": member["id"],
        "name": member.get("name", ""),
        "engine_model": ai.get("engine_model", ""),
        "tts_type": ai.get("tts_type", ""),
        "tts_voice_id": ai.get("tts_voice_id", ""),
        "stt_type": ai.get("stt_type", ""),
    }


async def _notify_member_switched(
    pipecatcall_id: str,
    from_member_id: str,
    to_member_id: str,
    function_name: str,
    resolved_team: dict,
):
    """Fire-and-forget HTTP notification to Go about a member switch."""
    from_member = _find_member(resolved_team, from_member_id)
    to_member = _find_member(resolved_team, to_member_id)

    if from_member is None or to_member is None:
        logger.warning(
            f"[team_flow] Could not find member details for notification. "
            f"from={from_member_id} to={to_member_id}"
        )
        return

    http_url = f"{common.PIPECATCALL_HTTP_URL}/{pipecatcall_id}/member-switched"
    http_body = {
        "transition_function_name": function_name,
        "from_member": _build_member_info(from_member),
        "to_member": _build_member_info(to_member),
    }

    try:
        session = await _get_http_session()
        async with session.post(http_url, json=http_body) as response:
            if response.status >= 400:
                text = await response.text()
                logger.warning(f"[team_flow][member-switched] HTTP {response.status}: {text[:500]}")
            else:
                logger.debug(f"[team_flow][member-switched] Notification sent successfully")
    except Exception as e:
        logger.warning(f"[team_flow][member-switched] Failed to notify: {e}")


# Module-level shared HTTP session for connection reuse across tool calls.
# Safe in CPython asyncio: no await between check and assignment in _get_http_session,
# so no coroutine interleaving is possible within a single event loop.
_http_session: aiohttp.ClientSession | None = None


async def _get_http_session() -> aiohttp.ClientSession:
    global _http_session
    if _http_session is None or _http_session.closed:
        _http_session = aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=10))
    return _http_session


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
        session = await _get_http_session()
        async with session.post(http_url, json=http_body) as response:
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
