.. _queue-tutorial:

Tutorial
========


Create a new queue
------------------
Create a new queue

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/queues?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
    --data-raw '{
        "name": "test queue",
        "detail": "test queue detail",
        "routing_method": "random",
        "tag_ids": [
            "d7450dda-21e0-4611-b09a-8d771c50a5e6"
        ],
        "wait_actions": [
            {
                "type": "talk",
                "option": {
                    "text": "All of the agents are busy. Thank you for your waiting.",
                    "language": "en-US"
                }
            },
            {
                "type": "sleep",
                "option": {
                    "duration": 10000
                }
            }

        ],
        "timeout_wait": 100000,
        "timeout_service": 10000000
    }'

Get list of queues
------------------
Gets the list of created queues.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/queues?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
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
                            "language": "en-US"
                        }
                    }
                ],
                "wait_timeout": 100000,
                "service_timeout": 10000000,
                "wait_queue_call_ids": [
                    "2eb40044-2e5e-4dae-b41e-61968e4febf9",
                    "b0aa4639-fea3-4727-8b86-44667d8f4c27",
                    "ec590f5b-6de5-477b-905b-1833dde213a0",
                    "003e8242-a0ed-4d55-9e4f-59c317c023ad",
                    "467fdfc2-fa2b-40f6-82cf-18dcb4c952c3",
                    "2973648e-5989-4f75-9bda-b356d7a470dc"
                ],
                "service_queue_call_ids": [],
                "total_incoming_count": 76,
                "total_serviced_count": 70,
                "total_abandoned_count": 21,
                "total_waittime": 338789,
                "total_service_duration": 4050690,
                "tm_create": "2021-12-24 06:33:10.556226",
                "tm_update": "2022-02-20 05:30:31.067539",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2021-12-24 06:33:10.556226"
    }

