.. _service_agent-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free -- the Agent Console surface itself has no per-call charge. Underlying resources (AI calls, transcription, ...) keep their own billing, unchanged by which path (``/service_agents/*`` vs the top-level path) was used to reach them.
   * **Async:** No. Every ``/service_agents/*`` endpoint documented on this page returns synchronously. Real-time push (new messages, case updates) is delivered over the WebSocket connection described below.

``/service_agents/*`` is a parallel, agent-scoped path prefix that mirrors a large slice of VoIPBIN's REST surface (agents, calls, contacts, AI calls, tags, extensions, transcription, files, conversations) plus a handful of endpoints that exist **only** under this prefix: case management, agent self-service (``me``), and public-channel discovery for Talk. This page documents the authentication/authorization model shared by all ``/service_agents/*`` endpoints and the pieces of the surface that are genuinely new. For everything else, see the resource's own page (:ref:`Contact <contact-main>`, :ref:`Call <call-main>`, :ref:`AI <ai-main>`, :ref:`Agent <agent-main>`, :ref:`Talk <talk-main>`, :ref:`Transcribe <transcribe-main>`, :ref:`Storage <storage-main>`, :ref:`Tag <tag-main>`, :ref:`Extension <extension-main>`, :ref:`Customer <customer-main>`) -- the request/response shapes are identical, only the path prefix and the permission check differ.

Why a separate path prefix
---------------------------
VoIPBIN exposes two REST surfaces backed by the same resources, deliberately kept separate:

.. list-table::
   :header-rows: 1

   * - Surface
     - Consumers
     - Typical permission requirement
   * - Top-level (e.g. ``/contacts``, ``/transcribes``)
     - Admin/manager consoles (square-admin)
     - ``PermissionCustomerAdmin`` and/or ``PermissionCustomerManager``
   * - ``/service_agents/*``
     - Agent-facing consoles (talk.voipbin.net, square-talk)
     - ``PermissionAll`` -- any authenticated agent of the customer

Both surfaces validate the same JWT and enforce the same tenant isolation (an agent can only ever see its own customer's data, unless it holds project-superadmin permission). The difference is the *authorization bitmask*: top-level endpoints are gated to admins/managers and are free to tighten or loosen that bitmask to fit the admin console's needs over time, while ``/service_agents/*`` endpoints exist specifically so that a rank-and-file agent (no admin/manager permission bits at all) can use them, and so that this permission model can evolve independently.

.. note:: **AI Implementation Hint**

   Agent-facing frontends should call **only** ``/service_agents/*`` paths, never the top-level ``/<resource>`` path directly -- even where the top-level path's current permission bitmask happens to allow it. The two surfaces are versioned independently; relying on the top-level path risks breaking silently if its admin-console permission requirements change.

Authentication
---------------
``/service_agents/*`` endpoints authenticate exactly like the rest of the API: a JWT issued by ``POST /auth/boot`` (agent login), supplied as a cookie (``token=<jwt>``), a query parameter (``?token=<jwt>``), or an ``Authorization: Bearer <jwt>`` header. See :ref:`Agent <agent-overview>` for the login flow. There is no separate "service agent" credential type -- any authenticated agent JWT works against ``/service_agents/*``; what changes is which permission bitmask (``PermissionAll`` instead of admin/manager bits) each endpoint checks, and that every response is pre-scoped to ``agent.customer_id`` (and, for self-service and case-ownership endpoints, to ``agent.id``) so the caller never has to pass a ``customer_id`` explicitly.

Direct tokens (``type: "direct"``, used for machine-to-machine integrations) are rejected by most ``/service_agents/*`` endpoints -- case creation/notes, contact-case attach/detach, and interaction listing explicitly require a genuine agent identity, since several of these endpoints derive an author/owner from the caller's own agent ID.

Endpoint catalog
-----------------
The table below groups every ``/service_agents/*`` path by the resource it fronts. **New** marks endpoints that exist only under this prefix and are documented in full below (or, for cases, on :ref:`Case management <service_agent-case-overview>`). Everything else is a thin, agent-scoped mirror of the resource's existing documentation.

.. list-table::
   :header-rows: 1

   * - Path
     - Documented on
     - Notes
   * - ``GET /service_agents/agents``, ``/agents/{id}``
     - :ref:`Agent <agent-main>`
     -
   * - ``GET/PUT /service_agents/me``, ``/me/addresses``, ``/me/password``, ``/me/status``
     - This page (below)
     - **New** -- self-service, no ``{id}`` needed
   * - ``GET/POST /service_agents/calls``, ``/calls/{id}``
     - :ref:`Call <call-main>`
     -
   * - ``GET/POST /service_agents/aicalls``, ``/aimessages``
     - :ref:`AI <ai-main>`
     -
   * - ``GET/POST/PUT/DELETE /service_agents/contacts*``
     - :ref:`Contact <contact-main>`
     -
   * - ``GET/POST/PUT/DELETE /service_agents/contact_addresses*``
     - :ref:`Contact <contact-main>`
     -
   * - ``GET /service_agents/contact_interactions``, ``/contact_peer_events``
     - :ref:`Contact <contact-peer-event-overview>`
     - Agent-scoped equivalent of ``GET /contact_interactions``/``/contact_peer_events`` -- identical response shape
   * - ``GET/POST/PUT/DELETE /service_agents/contact_cases*``
     - :ref:`Case management <service_agent-case-overview>`
     - **New**
   * - ``GET/POST /service_agents/conversations*``
     - :ref:`Conversation <conversation-main>`
     -
   * - ``GET /service_agents/customer``
     - :ref:`Customer <customer-main>`
     -
   * - ``GET /service_agents/extensions*``
     - :ref:`Extension <extension-main>`
     -
   * - ``GET/POST/DELETE /service_agents/files*``
     - :ref:`Storage <storage-main>`
     -
   * - ``GET /service_agents/tags*``
     - :ref:`Tag <tag-main>`
     -
   * - ``GET /service_agents/talk_channels``
     - This page (below)
     - **New** -- public channel discovery
   * - ``GET/POST/PUT/DELETE /service_agents/talk_chats*``, ``/talk_messages*``
     - :ref:`Talk <talk-main>`
     - ``POST .../talk_chats/{id}/join`` documented on this page (**New**)
   * - ``GET/POST /service_agents/transcribes``, ``/transcripts``
     - :ref:`Transcribe <transcribe-main>`
     -
   * - ``GET /service_agents/ws``
     - :ref:`WebSocket <websocket-main>`
     - Agent-scoped WebSocket connection (below)


Agent Self-Service (``/service_agents/me``)
---------------------------------------------
The ``me`` endpoints let an agent manage its own profile without knowing (or needing permission to look up) its own agent ID -- the identity is always taken from the authenticated JWT.

.. list-table::
   :header-rows: 1

   * - Method
     - Path
     - Description
   * - GET
     - ``/service_agents/me``
     - Get the authenticated agent's own details.
   * - PUT
     - ``/service_agents/me``
     - Update ``name``, ``detail``, ``ring_method``.
   * - PUT
     - ``/service_agents/me/addresses``
     - Replace the agent's own contact addresses.
   * - PUT
     - ``/service_agents/me/password``
     - Change the agent's own login password.
   * - PUT
     - ``/service_agents/me/status``
     - Set the agent's own availability status.

Every ``me`` response is the same :ref:`Agent <agent-struct-agent-agent>` object returned by ``GET /agents/{id}``. ``ring_method`` and ``status`` use the same enums documented there -- see :ref:`Ring method <agent-struct-agent-ring_method>` and :ref:`Status <agent-struct-agent-status>`.

**Get own profile:**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/me?token=<token>'

**Set status to available:**

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/service_agents/me/status?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "status": "available"
        }'

**Change own password:**

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/service_agents/me/password?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "password": "<new-password>"
        }'

**Update own addresses:**

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/service_agents/me/addresses?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "addresses": [
                {
                    "type": "tel",
                    "target": "+155****1234"
                }
            ]
        }'

.. note:: **AI Implementation Hint**

   ``PUT /service_agents/me/addresses`` replaces the agent's entire address list -- it is not additive. Fetch the current list via ``GET /service_agents/me`` first if you only need to add or remove one address.


Discovering and Joining Public Channels
------------------------------------------
:ref:`Talk <talk-main>` already documents creating chats and sending messages via ``POST /service_agents/talk_chats``. Two related endpoints exist only for the ``talk`` chat type (public, topic-based channels):

.. list-table::
   :header-rows: 1

   * - Method
     - Path
     - Description
   * - GET
     - ``/service_agents/talk_channels``
     - List every public ``talk``-type channel for the customer, regardless of whether the caller has joined.
   * - POST
     - ``/service_agents/talk_chats/{id}/join``
     - Join a ``talk``-type chat as a participant (convenience wrapper over the participants endpoint).

``GET /service_agents/talk_chats`` (documented on the :ref:`Talk <talk-main>` page) returns only chats the caller has already joined. ``GET /service_agents/talk_channels`` is the discovery endpoint: it returns every public channel for the customer so an agent's UI can present a "browse channels" list, independent of participation.

**List public channels:**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/talk_channels?token=<token>'

**Join a channel:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_chats/<channel-id>/join?token=<token>'

The join endpoint only works for ``talk``-type chats. For ``direct`` and ``group`` chats, add participants explicitly via ``POST /service_agents/talk_chats/{id}/participants`` (see :ref:`Talk <talk-overview>`).


WebSocket connection
-----------------------
``GET /service_agents/ws`` upgrades the connection to a WebSocket, scoped to the authenticated agent, using the same protocol documented in :ref:`WebSocket <websocket-main>`. Use this path (rather than the top-level ``/ws``) from agent-facing consoles so that event delivery stays consistent with the rest of the ``/service_agents/*`` surface.

.. code::

    $ wscat -c 'wss://api.voipbin.net/v1.0/service_agents/ws?token=<token>'
