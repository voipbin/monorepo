.. _team-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before creating a team, you need:

* A valid authentication token (String). Obtain via ``POST /auth/login`` or use an accesskey from ``GET /accesskeys``.
* At least two AI configurations (UUIDs). Create them via ``POST /ais`` or obtain from ``GET /ais``. Each AI defines a distinct persona with its own LLM, prompt, TTS, and STT settings.

.. note:: **AI Implementation Hint**

   Each member in a team references an independent AI configuration. Design each AI's ``init_prompt`` for a specific role (e.g., receptionist, billing specialist). The team's graph structure handles routing — individual AIs do not need to know about each other.

Step 1: Create AI Configurations
---------------------------------

First, create the AI configurations that will back each team member. In this example, we create a receptionist and a billing specialist.

**Create Receptionist AI:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/ais?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Receptionist AI",
            "detail": "Qualifies caller intent and routes to the right specialist",
            "engine_model": "openai.gpt-4o",
            "engine_key": "sk-...",
            "init_prompt": "You are a friendly receptionist. Greet the caller, ask how you can help, and determine if they need billing assistance or technical support.",
            "tts_type": "elevenlabs",
            "stt_type": "deepgram",
            "tool_names": ["all"]
        }'

Response:

.. code::

    {
        "id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",   // Save this as receptionist_ai_id
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Receptionist AI",
        ...
    }

**Create Billing AI:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/ais?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Billing Specialist AI",
            "detail": "Handles billing inquiries, payment issues, and account charges",
            "engine_model": "openai.gpt-4o",
            "engine_key": "sk-...",
            "init_prompt": "You are a billing specialist. Help callers with invoices, payments, charges, and account balance questions. Be precise with numbers and dates.",
            "tts_type": "elevenlabs",
            "stt_type": "deepgram",
            "tool_names": ["all"]
        }'

Response:

.. code::

    {
        "id": "b193d6ea-743d-59e8-c81c-5aaf3a195bc2",   // Save this as billing_ai_id
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Billing Specialist AI",
        ...
    }

Step 2: Create the Team
------------------------

Create a team with two members (receptionist and billing specialist) and a transition from the receptionist to the billing specialist.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/teams?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Customer Service Team",
            "detail": "Routes callers to the right specialist based on intent",
            "start_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
            "members": [
                {
                    "id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                    "name": "Receptionist",
                    "ai_id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
                    "transitions": [
                        {
                            "function_name": "transfer_to_billing",
                            "description": "Transfer to the billing specialist when the caller asks about invoices, payments, charges, or account balance.",
                            "next_member_id": "e5f6a7b8-c9d0-1234-efab-567890123456"
                        }
                    ]
                },
                {
                    "id": "e5f6a7b8-c9d0-1234-efab-567890123456",
                    "name": "Billing Specialist",
                    "ai_id": "b193d6ea-743d-59e8-c81c-5aaf3a195bc2",
                    "transitions": [
                        {
                            "function_name": "transfer_to_receptionist",
                            "description": "Transfer back to the receptionist if the caller wants help with something other than billing.",
                            "next_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345"
                        }
                    ]
                }
            ],
            "parameter": {
                "language": "en",
                "department": "customer-service"
            }
        }'

Response:

.. code::

    {
        "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",   // Save this as team_id
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Customer Service Team",
        "detail": "Routes callers to the right specialist based on intent",
        "start_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
        "members": [
            {
                "id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "name": "Receptionist",
                "ai_id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
                "transitions": [
                    {
                        "function_name": "transfer_to_billing",
                        "description": "Transfer to the billing specialist when the caller asks about invoices, payments, charges, or account balance.",
                        "next_member_id": "e5f6a7b8-c9d0-1234-efab-567890123456"
                    }
                ]
            },
            {
                "id": "e5f6a7b8-c9d0-1234-efab-567890123456",
                "name": "Billing Specialist",
                "ai_id": "b193d6ea-743d-59e8-c81c-5aaf3a195bc2",
                "transitions": [
                    {
                        "function_name": "transfer_to_receptionist",
                        "description": "Transfer back to the receptionist if the caller wants help with something other than billing.",
                        "next_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345"
                    }
                ]
            }
        ],
        "parameter": {
            "language": "en",
            "department": "customer-service"
        },
        "tm_create": "2026-02-27 10:00:00.000000",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Step 3: Get Team Details
-------------------------

Retrieve the team to verify its configuration.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/teams/c3d4e5f6-a7b8-9012-cdef-345678901234?token=<YOUR_AUTH_TOKEN>'

Response:

.. code::

    {
        "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Customer Service Team",
        "detail": "Routes callers to the right specialist based on intent",
        "start_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
        "members": [
            {
                "id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "name": "Receptionist",
                "ai_id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
                "transitions": [...]
            },
            {
                "id": "e5f6a7b8-c9d0-1234-efab-567890123456",
                "name": "Billing Specialist",
                "ai_id": "b193d6ea-743d-59e8-c81c-5aaf3a195bc2",
                "transitions": [...]
            }
        ],
        "parameter": {
            "language": "en",
            "department": "customer-service"
        },
        "tm_create": "2026-02-27 10:00:00.000000",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Step 4: Update a Team
----------------------

Add a third member (Technical Support) to the existing team.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/teams/c3d4e5f6-a7b8-9012-cdef-345678901234?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Customer Service Team",
            "detail": "Routes callers to billing or technical support",
            "start_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
            "members": [
                {
                    "id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                    "name": "Receptionist",
                    "ai_id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
                    "transitions": [
                        {
                            "function_name": "transfer_to_billing",
                            "description": "Transfer to the billing specialist when the caller asks about invoices, payments, charges, or account balance.",
                            "next_member_id": "e5f6a7b8-c9d0-1234-efab-567890123456"
                        },
                        {
                            "function_name": "transfer_to_support",
                            "description": "Transfer to technical support when the caller reports a technical issue, outage, or needs troubleshooting help.",
                            "next_member_id": "f6a7b8c9-d0e1-2345-fabc-678901234567"
                        }
                    ]
                },
                {
                    "id": "e5f6a7b8-c9d0-1234-efab-567890123456",
                    "name": "Billing Specialist",
                    "ai_id": "b193d6ea-743d-59e8-c81c-5aaf3a195bc2",
                    "transitions": [
                        {
                            "function_name": "transfer_to_receptionist",
                            "description": "Transfer back to the receptionist if the caller wants help with something other than billing.",
                            "next_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345"
                        }
                    ]
                },
                {
                    "id": "f6a7b8c9-d0e1-2345-fabc-678901234567",
                    "name": "Technical Support",
                    "ai_id": "c294e7fb-854e-6af9-d92d-6bb04b206cd3",
                    "transitions": [
                        {
                            "function_name": "escalate_to_receptionist",
                            "description": "Transfer back to the receptionist if the caller wants to discuss a different topic.",
                            "next_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345"
                        }
                    ]
                }
            ],
            "parameter": {
                "language": "en",
                "department": "customer-service"
            }
        }'

Response:

.. code::

    {
        "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Customer Service Team",
        "detail": "Routes callers to billing or technical support",
        "start_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
        "members": [
            ...
        ],
        "parameter": {
            "language": "en",
            "department": "customer-service"
        },
        "tm_create": "2026-02-27 10:00:00.000000",
        "tm_update": "2026-02-27 10:05:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Step 5: List Teams
-------------------

List all teams owned by your account.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/teams?token=<YOUR_AUTH_TOKEN>'

Response:

.. code::

    {
        "result": [
            {
                "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "name": "Customer Service Team",
                "detail": "Routes callers to billing or technical support",
                "start_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "members": [...],
                "parameter": {
                    "language": "en",
                    "department": "customer-service"
                },
                "tm_create": "2026-02-27 10:00:00.000000",
                "tm_update": "2026-02-27 10:05:00.000000",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": ""
    }

Step 6: Delete a Team
----------------------

Delete a team when it is no longer needed.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/teams/c3d4e5f6-a7b8-9012-cdef-345678901234?token=<YOUR_AUTH_TOKEN>'

Response:

.. code::

    {
        "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Customer Service Team",
        "detail": "Routes callers to billing or technical support",
        "start_member_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
        "members": [...],
        "parameter": {
            "language": "en",
            "department": "customer-service"
        },
        "tm_create": "2026-02-27 10:00:00.000000",
        "tm_update": "2026-02-27 10:05:00.000000",
        "tm_delete": "2026-02-27 11:00:00.000000"
    }

.. note:: **AI Implementation Hint**

   Deleting a team does not delete the underlying AI configurations. The AIs referenced by ``ai_id`` remain available and can be reused in other teams or standalone flows. To fully clean up, delete the AI configurations separately via ``DELETE /ais/{ai_id}``.

Best Practices
--------------

**Team Design:**

- Keep each member focused on a single domain (billing, support, sales). Broad prompts lead to less accurate routing.
- Write transition ``description`` fields as explicit conditions, not vague labels. The LLM relies on these to decide when to switch.
- Design bidirectional transitions for flexibility — allow callers to go back to the receptionist from any specialist.

**AI Configuration:**

- Each member's AI should have a prompt that includes context about its role within the team. For example: ``"You are the billing specialist on the customer service team. A receptionist has already greeted the caller and identified their billing question."``
- Use consistent TTS voices across members for a cohesive caller experience, or deliberately vary them to signal the handoff.

**Testing:**

- Test each AI configuration individually via ``POST /calls`` with an inline ``ai_talk`` action before assembling them into a team.
- Verify transitions by testing conversations that should trigger each ``function_name``.

Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** ``start_member_id`` does not match any member's ``id`` in the ``members`` array.
    * **Fix:** Verify that ``start_member_id`` is set to a valid member ``id``.

* **400 Bad Request:**
    * **Cause:** A ``next_member_id`` in a transition references a member that does not exist in the ``members`` array.
    * **Fix:** Ensure all ``next_member_id`` values point to valid member ``id`` values within the same team.

* **402 Payment Required:**
    * **Cause:** Insufficient account balance. Team conversations consume credits for each active member's LLM, TTS, and STT usage.
    * **Fix:** Check balance via ``GET /billing-accounts``. Top up before retrying.

* **404 Not Found:**
    * **Cause:** The team UUID does not exist or belongs to a different customer.
    * **Fix:** Verify the UUID was obtained from ``GET /teams`` or ``POST /teams``.

* **AI not switching members:**
    * **Cause:** The transition ``description`` is too vague for the LLM to match.
    * **Fix:** Make the description more specific. Example: ``"Transfer when the caller mentions billing, invoices, or payments"`` instead of ``"Transfer to billing"``.

* **Wrong member activated at start:**
    * **Cause:** ``start_member_id`` points to the wrong member.
    * **Fix:** Update the team via ``PUT /teams/{id}`` with the correct ``start_member_id``.
