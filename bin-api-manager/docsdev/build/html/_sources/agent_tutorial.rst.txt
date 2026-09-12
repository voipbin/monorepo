.. _agent-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before managing agents, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* (For tag assignment) Tag IDs (UUIDs). Create tags via ``POST /tags`` or obtain existing ones via ``GET /tags``.
* (For address assignment) Contact addresses in the correct format: E.164 for ``tel`` type (e.g., ``+15559876543``), numeric-only for ``extension`` type, or ``user@domain`` for ``sip`` type.

.. note:: **AI Implementation Hint**

   When creating an agent, the ``password`` field is required but is write-only and never returned in responses. The agent's initial ``status`` will be ``offline``. After creation, the agent must explicitly set their status to ``available`` via ``PUT /agents/{id}/status`` before they can receive calls from queues.

   ``addresses`` is optional. An agent with no addresses can still be assigned to chat/conversation work, since those are routed by agent ID rather than by dial address -- but it cannot receive voice-queue calls (``ring_method`` ``ringall``/``linear`` dials the agent's addresses directly) until at least one address is added, e.g. via ``PUT /agents/{id}/addresses``.

Create a new agent
------------------

Create a new agent.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/agents?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "username": "test2",
            "password": "test2",
            "name": "test tag",
            "detail": "test tag example",
            "ring_method": "ringall",
            "permission": 0,
            "tag_ids": ["d7450dda-21e0-4611-b09a-8d771c50a5e6"]
        }'

    {
        "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "username": "test2",
        "name": "test tag",
        "detail": "test tag example",
        "ring_method": "ringall",
        "status": "offline",
        "permission": 0,
        "tag_ids": ["d7450dda-21e0-4611-b09a-8d771c50a5e6"],
        "addresses": [],
        "direct_hash": "",
        "tm_create": "2022-10-22T16:16:16Z",
        "tm_update": null,
        "tm_delete": null
    }


Get list of agents
-------------------
Gets the list of agents. Supports filtering by ``status`` and by ``tag_ids`` (comma-separated tag id list -- only agents sharing at least one of the given tags are returned; the parameter must not be empty when present).

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/agents?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567890",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "username": "test2",
                "name": "test tag",
                "detail": "test tag example",
                "ring_method": "ringall",
                "status": "offline",
                "permission": 0,
                "tag_ids": ["d7450dda-21e0-4611-b09a-8d771c50a5e6"],
                "addresses": [],
                "direct_hash": "",
                "tm_create": "2022-10-22T16:16:16Z",
                "tm_update": null,
                "tm_delete": null
            }
        ],
        "next_page_token": "2022-10-22T16:16:16Z"
    }

Get an agent
------------
Get the details of an agent by its ID.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/agents/f1a2b3c4-d5e6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

Update an agent
----------------
Update an agent's name, detail, and ring method.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/f1a2b3c4-d5e6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "updated name",
            "detail": "updated detail",
            "ring_method": "linear"
        }'

Update agent's tag IDs
------------------------
Replace the full list of tag IDs assigned to the agent.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/f1a2b3c4-d5e6-7890-abcd-ef1234567890/tag_ids?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "tag_ids": [
                "d7450dda-21e0-4611-b09a-8d771c50a5e6",
                "b1a2c3d4-e5f6-7890-abcd-ef1234567890"
            ]
        }'

Update agent's permission
----------------------------
Update an agent's permission bitmask. See :ref:`Permission <agent-struct-agent-permission>` for valid values.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/f1a2b3c4-d5e6-7890-abcd-ef1234567890/permission?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "permission": 32
        }'

.. note:: **AI Implementation Hint**

   Setting a project-level permission bit (part of the ``0x000F`` / ``15`` project mask) requires the caller to already hold project-level permission. Customer-level permission changes (agent/admin/manager bits) require the caller to have customer admin or manager permission.

Update agent's password
--------------------------
Update an agent's password.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/f1a2b3c4-d5e6-7890-abcd-ef1234567890/password?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "password": "new_secure_password"
        }'

Delete an agent
----------------
Delete an agent by its ID.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/agents/f1a2b3c4-d5e6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

Update agent's status
---------------------
Update agent's status to the available.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b/status?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "status": "available"
        }'

Update agent's addresses
------------------------
Update agent's addresses.

.. note:: **AI Implementation Hint**

   The ``PUT /agents/{id}/addresses`` endpoint replaces all addresses for the agent. To add a new address while keeping existing ones, first retrieve the current addresses via ``GET /agents/{id}``, then include all desired addresses in the update request. Phone numbers in ``tel`` type addresses must be in E.164 format (e.g., ``+15559876543``).

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b/addresses?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "addresses": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ]
        }'

Regenerate direct agent hash
------------------------------

Regenerate the direct hash for an agent. This invalidates the previous SIP URI and creates a new one. If the agent has no existing direct hash, one is created automatically.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/agents/eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "username": "test2",
        "name": "test tag",
        "detail": "test tag example",
        "ring_method": "ringall",
        "status": "offline",
        "permission": 0,
        "tag_ids": [],
        "addresses": [],
        "direct_hash": "direct.e9f0a1b2c3d4",
        "tm_create": "2022-10-22T16:16:16Z",
        "tm_update": "2022-10-22T16:20:00Z",
        "tm_delete": null
    }

.. note:: **AI Implementation Hint**

   This endpoint requires no request body. The ``direct_hash`` in the response is the new hash (already prefixed with ``direct.``) — the previous hash is permanently invalidated. The direct SIP URI is formed directly from it: ``sip:<direct_hash>@sip.voipbin.net``.
