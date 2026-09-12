.. _mcpserver-struct-mcpserver:

MCP Server
==========

.. _mcpserver-struct-mcpserver-mcpserver:

MCP Server
----------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "url": "<string>",
        "status": "<string>",
        "auth_type": "<string>",
        "api_key_header": "<string>",
        "oauth_vendor": "<string>",
        "has_secret": <boolean>,
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The MCP server registration's unique identifier. Returned when creating an MCP server via ``POST /mcp_servers`` or when listing via ``GET /mcp_servers``. Referenced from an AI's :ref:`mcp_server_ids <ai-struct-ai-tool_names>` list.
* ``customer_id`` (UUID): The customer that owns this MCP server registration. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (String, Required): A human-readable name for the MCP server (e.g., ``"Internal Ticketing MCP"``).
* ``detail`` (String, Optional): A description of the MCP server's purpose.
* ``url`` (String, Required): The MCP server's Streamable-HTTP endpoint. **Must be** ``https://`` **only** — plain ``http://`` URLs are rejected. The URL is also validated against private/loopback/link-local/multicast address ranges (including IPv4-mapped IPv6 forms) both before persisting and again at the moment of every outbound connection, to prevent the server from being used to reach internal infrastructure.
* ``status`` (enum string, Optional): The server's lifecycle state. See :ref:`Status <mcpserver-struct-mcpserver-status>`. Defaults to ``active``.
* ``auth_type`` (enum string, Optional): How outbound MCP calls authenticate against this server. See :ref:`Auth Type <mcpserver-struct-mcpserver-auth_type>`. Defaults to no authentication. ``oauth`` is set implicitly by completing ``POST /mcpservers/oauth/complete`` -- it can never be set directly via ``POST``/``PUT`` with a customer-supplied secret.
* ``api_key_header`` (String, Optional): The HTTP header name used to send the secret when ``auth_type`` is ``api_key`` (e.g., ``"X-API-Key"``). Ignored for other auth types.
* ``oauth_vendor`` (enum string): Which OAuth vendor this server is connected to (``github`` or ``linear``). Only set when ``auth_type`` is ``oauth``; empty otherwise.
* ``has_secret`` (Boolean): Whether a secret (bearer token, API key, or OAuth access token) is currently stored for this server. The secret/token value itself is never returned in any response.
* ``tm_create`` (String, ISO 8601): Timestamp when the MCP server was registered.
* ``tm_update`` (String, ISO 8601): Timestamp when the MCP server was last updated.
* ``tm_delete`` (String, ISO 8601): Timestamp when the MCP server was deleted, if applicable.

.. note:: **MCP Server Implementation Hint**

   The secret (bearer token or API key) is write-only: it is accepted on ``POST /mcp_servers`` and ``PUT /mcp_servers/{id}`` but is **never returned** in any ``GET`` response. On ``PUT``, omitting the secret field leaves the currently stored secret unchanged; sending an explicit value (including an empty string) replaces it. This distinguishes "I'm not touching the secret" from "clear the secret."

.. note:: **MCP Server Implementation Hint**

   ``PUT /mcp_servers/{id}`` is a true partial update: every field (``name``, ``detail``, ``url``, ``status``, ``auth_type``, ``api_key_header``, in addition to ``secret`` above) is optional, and omitting a field leaves its current value unchanged. A full resend of every field is never required -- for example, ``PUT {"name": "new name"}`` renames the server and leaves everything else (including ``url``, ``status``, and ``auth_type``) exactly as it was. One exception worth calling out: ``auth_type: ""`` is a valid, meaningful value (no authentication) and is distinct from omitting the ``auth_type`` field entirely -- sending the empty string explicitly sets no-auth, while omitting the field preserves whatever ``auth_type`` the server already had.

.. note:: **MCP Server Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` indicates the MCP server has not been deleted and is still active. This sentinel value is used across all VoIPBin resources to represent "not yet occurred."

Example
+++++++

.. code::

    {
        "id": "b1a2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Internal Ticketing MCP",
        "detail": "Exposes ticket lookup/create tools to AI assistants",
        "url": "https://mcp.internal.example.com/",
        "status": "active",
        "auth_type": "bearer",
        "api_key_header": "",
        "has_secret": true,
        "tm_create": "2026-09-11 03:00:00.000000",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _mcpserver-struct-mcpserver-status:

Status
------
The ``status`` field controls whether the MCP server's tools are made available to AIs that reference it.

================ =======================================
Status           Description
================ =======================================
active           Default. Tools are discovered and callable from any AI whose ``mcp_server_ids`` includes this server.
disabled         Customer has deactivated the server. Its tools are silently omitted from every referencing AI's tool list, and any in-flight call attempt against a previously-resolved tool from this server fails closed.
================ =======================================

.. _mcpserver-struct-mcpserver-auth_type:

Auth Type
---------
The ``auth_type`` field controls how outbound MCP requests to this server authenticate.

================ =======================================
Type             Description
================ =======================================
(empty)          No ``Authorization`` header is sent.
bearer           Sends ``Authorization: Bearer <secret>``.
api_key          Sends the secret under the header named by ``api_key_header`` (e.g., ``X-API-Key: <secret>``).
oauth            Sends ``Authorization: Bearer <access_token>`` using an OAuth 2.1 access token obtained via ``POST /mcpservers/oauth/start`` and ``POST /mcpservers/oauth/complete``. The access token is transparently refreshed when it expires, if the connected vendor issued a refresh token. Never set directly by the customer -- only by completing the OAuth flow. See :ref:`oauth_vendor <mcpserver-struct-mcpserver-mcpserver>` for which vendor a server is connected to.
================ =======================================
