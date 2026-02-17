.. _quickstart_call:

Call
====
Make an outbound voice call using the VoIPBIN API. You can either define actions inline or reference an existing flow.

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart_authentication>`.
* A source phone number in E.164 format (e.g., ``+15551234567``). Must be a number owned by your VoIPBIN account. Obtain available numbers via ``GET /numbers``.
* A destination phone number in E.164 format (e.g., ``+15559876543``).
* (Optional) A flow ID (UUID). Create via ``POST /flows`` or obtain from ``GET /flows``.

.. note:: **AI Implementation Hint**

   Phone numbers must be in E.164 format: ``+`` followed by country code and number, no dashes or spaces (e.g., ``+15551234567``, ``+821012345678``). If the user provides a local format like ``010-1234-5678``, normalize it to E.164 before calling this API. The ``source`` number must be a VoIPBIN-owned number — using an unowned number will result in a ``400 Bad Request``.

Make a call with inline actions
-------------------------------
This example initiates a call and plays a text-to-speech message to the recipient:

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/calls?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "<your-source-number>"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "<your-destination-number>"
                }
            ],
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "Hello. This is a VoIPBIN test call. Thank you, bye.",
                        "language": "en-US"
                    }
                }
            ]
        }'

The response includes the call details with ``"status": "dialing"``:

.. code::

    [
        {
            "id": "e2a65df2-4e50-4e37-8628-df07b3cec579",
            "flow_id": "6cbaa351-b112-452d-84c2-01488671013d",
            "source": {
                "type": "tel",
                "target": "<your-source-number>"
            },
            "destination": {
                "type": "tel",
                "target": "<your-destination-number>"
            },
            "status": "dialing",
            "direction": "outgoing",
            ...
        }
    ]

Make a call with an existing flow
---------------------------------
If you have already created a flow, you can reference it by ``flow_id`` instead of defining actions inline:

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/calls?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "<your-source-number>"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "<your-destination-number>"
                }
            ],
            "flow_id": "<your-flow-id>"
        }'

.. note:: **AI Implementation Hint**

   The ``flow_id`` (UUID) must reference an existing flow. Obtain one from the ``id`` field of ``GET /flows`` or create one via ``POST /flows``. If the flow does not exist, the API returns ``404 Not Found``.

For more details on flows, see the :ref:`Flow tutorial <flow-main>`.
