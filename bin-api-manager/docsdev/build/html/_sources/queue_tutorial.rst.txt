.. _queue-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with queues, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* At least one agent with ``status: available`` belonging to the same customer as the queue. Create agents via ``POST /agents``.
* (Optional) Tag IDs (UUID) if you want to organize queues/agents by skill for your own reporting -- create tags via ``POST /tags``. Note that tags are **not** currently used to select which agent receives a call; see the Known Limitation in :ref:`Queue Overview <queue-overview>`.
* (Optional) A wait flow ID (UUID) referencing a flow that defines what callers hear while waiting. Create one via ``POST /flows`` with actions like ``talk`` (text-to-speech announcements) and ``sleep`` (pause between announcements).

.. note:: **AI Implementation Hint**

   Queues require at least one available agent belonging to the same customer to route calls. If you create a queue with no available agents for that customer, calls will wait indefinitely (or until ``wait_timeout``). Assigning ``tag_ids`` to a queue does **not** restrict which agent receives a call -- any available agent of the customer is eligible regardless of tags (see the Known Limitation in :ref:`Queue Overview <queue-overview>`).

Create a new queue
------------------
Create a new queue.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/queues?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test queue",
        "detail": "test queue detail",
        "routing_method": "random",
        "tag_ids": [
            "d7450dda-21e0-4611-b09a-8d771c50a5e6"
        ],
        "wait_flow_id": "00000000-0000-0000-0000-000000000000",
        "wait_timeout": 100000,
        "service_timeout": 10000000
    }'

    {
        "id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "customer_id": "7a1b2c3d-4e5f-6789-abcd-ef0123456789",
        "name": "test queue",
        "detail": "test queue detail",
        "routing_method": "random",
        "tag_ids": [
            "d7450dda-21e0-4611-b09a-8d771c50a5e6"
        ],
        "wait_flow_id": "00000000-0000-0000-0000-000000000000",
        "wait_timeout": 100000,
        "service_timeout": 10000000,
        "wait_queuecall_ids": [],
        "service_queuecall_ids": [],
        "total_incoming_count": 0,
        "total_serviced_count": 0,
        "total_abandoned_count": 0,
        "tm_create": "2021-12-24 06:33:10.556226",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. note:: **AI Implementation Hint**

   The ``tag_ids`` field must contain valid tag UUIDs obtained from ``GET /tags``. The ``wait_flow_id`` (UUID) references a flow created via ``POST /flows`` that defines what callers hear while waiting -- use actions like ``talk`` for announcements and ``sleep`` for pauses within that flow. Set to ``00000000-0000-0000-0000-000000000000`` if no wait flow is needed. Timeout values (``wait_timeout``, ``service_timeout``) are in **milliseconds**: ``100000`` = 100 seconds, ``10000000`` = ~2.8 hours.

Get a queue
-----------
Get the details of a queue by its ID.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/queues/99bf739a-932f-433c-b1bf-103d33d7e9bb?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "customer_id": "7a1b2c3d-4e5f-6789-abcd-ef0123456789",
        "name": "test queue",
        "detail": "test queue detail",
        "routing_method": "random",
        "tag_ids": [
            "d7450dda-21e0-4611-b09a-8d771c50a5e6"
        ],
        "wait_flow_id": "00000000-0000-0000-0000-000000000000",
        "wait_timeout": 100000,
        "service_timeout": 10000000,
        "wait_queuecall_ids": [],
        "service_queuecall_ids": [],
        "direct_hash": "",
        "total_incoming_count": 0,
        "total_serviced_count": 0,
        "total_abandoned_count": 0,
        "tm_create": "2021-12-24 06:33:10.556226",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Update a queue
--------------
Update a queue's basic settings (name, detail, routing method, tags, wait flow, timeouts). All fields shown are required in the request body.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/queues/99bf739a-932f-433c-b1bf-103d33d7e9bb?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "updated queue",
            "detail": "updated queue detail",
            "routing_method": "random",
            "tag_ids": [
                "d7450dda-21e0-4611-b09a-8d771c50a5e6"
            ],
            "wait_flow_id": "00000000-0000-0000-0000-000000000000",
            "wait_timeout": 200000,
            "service_timeout": 0
        }'

Update a queue's routing method
--------------------------------
Update only the queue's routing method.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/queues/99bf739a-932f-433c-b1bf-103d33d7e9bb/routing_method?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "routing_method": "random"
        }'

.. note:: **AI Implementation Hint**

   ``"random"`` is currently the only non-empty routing method value supported. See :ref:`Routing Method <queue-struct-queue-routing-method>`.

Update a queue's tag IDs
-------------------------
Replace the queue's ``tag_ids`` list. These are stored and returned by the API, but -- as noted above -- not currently used to select which agent receives a call.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/queues/99bf739a-932f-433c-b1bf-103d33d7e9bb/tag_ids?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "tag_ids": [
                "d7450dda-21e0-4611-b09a-8d771c50a5e6",
                "b1a2c3d4-e5f6-7890-abcd-ef1234567890"
            ]
        }'

.. note:: **AI Implementation Hint**

   This replaces the entire ``tag_ids`` array. To add a tag while keeping existing ones, first retrieve the current list via ``GET /queues/{id}`` and include it in the request.

Delete a queue
--------------
Delete a queue by its ID.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/queues/99bf739a-932f-433c-b1bf-103d33d7e9bb?token=<YOUR_AUTH_TOKEN>'

Regenerate direct queue hash
----------------------------
Regenerate the direct hash for a queue. This invalidates the previous SIP URI and creates a new one. If the queue has no existing direct hash, one is created automatically.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/queues/<queue-id>/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

.. code::

    {
        "id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "name": "test queue",
        "detail": "test queue detail",
        "routing_method": "random",
        "direct_hash": "direct.e9f0a1b2c3d4",
        ...
    }

.. note:: **AI Implementation Hint**

   This endpoint requires no request body. The ``direct_hash`` in the response is the new hash (already prefixed with ``direct.``) -- the previous hash is permanently invalidated. The direct SIP URI is formed directly from it: ``sip:<direct_hash>@sip.voipbin.net``.

Get list of queues
------------------
Gets the list of created queues.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/queues?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
                "customer_id": "7a1b2c3d-4e5f-6789-abcd-ef0123456789",
                "name": "test queue",
                "detail": "test queue detail",
                "routing_method": "random",
                "tag_ids": [
                    "d7450dda-21e0-4611-b09a-8d771c50a5e6"
                ],
                "direct_hash": "",
                "wait_flow_id": "00000000-0000-0000-0000-000000000000",
                "wait_timeout": 100000,
                "service_timeout": 10000000,
                "wait_queuecall_ids": [
                    "2eb40044-2e5e-4dae-b41e-61968e4febf9",
                    "b0aa4639-fea3-4727-8b86-44667d8f4c27",
                    "ec590f5b-6de5-477b-905b-1833dde213a0",
                    "003e8242-a0ed-4d55-9e4f-59c317c023ad",
                    "467fdfc2-fa2b-40f6-82cf-18dcb4c952c3",
                    "2973648e-5989-4f75-9bda-b356d7a470dc"
                ],
                "service_queuecall_ids": [],
                "total_incoming_count": 76,
                "total_serviced_count": 70,
                "total_abandoned_count": 21,
                "tm_create": "2021-12-24 06:33:10.556226",
                "tm_update": "2022-02-20 05:30:31.067539",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2021-12-24 06:33:10.556226"
    }

Get list of queuecalls
-----------------------
Gets the list of queuecalls (calls currently in, or having passed through, a queue).

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/queuecalls?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
                "customer_id": "5e4a0680-804e-11ec-98a7-2fea5968d85b",
                "reference_type": "call",
                "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
                "status": "waiting",
                "service_agent_id": "00000000-0000-0000-0000-000000000000",
                "duration_waiting": 0,
                "duration_service": 0,
                "tm_create": "2022-03-29 15:07:46.111715",
                "tm_service": "9999-01-01 00:00:00.000000",
                "tm_update": "2022-03-29 15:07:46.111715",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-03-29 15:07:46.111715"
    }

Get a queuecall
----------------
Get the details of a queuecall by its ID.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/queuecalls/c7c1e226-8c86-4b43-9606-2d5bb2059a09?token=<YOUR_AUTH_TOKEN>'

Kick a queuecall from the queue
---------------------------------
Force-remove a queuecall from the queue by its queuecall ID. The queuecall's status becomes ``abandoned``.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/queuecalls/c7c1e226-8c86-4b43-9606-2d5bb2059a09/kick?token=<YOUR_AUTH_TOKEN>'

Kick a queuecall by reference ID
-----------------------------------
Force-remove a queuecall from the queue using the ID of the referenced resource (e.g., the call ID) instead of the queuecall ID.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/queuecalls/reference_id/1fe1356f-3f7f-4ff9-9d33-08136b38f506/kick?token=<YOUR_AUTH_TOKEN>'

Delete a queuecall
--------------------
Delete a queuecall by its ID.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/queuecalls/c7c1e226-8c86-4b43-9606-2d5bb2059a09?token=<YOUR_AUTH_TOKEN>'

