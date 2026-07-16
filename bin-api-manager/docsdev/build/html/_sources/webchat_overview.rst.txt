.. _webchat-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Medium -- three resources (Widget, Session, Message) with two distinct auth models: customer-JWT for admin CRUD, and a widget-scoped direct-token JWT for the anonymous visitor flow.
   * **Cost:** No direct charge for webchat operations.
   * **Async:** Session creation and message send are synchronous; the Flow triggered on a Session's first inbound message runs asynchronously. Subscribe to the WebSocket (``GET /ws``) to receive real-time outbound messages.

VoIPBin's webchat feature lets you embed a live chat widget on your website. Anonymous website visitors can start a conversation without creating an account; the first message they send triggers your configured Flow, which can route the conversation to an AI assistant or a human agent.

Domain model
------------
Three resources work together:

* **Widget** -- a customer's chat widget configuration (name, welcome message, the Flow to trigger, appearance theme). Created and managed via customer-JWT authenticated admin endpoints. Also issues a **direct hash**, used to authenticate anonymous visitors.
* **Session** -- a single visitor's continuity token for one browsing session. ``Session.id`` is the value your frontend uses to send/receive messages for that visitor.
* **Message** -- an individual chat message, either ``inbound`` (from the visitor) or ``outbound`` (from your Flow, an AI, or a human agent).

Visitor authentication flow
----------------------------
The visitor-facing flow never uses a customer JWT. Instead:

::

    +------------------------------------------------------------------+
    |                     Visitor Authentication Flow                   |
    +------------------------------------------------------------------+

    1. Widget embed script          2. POST /auth/boot          3. Widget-scoped JWT
    +-------------------+           +-----------------+          +------------------+
    | direct_hash        |--------->| Exchange hash   |--------->| Direct-scope JWT |
    | baked into script   |          | for a JWT       |          | (allowed:        |
    +-------------------+           +-----------------+          |  webchat_session)|
                                                                   +--------+---------+
                                                                            |
                                                                            v
                                                                  +------------------+
                                                                  | POST /webchat_    |
                                                                  | sessions           |
                                                                  | (creates Session,  |
                                                                  |  returns welcome_  |
                                                                  |  message)          |
                                                                  +------------------+

1. The widget embed script (delivered by your site) carries the widget's ``direct_hash``.
2. On page load, the script calls ``POST /auth/boot`` with the hash to exchange it for a short-lived, widget-scoped JWT.
3. That JWT authorizes exactly one resource type (``webchat_session``) and is scoped to the specific Widget it was issued for -- it cannot be reused to access a different widget's sessions, nor can it read the Widget's admin configuration.
4. The script then calls ``POST /webchat_sessions`` with the JWT to create a Session and receive the widget's ``welcome_message``.
5. Subsequent messages are sent via ``POST /webchat_messages`` using the same JWT, referencing the created ``session_id``.

.. note:: **AI Implementation Hint**

   The direct-token JWT from ``POST /auth/boot`` is scoped to a single ``resource_id`` (the Widget) and a single allowed resource type (``webchat_session``). It cannot be used to call the admin ``GET/PUT/DELETE /webchat_widgets/{id}`` endpoints -- those require a customer JWT (agent or accesskey) with Customer Admin or Customer Manager permission.

Flow trigger
------------
The configured Flow (``Widget.flow_id``) is triggered exactly once per Session, on that Session's **first inbound message** -- not on Session creation. This keeps session creation cheap (safe to call on every page load) while still giving the Flow the full conversation context once the visitor actually engages.

Session lifecycle
------------------
A Session is ``active`` from creation until it is explicitly ended (``POST /webchat_sessions/{id}/end``) or automatically ended after a period of inactivity (``Widget.session_idle_timeout``, default 1800 seconds). Once ``ended``, a Session cannot accept new messages; the visitor's next message triggers a new Session.
