.. _conversation-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with conversations, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A conversation already created by an incoming message (SMS/MMS via ``message`` type, or LINE via ``line`` type). Conversations are auto-created when messages arrive -- there is no ``POST /conversations`` endpoint.
* (Optional) A conversation account configured for the messaging channel (LINE credentials, etc.). Manage accounts via ``GET /conversation_accounts``.

.. note:: **AI Implementation Hint**

   Conversations are auto-created when an inbound message arrives on a configured channel (SMS/MMS or LINE). There is no ``POST /conversations`` endpoint to create conversations manually. The ``type`` field indicates the channel: ``message`` for SMS/MMS or ``line`` for LINE. Messages sent within a conversation incur per-channel delivery costs. When sending a message to a conversation, pass the conversation UUID in the URL path: ``POST /conversations/{id}/messages``.

Get list of conversations
-------------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "owner_type": "agent",
                "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
                "account_id": "c5d6e7f8-a9b0-1234-cdef-567890abcdef",
                "name": "conversation",
                "detail": "conversation detail",
                "type": "line",
                "dialog_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "self": {
                    "type": "line",
                    "target": "",
                    "target_name": "me",
                    "name": "",
                    "detail": ""
                },
                "peer": {
                    "type": "line",
                    "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                    "target_name": "Unknown",
                    "name": "",
                    "detail": ""
                },
                "tm_create": "2022-06-17 06:06:14.446158",
                "tm_update": "2022-06-17 06:06:14.446167",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-06-17 06:06:14.446158"
    }

Get detail of conversation
--------------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "account_id": "c5d6e7f8-a9b0-1234-cdef-567890abcdef",
        "name": "conversation",
        "detail": "conversation detail",
        "type": "line",
        "dialog_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "self": {
            "type": "line",
            "target": "",
            "target_name": "me",
            "name": "",
            "detail": ""
        },
        "peer": {
            "type": "line",
            "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
            "target_name": "Unknown",
            "name": "",
            "detail": ""
        },
        "tm_create": "2022-06-17 06:06:14.446158",
        "tm_update": "2022-06-17 06:06:14.446167",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Send a message to the conversation
----------------------------------

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/messages?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "text": "Hello, this is a test message. Thank you for your time.",
            "medias": []
        }'

    {
        "id": "0c8f23cb-e878-49bf-b69e-03f59252f217",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "direction": "outgoing",
        "status": "progressing",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "source": {
            "type": "line",
            "target": ""
        },
        "destination": {
            "type": "line",
            "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
        },
        "text": "Hello, this is a test message. Thank you for your time.",
        "medias": [],
        "tm_create": "2022-06-20 03:07:11.372307",
        "tm_update": "2022-06-20 03:07:11.372315",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Get list of conversation messages
---------------------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/messages?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "0c8f23cb-e878-49bf-b69e-03f59252f217",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
                "direction": "outgoing",
                "status": "done",
                "reference_type": "line",
                "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "source": {
                    "type": "line",
                    "target": ""
                },
                "destination": {
                    "type": "line",
                    "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
                },
                "text": "Hello, this is a test message. Thank you for your time.",
                "medias": [],
                "tm_create": "2022-06-20 03:07:11.372307",
                "tm_update": "2022-06-20 03:07:11.372315",
                "tm_delete": "9999-01-01 00:00:00.000000"
            },
            ...
        ],
        "next_page_token": "2022-06-17 06:06:14.948432"
    }


Assigning a conversation to an agent (walkthrough)
--------------------------------------------------

This walkthrough shows the full lifecycle of assigning an inbound conversation to a specific agent so the agent can handle it manually, then unassigning it so the registered flow resumes.

For the conceptual model and permission rules, see :ref:`Assigning a Conversation to an Agent <conversation-overview-assigning-conversation-to-agent>`.

Prerequisites for this walkthrough
++++++++++++++++++++++++++++++++++

* An admin or manager auth token (``<ADMIN_TOKEN>`` below) — required for the initial assignment.
* The agent's auth token (``<AGENT_TOKEN>`` below) — used by the agent to reply and to self-unassign.
* The agent UUID. Obtain it from the ``id`` field of ``GET https://api.voipbin.net/v1.0/agents``. The examples below use the literal value ``eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b``; substitute your own.
* An existing conversation UUID. Obtain it from ``GET https://api.voipbin.net/v1.0/conversations``. The examples below use the literal value ``a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f``; substitute your own.

Replace ``<ADMIN_TOKEN>`` and ``<AGENT_TOKEN>`` with the corresponding auth tokens in each request.

Step 1. Admin assigns the conversation to the agent
+++++++++++++++++++++++++++++++++++++++++++++++++++

The admin (or manager) sends a ``PUT`` with only the ``owner_id`` field. The server derives ``owner_type`` from ``owner_id`` and ignores any caller-supplied ``owner_type``.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f?token=<ADMIN_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"
        }'

    {
        "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "account_id": "c5d6e7f8-a9b0-1234-cdef-567890abcdef",
        "name": "conversation",
        "detail": "conversation detail",
        "type": "line",
        "dialog_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "self": { "type": "line", "target": "", "target_name": "me", "name": "", "detail": "" },
        "peer": { "type": "line", "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f", "target_name": "Unknown", "name": "", "detail": "" },
        "tm_create": "2022-06-17 06:06:14.446158",
        "tm_update": "2026-04-30 09:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Step 2. Agent receives a ``conversation_updated`` webhook
+++++++++++++++++++++++++++++++++++++++++++++++++++++++++

VoIPBIN delivers a webhook to your application reflecting the new owner. The webhook body is the conversation's ``WebhookMessage``, with the assignment fields populated.

.. code::

    {
        "type": "conversation_updated",
        "data": {
            "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "owner_type": "agent",
            "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "account_id": "c5d6e7f8-a9b0-1234-cdef-567890abcdef",
            "name": "conversation",
            "detail": "conversation detail",
            "type": "line",
            "dialog_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
            "self": { "type": "line", "target": "", "target_name": "me", "name": "", "detail": "" },
            "peer": { "type": "line", "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f", "target_name": "Unknown", "name": "", "detail": "" },
            "tm_create": "2022-06-17 06:06:14.446158",
            "tm_update": "2026-04-30 09:00:00.000000",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

Step 3. Inbound message arrives on the assigned conversation
++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

While the conversation is assigned, the next inbound message from the peer does **not** trigger the registered flow. No new activeflow is created. The assigned agent receives the standard ``conversation_message_created`` webhook (with ``direction: "incoming"``) and is expected to reply via the API.

You can verify this by listing recent activeflows for the customer (``GET https://api.voipbin.net/v1.0/activeflows``) before and after the inbound message arrives — there will be no new activeflow tied to the inbound message.

.. note:: **AI Implementation Hint**

   Already-running activeflows are unaffected by assignment. If a flow was already running on this conversation when assignment happened, it will continue to run to completion. Only **new** inbound messages that arrive while the conversation is assigned will skip the flow trigger.

Step 4. Agent replies to the conversation
+++++++++++++++++++++++++++++++++++++++++

.. note:: **Permission scope**

   The message-send endpoint (``POST /conversations/{id}/messages``) currently
   requires admin or manager permission. A per-agent JWT (``<AGENT_TOKEN>``)
   does **not** have permission to send messages — it returns ``403 Forbidden``.
   In practice the agent reply is sent from an admin/manager-scoped session
   (for example, the agent web app at https://talk.voipbin.net authenticated
   as an admin or manager user, which fans out replies on behalf of the agent).
   Use ``<ADMIN_TOKEN>`` (or any admin/manager-scoped token) for this step.

.. note:: **Future work (out of scope for this release)**

   ``ConversationMessageSend`` does not yet have a self-reply carve-out for the
   owning agent — the owning agent cannot post messages with their own JWT.
   A follow-up could mirror the ``POST /conversations/{id}/unassign`` pattern
   and allow the owning agent to send messages on a conversation they own.
   This is intentionally out of scope for the current assignment release.

The agent reply uses the standard message-send API (no special endpoint required for assigned conversations).

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/messages?token=<ADMIN_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "text": "Hi, this is the support agent. How can I help you today?",
            "medias": []
        }'

    {
        "id": "0c8f23cb-e878-49bf-b69e-03f59252f217",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "direction": "outgoing",
        "status": "progressing",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "source": {
            "type": "line",
            "target": ""
        },
        "destination": {
            "type": "line",
            "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
        },
        "text": "Hi, this is the support agent. How can I help you today?",
        "medias": [],
        "tm_create": "2026-04-30 09:05:00.000000",
        "tm_update": "2026-04-30 09:05:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Step 5. Agent self-unassigns when handling is complete
++++++++++++++++++++++++++++++++++++++++++++++++++++++

The owning agent unassigns themselves using the dedicated ``POST /unassign`` endpoint. No request body is required. This is the only assignment-related operation an owning agent is permitted to perform.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/unassign?token=<AGENT_TOKEN>'

    {
        "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "",
        "owner_id": "00000000-0000-0000-0000-000000000000",
        "account_id": "c5d6e7f8-a9b0-1234-cdef-567890abcdef",
        "name": "conversation",
        "detail": "conversation detail",
        "type": "line",
        "dialog_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "self": { "type": "line", "target": "", "target_name": "me", "name": "", "detail": "" },
        "peer": { "type": "line", "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f", "target_name": "Unknown", "name": "", "detail": "" },
        "tm_create": "2022-06-17 06:06:14.446158",
        "tm_update": "2026-04-30 09:30:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. note:: **AI Implementation Hint**

   Use ``POST /v1.0/conversations/<id>/unassign`` for agent-initiated unassignment. The endpoint takes **no request body** and returns the updated conversation. Do **not** use ``PUT /conversations/<id>`` with the nil UUID — that endpoint now requires admin or manager permission and returns ``403 Forbidden`` for agent callers.

   Admin and manager callers may use either ``POST /unassign`` or ``PUT /conversations/<id>`` with ``{"owner_id": "00000000-0000-0000-0000-000000000000"}``.

   **Error responses:**

   * **403 Forbidden:** The caller is neither an admin/manager nor the current owner of the conversation.
   * **404 Not Found:** The conversation UUID does not exist or belongs to a different customer.

Step 6. Next inbound message: registered flow resumes
+++++++++++++++++++++++++++++++++++++++++++++++++++++

Once unassigned (``owner_id`` is the nil UUID and ``owner_type`` is empty), the conversation reverts to standard behavior. The next inbound message triggers the registered flow as usual, creating a fresh activeflow you can observe via ``GET https://api.voipbin.net/v1.0/activeflows``.

Listing "my conversations"
++++++++++++++++++++++++++

Agents can build a "my conversations" view by filtering on ``owner_id`` against their own agent UUID:

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations?owner_id=eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b&page_token=2026-04-30T10:00:00.000000Z' \
        --header 'Authorization: Bearer <AGENT_TOKEN>'

.. note:: **AI Implementation Hint**

   Agent callers (non-admin, non-manager) MUST set ``owner_id`` to their own agent UUID. Any other value — or omitting the parameter entirely — returns ``403 Forbidden``. Admin and manager callers have no such restriction; they may pass any ``owner_id`` or omit the filter to list all conversations for the customer.


Assignment and Unassignment Endpoint Reference
----------------------------------------------

This section summarises the four endpoints introduced or changed in this release.

PUT /v1.0/conversations/{id} (admin/manager only)
+++++++++++++++++++++++++++++++++++++++++++++++++

Update conversation details, including ownership assignment and reassignment.

* **Method:** ``PUT``
* **Path:** ``https://api.voipbin.net/v1.0/conversations/{id}``
* **Permission:** Admin or manager only. Agent callers receive ``403 Forbidden``.
* **Request body fields:**

  * ``owner_id`` (UUID, Optional): Agent UUID to assign as owner. Set to ``00000000-0000-0000-0000-000000000000`` to unassign.
  * ``owner_type``: Ignored. The server always derives this from ``owner_id``.
  * ``name`` (String, Optional): Human-readable conversation name.
  * ``detail`` (String, Optional): Free-form detail field.

* **Response:** The updated conversation object.

**Error responses:**

* **403 Forbidden:** Caller does not have admin or manager permission.
* **400 Bad Request:** ``owner_id`` references an agent UUID that does not exist or belongs to a different customer.

POST /v1.0/conversations/{id}/unassign (admin/manager + owning agent)
++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

Remove the current owner from a conversation. No request body required.

* **Method:** ``POST``
* **Path:** ``https://api.voipbin.net/v1.0/conversations/{id}/unassign``
* **Permission:** Admin, manager, or the current owning agent. Non-owning agents receive ``403 Forbidden``.
* **Request body:** None.
* **Response:** The updated conversation object (``owner_id`` is the nil UUID, ``owner_type`` is empty).

**Example:**

.. code::

    $ curl --location --request POST \
        'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/unassign?token=<AGENT_OR_ADMIN_TOKEN>'

**Error responses:**

* **403 Forbidden:** Caller is neither an admin/manager nor the current owner.
* **404 Not Found:** Conversation UUID does not exist or belongs to a different customer.

PUT /v1.0/service_agents/conversations/{id} (admin/manager only)
+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

Update conversation details via the service-agents surface. Identical semantics to ``PUT /v1.0/conversations/{id}`` but available under the ``/service_agents/`` prefix.

* **Method:** ``PUT``
* **Path:** ``https://api.voipbin.net/v1.0/service_agents/conversations/{id}``
* **Permission:** Admin or manager only.
* **Request body fields:**

  * ``owner_id`` (UUID, Optional): Agent UUID to assign as owner, or the nil UUID to unassign.
  * ``owner_type``: Ignored.
  * ``name`` (String, Optional): Human-readable conversation name.
  * ``detail`` (String, Optional): Free-form detail field.

* **Response:** The updated conversation object.

**Error responses:**

* **403 Forbidden:** Caller does not have admin or manager permission.
* **400 Bad Request:** ``owner_id`` references an agent that does not exist or belongs to a different customer.

POST /v1.0/service_agents/conversations/{id}/unassign (admin/manager + owning agent)
+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

Remove the current owner from a conversation via the service-agents surface. Identical semantics to ``POST /v1.0/conversations/{id}/unassign``.

* **Method:** ``POST``
* **Path:** ``https://api.voipbin.net/v1.0/service_agents/conversations/{id}/unassign``
* **Permission:** Admin, manager, or the current owning agent.
* **Request body:** None.
* **Response:** The updated conversation object (``owner_id`` is the nil UUID, ``owner_type`` is empty).

**Example:**

.. code::

    $ curl --location --request POST \
        'https://api.voipbin.net/v1.0/service_agents/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/unassign?token=<AGENT_OR_ADMIN_TOKEN>'

**Error responses:**

* **403 Forbidden:** Caller is neither an admin/manager nor the current owner.
* **404 Not Found:** Conversation UUID does not exist or belongs to a different customer.

.. note:: **AI Implementation Hint**

   The ``/service_agents/conversations/`` prefix is functionally equivalent to ``/conversations/`` for these endpoints. Use the ``/conversations/`` path in typical integrations. The ``/service_agents/`` prefix is provided for service-level tooling and admin workflows that operate on behalf of agents.
