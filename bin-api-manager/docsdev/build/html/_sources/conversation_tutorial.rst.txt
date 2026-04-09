.. _conversation-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with conversations, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A conversation already created by an incoming message (SMS/MMS via ``message`` type, or LINE via ``line`` type). Conversations are auto-created when messages arrive -- there is no ``POST /conversations`` endpoint.
* (Optional) A conversation account configured for the messaging channel (LINE credentials, etc.). Manage accounts via ``GET /conversation_accounts``.

.. note:: **AI Implementation Hint**

   Conversations are auto-created when an inbound message arrives on a configured channel (SMS/MMS or LINE). There is no ``POST /conversations`` endpoint to create conversations manually. The ``type`` field indicates the channel: ``message`` for SMS/MMS or ``line`` for LINE. Messages sent within a conversation incur per-channel delivery costs. When sending a message to a conversation, pass the conversation UUID in the URL path: ``POST /conversations/{id}/messages``.

Get list of conversations
-------------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "owner_type": "agent",
                "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
                "account_id": "c5d6e7f8-a9b0-1234-cdef-567890abcdef",
                "name": "conversation",
                "detail": "conversation detail",
                "type": "line",
                "dialog_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "self": {
                    "type": "line",
                    "target": "",
                    "target_name": "me",
                    "name": "",
                    "detail": ""
                },
                "peer": {
                    "type": "line",
                    "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                    "target_name": "Unknown",
                    "name": "",
                    "detail": ""
                },
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
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "account_id": "c5d6e7f8-a9b0-1234-cdef-567890abcdef",
        "name": "conversation",
        "detail": "conversation detail",
        "type": "line",
        "dialog_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "self": {
            "type": "line",
            "target": "",
            "target_name": "me",
            "name": "",
            "detail": ""
        },
        "peer": {
            "type": "line",
            "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
            "target_name": "Unknown",
            "name": "",
            "detail": ""
        },
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
            "text": "Hello, this is a test message. Thank you for your time.",
            "medias": []
        }'

    {
        "id": "0c8f23cb-e878-49bf-b69e-03f59252f217",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "direction": "outgoing",
        "status": "progressing",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
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
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
                "direction": "outgoing",
                "status": "done",
                "reference_type": "line",
                "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
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
