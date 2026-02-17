.. _conversation-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with conversations, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A ``reference_type`` value indicating the channel for the conversation (e.g., ``message`` for SMS/MMS, ``line`` for Line). See :ref:`Reference type <conversation-struct-conversation-reference_type>`.
* (Optional) Participant addresses in the appropriate format for the channel (E.164 phone numbers for SMS, Line user IDs for Line).

.. note:: **AI Implementation Hint**

   Conversations are free to create, but messages sent within them incur per-channel delivery costs (SMS, email, etc.). When sending a message to a conversation via ``POST /conversations/{id}/messages``, the conversation ID is passed in the URL path, not the request body.

Setup the conversation
----------------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/conversations/setup?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "reference_type": "line"
    }'

Get list of conversations
-------------------------

Example
+++++++

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/conversations?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
            "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
            "name": "conversation",
            "detail": "conversation detail",
            "reference_type": "line",
            "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
            "participants": [
                {
                "type": "line",
                "target": "",
                "target_name": "me",
                "name": "",
                "detail": ""
                },
                {
                "type": "line",
                "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "target_name": "Unknown",
                "name": "",
                "detail": ""
                }
            ],
            "tm_create": "2022-06-17 06:06:14.446158",
            "tm_update": "2022-06-17 06:06:14.446167",
            "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-06-17 06:06:14.446158"
    }

Get detail of conversation
--------------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "name": "conversation",
        "detail": "conversation detail",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "participants": [
            {
                "type": "line",
                "target": "",
                "target_name": "me",
                "name": "",
                "detail": ""
            },
            {
                "type": "line",
                "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "target_name": "Unknown",
                "name": "",
                "detail": ""
            }
        ],
        "tm_create": "2022-06-17 06:06:14.446158",
        "tm_update": "2022-06-17 06:06:14.446167",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Send a message to the conversation
----------------------------------

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/messages?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "text": "Hello, this is a test message. Thank you for your time."
        }'

    {
        "id": "0c8f23cb-e878-49bf-b69e-03f59252f217",
        "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "status": "sent",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "source_target": "",
        "text": "Hello, this is a test message. Thank you for your time.",
        "medias": [],
        "tm_create": "2022-06-20 03:07:11.372307",
        "tm_update": "2022-06-20 03:07:11.372315",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Get list of conversation messages
---------------------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/messages?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "0c8f23cb-e878-49bf-b69e-03f59252f217",
                "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
                "status": "sent",
                "reference_type": "line",
                "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "source_target": "",
                "text": "Hello, this is a test message. Thank you for your time.",
                "medias": [],
                "tm_create": "2022-06-20 03:07:11.372307",
                "tm_update": "2022-06-20 03:07:11.372315",
                "tm_delete": "9999-01-01 00:00:00.000000"
            },
            ...
        ],
        "next_page_token": "2022-06-17 06:06:14.948432"
    }
