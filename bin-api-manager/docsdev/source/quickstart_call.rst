.. _quickstart_call:

Call
====
In this Quickstart, you'll learn how to make an outbound voice call.

Make your first voice call with manual actions
-----------------------------------------------
Use the VoIPBIN API to initiate an outbound telephone call from your VoIPBIN account.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=your-voipbin-token' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+82XXXXXXXX"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+82XXXXXXXX"
                }
            ],
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "Hello. This is voipbin test. The voipbin provides ready to go CPaaS service. Thank you, bye.",
                        "gender": "female",
                        "language": "en-US"
                    }
                }
            ]
        }'

    [
        {
            "id": "e2a65df2-4e50-4e37-8628-df07b3cec579",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "flow_id": "6cbaa351-b112-452d-84c2-01488671013d",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+821028286521",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+821021656521",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "status": "dialing",
            "action": {
                "id": "00000000-0000-0000-0000-000000000001",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": ""
            },
            "direction": "outgoing",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_progressing": "9999-01-01 00:00:00.000000",
            "tm_ringing": "9999-01-01 00:00:00.000000",
            "tm_hangup": "9999-01-01 00:00:00.000000",
            "tm_create": "2023-03-28 12:00:05.248732",
            "tm_update": "9999-01-01 00:00:00.000000",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    ]

Make your first voice call with existing flow
---------------------------------------------
Use the VoIPBIN API to initiate an outbound telephone call with existing flow.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=your-voipbin-token' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "<your source number>"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "<your destination number>"
                }
            ],
            "flow_id": "ed95a3c4-22d4-11ee-add7-8742a741581e",
        }'

    [
        {
            "id": "e2a65df2-4e50-4e37-8628-df07b3cec579",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "flow_id": "ed95a3c4-22d4-11ee-add7-8742a741581e",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+821028286521",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+821021656521",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "status": "dialing",
            "action": {
                "id": "00000000-0000-0000-0000-000000000001",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": ""
            },
            "direction": "outgoing",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_progressing": "9999-01-01 00:00:00.000000",
            "tm_ringing": "9999-01-01 00:00:00.000000",
            "tm_hangup": "9999-01-01 00:00:00.000000",
            "tm_create": "2023-03-28 12:00:05.248732",
            "tm_update": "9999-01-01 00:00:00.000000",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    ]
