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
        "username": "test2",
        "name": "test tag",
        "detail": "test tag example",
        "ring_method": "ringall",
        "status": "offline",
        "permission": 0,
        "tag_ids": ["d7450dda-21e0-4611-b09a-8d771c50a5e6"],
        "addresses": [],
        "tm_create": "2022-10-22 16:16:16.874761",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


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
