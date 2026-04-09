.. _talk-tutorial:

Tutorial
========

This tutorial demonstrates how to use the Talk API to create conversations, manage participants, send messages with threading and reactions.

Prerequisites
+++++++++++++

Before using the Talk API, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* Agent IDs (UUIDs) for the participants you want to add. Obtain agent IDs via ``GET /agents``.
* (For threading) An existing message ID to reply to. Obtain message IDs via ``GET https://api.voipbin.net/v1.0/service_agents/talk_messages``.

.. note:: **AI Implementation Hint**

   Talk API uses the ``/service_agents/talk_chats`` and ``/service_agents/talk_messages`` endpoints. The agent making the request must be authenticated and will automatically become a participant. When adding other participants, use their agent ``id`` (UUID) from ``GET /agents`` as the ``owner_id``. Chat types are: ``direct`` (1:1 private), ``group`` (private multi-user), or ``talk`` (public channel).

Create a Talk
-------------

Create a new talk conversation:

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/service_agents/talk_chats?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "type": "normal"
    }'

    {
        "id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "normal",
        "member_count": 0,
        "tm_create": "2024-01-17 10:30:00.000000",
        "tm_update": "2024-01-17 10:30:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Add Participants
----------------

Add agents to the talk:

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/service_agents/talk_chats/e8b2e976-f043-44c8-bb89-e214e225e813/participants?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "owner_type": "agent",
        "owner_id": "47fe0b7c-7333-46cf-8b23-61e14e62490a"
    }'

    {
        "id": "f4d6e9b3-5c4a-4d5e-9f8a-2b3c4d5e6f70",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "47fe0b7c-7333-46cf-8b23-61e14e62490a",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "tm_joined": "2024-01-17 10:31:00.000000"
    }

Send a Message
--------------

Send a message to the talk:

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/service_agents/talk_messages?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "type": "normal",
        "text": "Hello team! Let'\''s discuss the new feature."
    }'

    {
        "id": "a3c5e8f2-4d3a-4b5c-9e7f-1a2b3c4d5e6f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "type": "normal",
        "text": "Hello team! Let's discuss the new feature.",
        "medias": [],
        "metadata": {
            "reactions": []
        },
        "tm_create": "2024-01-17 10:32:00.000000",
        "tm_update": "2024-01-17 10:32:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Reply to a Message (Threading)
-------------------------------

Reply to an existing message by specifying the parent_id:

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/service_agents/talk_messages?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "parent_id": "a3c5e8f2-4d3a-4b5c-9e7f-1a2b3c4d5e6f",
        "type": "normal",
        "text": "Great idea! I'\''ll start working on the requirements."
    }'

    {
        "id": "b2d4f7a1-3c2e-4f5a-8d9c-2e3f4a5b6c7d",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "47fe0b7c-7333-46cf-8b23-61e14e62490a",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "parent_id": "a3c5e8f2-4d3a-4b5c-9e7f-1a2b3c4d5e6f",
        "type": "normal",
        "text": "Great idea! I'll start working on the requirements.",
        "medias": [],
        "metadata": {
            "reactions": []
        },
        "tm_create": "2024-01-17 10:33:00.000000",
        "tm_update": "2024-01-17 10:33:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Add a Reaction
--------------

Add an emoji reaction to a message:

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/service_agents/talk_messages/b2d4f7a1-3c2e-4f5a-8d9c-2e3f4a5b6c7d/reactions?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "emoji": "thumbsup"
    }'

    {
        "id": "b2d4f7a1-3c2e-4f5a-8d9c-2e3f4a5b6c7d",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "47fe0b7c-7333-46cf-8b23-61e14e62490a",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "parent_id": "a3c5e8f2-4d3a-4b5c-9e7f-1a2b3c4d5e6f",
        "type": "normal",
        "text": "Great idea! I'll start working on the requirements.",
        "medias": [],
        "metadata": {
            "reactions": [
                {
                    "emoji": "thumbsup",
                    "owner_type": "agent",
                    "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
                    "tm_create": "2024-01-17 10:34:00.000000"
                }
            ]
        },
        "tm_create": "2024-01-17 10:33:00.000000",
        "tm_update": "2024-01-17 10:34:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

List Messages
-------------

Retrieve messages from a talk with pagination:

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/service_agents/talk_messages?page_size=50&token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "a3c5e8f2-4d3a-4b5c-9e7f-1a2b3c4d5e6f",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "owner_type": "agent",
                "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
                "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
                "type": "normal",
                "text": "Hello team! Let's discuss the new feature.",
                "medias": [],
                "metadata": {
                    "reactions": []
                },
                "tm_create": "2024-01-17 10:32:00.000000",
                "tm_update": "2024-01-17 10:32:00.000000",
                "tm_delete": "9999-01-01 00:00:00.000000"
            },
            {
                "id": "b2d4f7a1-3c2e-4f5a-8d9c-2e3f4a5b6c7d",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "owner_type": "agent",
                "owner_id": "47fe0b7c-7333-46cf-8b23-61e14e62490a",
                "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
                "parent_id": "a3c5e8f2-4d3a-4b5c-9e7f-1a2b3c4d5e6f",
                "type": "normal",
                "text": "Great idea! I'll start working on the requirements.",
                "medias": [],
                "metadata": {
                    "reactions": [
                        {
                            "emoji": "thumbsup",
                            "owner_type": "agent",
                            "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
                            "tm_create": "2024-01-17 10:34:00.000000"
                        }
                    ]
                },
                "tm_create": "2024-01-17 10:33:00.000000",
                "tm_update": "2024-01-17 10:34:00.000000",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2024-01-17 10:33:00.000000"
    }

Remove a Participant
--------------------

Remove a participant from the talk:

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/service_agents/talk_chats/e8b2e976-f043-44c8-bb89-e214e225e813/participants/f4d6e9b3-5c4a-4d5e-9f8a-2b3c4d5e6f70?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "f4d6e9b3-5c4a-4d5e-9f8a-2b3c4d5e6f70",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "47fe0b7c-7333-46cf-8b23-61e14e62490a",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "tm_joined": "2024-01-17 10:31:00.000000"
    }

Delete a Talk
-------------

Delete a talk (soft delete):

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/service_agents/talk_chats/e8b2e976-f043-44c8-bb89-e214e225e813?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "normal",
        "member_count": 0,
        "tm_create": "2024-01-17 10:30:00.000000",
        "tm_update": "2024-01-17 10:35:00.000000",
        "tm_delete": "2024-01-17 10:35:00.000000"
    }
