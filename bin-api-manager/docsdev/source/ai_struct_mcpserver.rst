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

.. _mcpserver-struct-mcpserver-tool-use-scope:

.. note:: **Tool use scope (Normal-type AIs, single-AI sessions)**

   A whitelisted server's tools are presented to the AI's Normal-type, single-AI sessions -- realtime voice call sessions, ``type=insight`` AIs, and team-typed AI calls do not receive them (design docs/plans/2026-09-29-mcp-tool-exposure-pr-b2-design.md). Sessions that do discover and advertise include chat, ``ai_task`` flow actions and API-created single-AI sessions, so expect ``tools/list`` (and, when the model calls a tool, ``tools/call``) requests authenticated per this server's ``auth_type`` in its logs. Of what is discovered, the tool **names** are stored on the AI session record that resolved them, so they outlive the request; the input schemas and descriptions are held only for the lifetime of that resolution and are not stored. Stored names live as long as that session record does: deleting the MCP server, or the AI, does not remove names already written to past session records, and session deletion is a soft delete that retains the record. Ask support@voipbin.net if you need names purged from historical sessions. The source ranges these connections originate from are available on request from support@voipbin.net. A remote tool call result the server itself marks as an error is surfaced to the model as a failure, never mislabeled a success. Up to 256 tools are taken per session resolution across an AI's whitelisted servers, each name namespaced ``mcp_<first 8 hex chars of the server id>_<tool name>``; a server is skipped silently, rather than failing the session, when its ``status`` is not ``active``, when it has been deleted, or when it is no longer owned by this customer.

   **Tool input schemas.** VoIPBin advertises each tool's input schema to the AI model in a provider-neutral subset of JSON Schema: ``type``, ``description``, ``properties``, ``required``, ``items``, ``enum``, ``anyOf``, ``format``, ``minimum``, ``maximum``, ``minItems`` and ``maxItems``. ``oneOf``, ``const``, a single-member ``allOf``, local ``$ref`` (``#/$defs/...`` and ``#/definitions/...``) and a list-valued ``type`` are rewritten into that subset; every other keyword (for example ``x-*`` extensions, ``title``, ``default``, ``pattern``, ``additionalProperties``) is removed, and ``format`` and ``enum`` are kept only where every supported model accepts them. An optional parameter whose schema cannot be expressed this way is omitted from what the AI sees. A tool whose required parameters cannot be expressed is not offered to the AI at all; the server's other tools are still offered. The arguments the AI sends are forwarded to your server unchanged, so your server's own validation still applies and should report a failed call when a constraint the AI could not see is not met.

   Schema tips, so every tool and parameter reaches the AI:

   * Give every property a ``type`` (a property that accepts "any JSON value" cannot be expressed).
   * Give every ``array`` an ``items`` schema.
   * Avoid ``allOf`` with more than one member.
   * Prefer string enums (``{"type": "string", "enum": [...]}``); enums on other types are removed.

* ``id`` (UUID): The MCP server registration's unique identifier. Returned when creating an MCP server via ``POST /mcpservers`` or when listing via ``GET /mcpservers``. Referenced from an AI's :ref:`mcp_server_ids <ai-struct-ai-tool_names>` list.
* ``customer_id`` (UUID): The customer that owns this MCP server registration. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (String, Required): A human-readable name for the MCP server (e.g., ``"Internal Ticketing MCP"``).
* ``detail`` (String, Optional): A description of the MCP server's purpose.
* ``url`` (String, Required): The MCP server's Streamable-HTTP endpoint. **Must be** ``https://`` **only** (plain ``http://`` URLs are rejected). The URL is also validated against private/loopback/link-local/multicast address ranges (including IPv4-mapped IPv6 forms) both before persisting and again at the moment of every outbound connection, to prevent the server from being used to reach internal infrastructure. **Immutable while** ``auth_type`` **is** ``oauth``: the OAuth flow, not this field, wrote it to the vendor's own fixed endpoint, and changing it would send the stored vendor tokens to an arbitrary address. ``PUT`` rejects a real change with ``MCP_SERVER_OAUTH_URL_IMMUTABLE``; re-sending the current value is accepted.
* ``status`` (enum string, Optional): The server's lifecycle state. See :ref:`Status <mcpserver-struct-mcpserver-status>`. Defaults to ``active``.
* ``auth_type`` (enum string, Optional): How outbound MCP calls authenticate against this server. See :ref:`Auth Type <mcpserver-struct-mcpserver-auth_type>`. Defaults to no authentication. A server is moved **into** ``oauth`` only by completing ``POST /mcpservers/oauth/complete``; sending ``oauth`` on a server that is not already connected is rejected with ``INVALID_MCP_SERVER_AUTH_TYPE``. Re-sending the current ``oauth`` value on an already-connected server is accepted, so a client that submits the full object on every save is never rejected. Moving a connected server **out** of ``oauth`` is allowed and erases its stored OAuth tokens.
* ``api_key_header`` (String, Optional): The HTTP header name used to send the secret when ``auth_type`` is ``api_key`` (e.g., ``"X-API-Key"``). Ignored for other auth types.
* ``oauth_vendor`` (enum string): Which OAuth vendor this server is connected to (``github`` or ``linear``). Only set when ``auth_type`` is ``oauth``; empty otherwise.
* ``has_secret`` (Boolean): Whether a secret (bearer token, API key, or OAuth access token) is currently stored for this server. The secret/token value itself is never returned in any response.
* ``tm_create`` (String, ISO 8601): Timestamp when the MCP server was registered.
* ``tm_update`` (String, ISO 8601): Timestamp when the MCP server was last updated.
* ``tm_delete`` (String or null, ISO 8601): Timestamp when the MCP server was deleted. ``null`` while the server has not been deleted.

.. note:: **MCP Server Implementation Hint**

   The secret (bearer token or API key) is write-only: it is accepted on ``POST /mcpservers`` and ``PUT /mcpservers/{id}`` but is **never returned** in any ``GET`` response. On ``PUT``, omitting the secret field leaves the currently stored secret unchanged; sending an explicit value (including an empty string) replaces it. This distinguishes "I'm not touching the secret" from "clear the secret."

.. note:: **MCP Server Implementation Hint**

   ``PUT /mcpservers/{id}`` is a true partial update: every field (``name``, ``detail``, ``url``, ``status``, ``auth_type``, ``api_key_header``, in addition to ``secret`` above) is optional, and omitting a field leaves its current value unchanged. A full resend of every field is never required -- for example, ``PUT {"name": "new name"}`` renames the server and leaves everything else (including ``url``, ``status``, and ``auth_type``) exactly as it was. One exception worth calling out: ``auth_type: ""`` is a valid, meaningful value (no authentication) and is distinct from omitting the ``auth_type`` field entirely -- sending the empty string explicitly sets no-auth, while omitting the field preserves whatever ``auth_type`` the server already had.

.. note:: **MCP Server Implementation Hint**

   MCP servers do **not** use the ``9999-01-01 00:00:00.000000`` sentinel that some older VoIPBin resources use for "not yet occurred." ``tm_update`` and ``tm_delete`` are ``null`` until the server is first updated or deleted, so test for ``null`` rather than comparing against a sentinel date.

.. note:: **Deleting a server destroys its credentials**

   ``DELETE /mcpservers/{id}`` is a revocation, not an archive. The stored secret, and any OAuth access and refresh tokens, are erased at the moment of deletion and cannot be recovered -- not by VoIPBin support either. The record itself is retained so that audit history and existing AI references stay readable, but ``has_secret`` becomes ``false`` and the server is excluded from every AI that references it immediately, including from tool discovery and dispatch; see the note under :ref:`Status <mcpserver-struct-mcpserver-status>`.

   Deletion is not the only irreversible path. Changing ``auth_type`` away from ``oauth`` also erases the stored OAuth access and refresh tokens, and clears ``oauth_vendor``: the server is being told to authenticate a different way, so the connection it replaces is not kept. Re-running ``POST /mcpservers/oauth/start`` is the way back, not an undo.

   To use the same endpoint again, register a new MCP server and supply the secret again (or complete the OAuth flow again). If you only want to stop the server temporarily, set ``status`` to ``disabled`` instead: that keeps the credential and is reversible.

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
        "tm_update": null,
        "tm_delete": null
    }

.. _mcpserver-struct-mcpserver-status:

Status
------
The ``status`` field controls whether the MCP server's tools are made available to AIs that reference it.

.. note:: **Tool use scope**

   Which sessions receive a whitelisted server's tools, and how their input schemas are advertised, is described in :ref:`Tool use scope <mcpserver-struct-mcpserver-tool-use-scope>`.

   **Setting the status to disabled stops VoIPBin connecting to this server, and stops its tools being offered to or called from any referencing AI, immediately.**

   An AI's callable actions come from its built-in ``tool_names`` set plus any whitelisted MCP server's tools (Normal-type, single-AI sessions); custom logic beyond either is reached from a :ref:`Flow <flow-overview>`.

================ =======================================
Status           Description
================ =======================================
active           Default. Tools are discovered and made callable for every Normal-type, single-AI session of any AI whose ``mcp_server_ids`` includes this server (not realtime voice calls, Insight AIs, or team AI calls).
disabled         Customer has deactivated the server. It is skipped during discovery, its tools are omitted from every referencing AI's tool list, and any in-flight call attempt against a previously-resolved tool from this server fails closed.
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
oauth            Sends ``Authorization: Bearer <access_token>`` using an OAuth 2.1 access token obtained via ``POST /mcpservers/oauth/start`` and ``POST /mcpservers/oauth/complete``. The access token is transparently refreshed when it expires, if the connected vendor issued a refresh token. Never entered directly by the customer -- a server enters this state only by completing the OAuth flow, though re-sending the current value on an already-connected server is accepted. Leaving this state erases the stored tokens. See :ref:`oauth_vendor <mcpserver-struct-mcpserver-mcpserver>` for which vendor a server is connected to.
================ =======================================
