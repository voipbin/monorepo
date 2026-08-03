.. _activeflow-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with activeflows, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* (To inspect an existing one) An activeflow ID (UUID). Activeflows are also created automatically when a flow is triggered (e.g., by an incoming call or ``POST /calls``). List them via ``GET /activeflows``.

.. note:: **AI Implementation Hint**

   You can create an activeflow directly via ``POST /activeflows`` by supplying either a ``flow_id`` (to run an existing flow) or an inline ``actions`` array. Activeflows are also created automatically when a flow is triggered by another resource, such as an incoming call or ``POST /calls``. To inspect one, list activeflows with ``GET /activeflows``. The ``status`` field will be ``running`` during execution and ``ended`` after completion.

Create activeflow
-------------------
Create an activeflow directly, either from an existing flow (``flow_id``) or from an inline ``actions`` array. You can also seed initial variables and register a per-activeflow webhook destination.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/activeflows?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "flow_id": "93993ae1-0408-4639-ad5f-1288aa8d4325",
            "variables": {
                "campaign_id": "summer-2026",
                "customer_name": "Jane Doe"
            },
            "webhook_uri": "https://example.com/webhooks/activeflow",
            "webhook_method": "POST"
        }'

    {
        "id": "6f18ae1c-ddf8-413b-9572-ad30574604ef",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "flow_id": "93993ae1-0408-4639-ad5f-1288aa8d4325",
        "status": "running",
        "reference_type": "api",
        "reference_id": "00000000-0000-0000-0000-000000000000",
        "reference_activeflow_id": "00000000-0000-0000-0000-000000000000",
        "on_complete_flow_id": "00000000-0000-0000-0000-000000000000",
        "webhook_uri": "https://example.com/webhooks/activeflow",
        "webhook_method": "POST",
        "current_action": {
            "id": "93ebcadb-ecae-4291-8d49-ca81a926b8b3",
            "next_id": "00000000-0000-0000-0000-000000000000",
            "type": "talk",
            "option": {
                "text": "Hello, Jane Doe."
            }
        },
        "forward_action_id": "00000000-0000-0000-0000-000000000000",
        "executed_actions": [],
        "tm_create": "2026-01-15T09:30:00Z",
        "tm_update": "2026-01-15T09:30:00Z",
        "tm_delete": null
    }

Instead of ``flow_id``, you can provide an inline ``actions`` array to run a one-off sequence without first creating a reusable flow. Provide either ``flow_id`` or ``actions``, not both.

Get activeflow list
-------------------
Getting a list of activeflows.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/activeflows?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "6f18ae1c-ddf8-413b-9572-ad30574604ef",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "flow_id": "93993ae1-0408-4639-ad5f-1288aa8d4325",
                "status": "ended",
                "reference_type": "call",
                "reference_id": "fd581a20-2606-47fd-a7e8-6bba7c294170",
                "reference_activeflow_id": "00000000-0000-0000-0000-000000000000",
                "on_complete_flow_id": "00000000-0000-0000-0000-000000000000",
                "current_action": {
                    "id": "93ebcadb-ecae-4291-8d49-ca81a926b8b3",
                    "next_id": "00000000-0000-0000-0000-000000000000",
                    "type": "digits_receive",
                    "option": {
                        "length": 1,
                        "duration": 5000
                    }
                },
                "forward_action_id": "00000000-0000-0000-0000-000000000000",
                "executed_actions": [],
                "tm_create": "2023-04-06T14:53:12Z",
                "tm_update": "2023-04-06T14:54:24Z",
                "tm_delete": null
            },
            ...
        ],
        "next_page_token": "2023-04-02T13:43:30Z"
    }

Get activeflow details
-----------------------
Getting the details of a specific activeflow by its ID.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/activeflows/6f18ae1c-ddf8-413b-9572-ad30574604ef?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "6f18ae1c-ddf8-413b-9572-ad30574604ef",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "flow_id": "93993ae1-0408-4639-ad5f-1288aa8d4325",
        "status": "running",
        "reference_type": "call",
        "reference_id": "fd581a20-2606-47fd-a7e8-6bba7c294170",
        "reference_activeflow_id": "00000000-0000-0000-0000-000000000000",
        "on_complete_flow_id": "00000000-0000-0000-0000-000000000000",
        "current_action": {
            "id": "93ebcadb-ecae-4291-8d49-ca81a926b8b3",
            "next_id": "00000000-0000-0000-0000-000000000000",
            "type": "digits_receive",
            "option": {
                "length": 1,
                "duration": 5000
            }
        },
        "forward_action_id": "00000000-0000-0000-0000-000000000000",
        "executed_actions": [],
        "tm_create": "2023-04-06T14:53:12Z",
        "tm_update": "2023-04-06T14:54:24Z",
        "tm_delete": null
    }

Stop activeflow
-------------------
Stop the activeflow.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/activeflows/1cb0566c-6aa5-45fd-beb7-e71a968075ea/stop?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "1cb0566c-6aa5-45fd-beb7-e71a968075ea",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "flow_id": "93993ae1-0408-4639-ad5f-1288aa8d4325",
        "status": "ended",
        "reference_type": "call",
        "reference_id": "cd40b5f5-dafc-43e6-9b70-38edc1155a0f",
        "reference_activeflow_id": "00000000-0000-0000-0000-000000000000",
        "on_complete_flow_id": "00000000-0000-0000-0000-000000000000",
        "current_action": {
            "id": "f9720d64-a8a8-11ed-8853-3f29a447aac1",
            "next_id": "00000000-0000-0000-0000-000000000000",
            "type": "talk",
            "option": {
                "text": "Hello. Welcome to the VoIPBIN service. Please select a service. For simple talk, press 1. For simple transcribe, press 2. For queue join, press 3. For voicemail, press 4. For conference, press 5. For chatbot talk, press 6. To contact support, press 0.",
                "language": "en-US",
                "digits_handle": "next"
            }
        },
        "forward_action_id": "00000000-0000-0000-0000-000000000000",
        "executed_actions": [],
        "tm_create": "2023-04-07T17:23:33Z",
        "tm_update": "2023-04-07T17:23:52Z",
        "tm_delete": null
    }

Delete activeflow
-------------------
Delete an activeflow by its ID. This removes the activeflow record; it does not stop a still-running activeflow (use the stop endpoint above for that first).

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/activeflows/1cb0566c-6aa5-45fd-beb7-e71a968075ea?token=<YOUR_AUTH_TOKEN>'

    -> 204 No Content

