"""MCP reference server used by bin-ai-manager's conformance test.

Serves the one tool the Go test expects, in either the SDK's default
(stateful, SSE-framed) mode or its stateless JSON mode, selected by the
STATELESS environment variable. Run with uvicorn:

    uvicorn mcp_reference_server:app --port 8871
    STATELESS=1 uvicorn mcp_reference_server:app --port 8872
"""

import os

from mcp.server.mcpserver import MCPServer

stateless = os.environ.get("STATELESS") == "1"
server = MCPServer("voipbin-conformance")


@server.tool()
def lookup_order(order_id: str, verbose: bool = False) -> str:
    """Look up an order by id."""
    return f"order {order_id} verbose={verbose}"


app = server.streamable_http_app(json_response=stateless, stateless_http=stateless)
