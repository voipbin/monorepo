.. _quickstart_queue:

Queue
=====
In this Quickstart, you'll learn how to set the queue.

Create your first queue
--------------------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/queues?token=your-voipbin-token' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "test queue",
            "detail": "test queue detail",
            "routing_method": "random",
            "tag_ids": ["d7450dda-21e0-4611-b09a-8d771c50a5e6"],
            "wait_actions": [
                {
                    "type":"talk",
                    "option": {
                        "text": "Hello. This is test queue. Please wait.",
                        "gender": "female",
                        "language": "en-US"
                    }
                },
                {
                    "type": "sleep",
                    "option": {
                        "duration": 1000
                    }
                }
            ],
            "timeout_wait": 100000,
            "timeout_service": 10000000
        }'

    {
        "id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "test queue",
        "detail": "test queue detail",
        "routing_method": "random",
        "tag_ids": [
            "d7450dda-21e0-4611-b09a-8d771c50a5e6"
        ],
        "wait_actions": [
            {
                "id": "00000000-0000-0000-0000-000000000000",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "talk",
                "option": {
                    "text": "Hello. This is test queue. Please wait.",
                    "gender": "female",
                    "language": "en-US"
                }
            },
            {
                "id": "00000000-0000-0000-0000-000000000000",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "sleep",
                "option": {
                    "duration": 1000
                }
            }
        ],
        "wait_timeout": 100000,
        "service_timeout": 10000000,
        "wait_queuecall_ids": [
            "65b3f8c3-ce8e-4a5d-ae13-598aa2889377"
        ],
        "service_queuecall_ids": [],
        "total_incoming_count": 228,
        "total_serviced_count": 169,
        "total_abandoned_count": 99,
        "tm_create": "2021-12-24 06:33:10.556226",
        "tm_update": "2023-03-07 12:39:54.664143",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
