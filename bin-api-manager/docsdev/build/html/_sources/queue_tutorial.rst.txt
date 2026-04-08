.. _queue-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with queues, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* At least one tag ID (UUID). Create tags via ``POST /tags`` or obtain existing ones from ``GET /tags``. Tags define the skills required for agents to receive calls from this queue.
* At least one agent configured with matching tags. Create agents via ``POST /agents`` and assign tags via ``PUT /agents/{id}``. Verify agent tags with ``GET /agents/{id}``.
* (Optional) A wait flow ID (UUID) referencing a flow that defines what callers hear while waiting. Create one via ``POST /flows`` with actions like ``talk`` (text-to-speech announcements) and ``sleep`` (pause between announcements).

.. note:: **AI Implementation Hint**

   Queues require tags and agents to function. If you create a queue without any agents having matching tags, calls will wait indefinitely (or until ``wait_timeout``). Always verify that at least one agent has all the tags listed in the queue's ``tag_ids`` before routing calls to the queue.

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

   This endpoint requires no request body. The ``direct_hash`` in the response is the new hash -- the previous hash is permanently invalidated. The direct SIP URI format is ``sip:direct.<hash>@sip.voipbin.net``.

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

