.. _activeflow-tutorial:

Tutorial
========

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
                "tm_create": "2023-04-06 14:53:12.569073",
                "tm_update": "2023-04-06 14:54:24.652558",
                "tm_delete": "9999-01-01 00:00:00.000000"
            },
            ...
        ],
        "next_page_token": "2023-04-02 13:43:30.576077"
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
        "tm_create": "2023-04-07 17:23:33.665475",
        "tm_update": "2023-04-07 17:23:52.561527",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


