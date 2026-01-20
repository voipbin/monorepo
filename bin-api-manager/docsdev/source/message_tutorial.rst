.. _message-tutorial:

Tutorial
========

Send a message
--------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/messages?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+15559876543"
        },
        "destinations": [
            {
                "type": "tel",
                "target":"+31616818985"
            }
        ],
        "text": "hello, this is test message."
    }'

Get list of messages
--------------------

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/messages?token=<YOUR_AUTH_TOKEN>&page_size=10'

    {
    "result": [
        {
            "id": "a5d2114a-8e84-48cd-8bb2-c406eeb08cd1",
            "type": "sms",
            "source": {
                "type": "tel",
                "target": "+15551234567",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "targets": [
                {
                    "destination": {
                        "type": "tel",
                        "target": "+15559876543",
                        "target_name": "",
                        "name": "",
                        "detail": ""
                    },
                    "status": "sent",
                    "parts": 1,
                    "tm_update": "2022-03-13 15:11:06.497184184"
                }
            ],
            "text": "Hello, this is test message.",
            "direction": "outbound",
            "tm_create": "2022-03-13 15:11:05.235717",
            "tm_update": "2022-03-13 15:11:06.497278",
            "tm_delete": "9999-01-01 00:00:00.000000"
        },
        ...
    ]
