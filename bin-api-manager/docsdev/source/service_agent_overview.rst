.. _service_agent-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free -- the Agent Console surface itself has no per-call charge. Underlying resources (AI calls, transcription, ...) keep their own billing, unchanged by which path (``/service_agents/*`` vs the top-level path) was used to reach them.
   * **Async:** No. Every ``/service_agents/*`` endpoint documented on this page returns synchronously. Real-time push (new messages, case updates) is delivered over the WebSocket connection described below.

``/service_agents/*`` is a parallel, agent-scoped path prefix that mirrors a large slice of VoIPBin's REST surface (agents, calls, contacts, AI calls, tags, extensions, transcription, files, conversations) plus a handful of endpoints that exist **only** under this prefix: case management, agent self-service (``me``), and public-channel discovery for Talk. This page documents the authentication/authorization model shared by all ``/service_agents/*`` endpoints and the pieces of the surface that are genuinely new. For everything else, see the resource's own page (:ref:`Contact <contact-main>`, :ref:`Call <call-main>`, :ref:`AI <ai-main>`, :ref:`Agent <agent-main>`, :ref:`Talk <talk-main>`, :ref:`Transcribe <transcribe-main>`, :ref:`Storage <storage-main>`, :ref:`Tag <tag-main>`, :ref:`Extension <extension-main>`, :ref:`Customer <customer-main>`) -- the request/response shapes are identical, only the path prefix and the permission check differ.

Why a separate path prefix
---------------------------
VoIPBin exposes two REST surfaces backed by the same resources, deliberately kept separate:

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
   * - ``GET /service_agents/calls/{id}/queuecall``
     - This page
     - **New** -- the queue a call the agent owns was routed through. Returns ``id`` (queuecall), ``queue_id``, ``queue_name``, ``status`` and ``tm_create``; ``404`` if the call has no queue entry handled by the calling agent
   * - ``POST /service_agents/calls/{id}/participants``
     - :ref:`Adding a participant <call-transfer-add_participant>`
     - **New** -- brings a colleague's extension into a two-party call the agent owns, which makes it a three-party call. ``address`` takes exactly one address of type ``extension`` whose ``target`` is the extension ID of the same customer. Returns the group call of the new leg (``call_ids``). The call must be in ``progressing`` status and have two parties; a call with three parties, a request while another participant is being added, and an extension without a registered phone return ``409``. When the agent places the call and leaves, the call ends for everyone
   * - ``POST /service_agents/transfers``
     - :ref:`Transfer <call-transfer>`
     - **New** -- blind or attended transfer of a call the agent owns to a colleague. ``transferee_addresses`` takes exactly one address of type ``extension`` whose ``target`` is the extension ID (the ``target`` of the extension address returned by ``GET /service_agents/agents``) of the same customer. The call must be in ``progressing`` status and bridged; other address types and unknown or foreign extensions return ``400``. Cancelling an attended transfer is not available
   * - ``GET/POST /service_agents/aicalls``, ``/aimessages``
     - :ref:`AI <ai-main>`
     -
   * - ``POST /service_agents/aimessages/draft-decision``
     - :ref:`Insight Tools <ai-struct-tool-draft_message>`
     - **New** -- approves or rejects an SMS draft card (``draft_message``) or an email draft card (``draft_email``) that the Insight Assistant produced. Body: ``aicall_id``, ``draft_id`` (the ``id`` of the ``tool`` message that carries the ``sms_draft`` or ``email_draft`` block) and ``decision`` (``approve`` or ``reject``). Returns the ``tool`` message that records the result. A draft can be decided once (``409`` ``DRAFT_ALREADY_DECIDED``) and only within 10 minutes of its creation by default (``409`` ``DRAFT_EXPIRED``); an unknown draft, or a draft of another AI call or customer, returns ``404`` ``DRAFT_NOT_FOUND``; an approve of a stored draft that no longer passes validation returns ``400`` ``DRAFT_INVALID``
   * - ``GET/POST/PUT/DELETE /service_agents/contacts*``
     - :ref:`Contact <contact-main>`
     -
   * - ``GET/POST/PUT/DELETE /service_agents/contact_addresses*``
     - :ref:`Contact <contact-main>`
     -
   * - ``PUT/DELETE /service_agents/contacts/{id}/star``, ``GET /service_agents/contact_stars``, ``GET /service_agents/contacts?starred=true``
     - :ref:`Starred contacts <contact-overview-starred>`
     - **New** -- the calling agent's private contact stars
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
     - Customer-wide: returns every extension of the agent's customer. ``password`` and ``direct_hash`` are filled in only for the extensions assigned to the calling agent (empty strings otherwise)
   * - ``GET/POST/DELETE /service_agents/files*``
     - :ref:`Storage <storage-main>`
     -
   * - ``GET /service_agents/tags*``
     - :ref:`Tag <tag-main>`
     -
   * - ``GET /service_agents/talk_channels``
     - This page (below)
     - **New** -- public channel discovery
   * - ``GET /service_agents/talk_mentions``
     - :ref:`Talk <talk-main>`
     - **New** -- messages that mention the calling agent
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
     - Replace the agent's own contact addresses. ``extension`` addresses can only be kept or removed, not added or changed.
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

   ``PUT /service_agents/me/addresses`` replaces the agent's entire address list -- it is not additive. Fetch the current list via ``GET /service_agents/me`` first if you only need to add or remove one address. ``tel`` and ``sip`` addresses can be freely added, changed and removed, and a ``tel`` address must contain at least one digit. An ``extension`` address can only be kept (send it back unchanged) or removed, so sending only the ``tel`` addresses removes the stored ``extension``. Adding or changing an ``extension`` is rejected with ``403`` (``EXTENSION_ADDRESS_ADMIN_ONLY``) regardless of the caller's role, because it is assigned by an admin through ``PUT /agents/{id}/addresses`` or ``POST /agents``. An extension that is stored in a non-canonical form can only be removed. Ask an admin to correct it.


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
