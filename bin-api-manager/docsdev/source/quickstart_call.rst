.. _quickstart_call:

Call
====
Make an outbound voice call using the VoIPBin API. You can either define actions inline or reference an existing flow.

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
                        "text": "Hello. This is a VoIPBin test call. Thank you, bye.",
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

For more details on flows, see the :ref:`Flow tutorial <flow-main>`.
