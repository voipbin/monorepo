.. _message-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before sending messages, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A source phone number in E.164 format (e.g., ``+15551234567``). This must be a number you own. Obtain your numbers via ``GET /numbers``.
* A destination phone number in E.164 format (e.g., ``+15559876543``).

.. note:: **AI Implementation Hint**

   Sending messages incurs charges per message segment. All phone numbers must be in E.164 format: start with ``+``, followed by country code and number, no dashes or spaces. The ``source`` and ``destinations`` fields use the :ref:`Address <common-struct-address-address>` format with ``type`` set to ``tel``. Target delivery statuses follow this lifecycle: ``queued`` -> ``sent`` -> ``delivered``. Failures may produce ``gw_timeout``, ``dlr_timeout``, or ``failed``. Inbound messages arrive with status ``received``.

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

Send an MMS
-----------

Upload the file with the type ``media``, then send the message with the returned file id in ``medias``.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/storage_files?token=<YOUR_AUTH_TOKEN>' \
    --form 'file=@/path/to/photo.png' \
    --form 'type=media'

    {
        "id": "3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
        "type": "media",
        "filename": "photo.png",
        ...
    }

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/messages?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+155****6543"
        },
        "destinations": [
            {
                "type": "tel",
                "target": "+155****8985"
            }
        ],
        "text": "Here is the photo.",
        "medias": ["3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"]
    }'

The response has ``"type": "mms"`` and the ``medias`` list. See :ref:`Sending an MMS <message-overview-mms>` for the limits and the errors.

Get list of messages
--------------------

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/messages?token=<YOUR_AUTH_TOKEN>&page_size=10'

    {
    "result": [
        {
            "id": "a5d2114a-8e84-48cd-8bb2-c406eeb08cd1",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
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
                    "tm_update": "2022-03-13 15:11:06.497184"
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
