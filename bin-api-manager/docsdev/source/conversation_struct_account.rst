.. _conversation-struct-account:

Conversation Account
====================

.. _conversation-struct-account-account:

Account
-------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "type": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "provider_data": {},
        "message_flow_id": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The conversation account's unique identifier. Returned when creating via ``POST /conversation-accounts`` or listing via ``GET /conversation-accounts``.
* ``customer_id`` (UUID): The customer who owns this conversation account. Obtained from the ``id`` field of ``GET /customers``.
* ``type`` (enum string): The messaging platform type. See :ref:`Type <conversation-struct-account-type>`.
* ``name`` (string): A human-readable name for this conversation account.
* ``detail`` (string): Additional description or notes about this account.
* ``provider_data`` (object): Provider-specific configuration set at creation time. Omitted if not set. For WhatsApp accounts, contains ``phone_number_id`` and ``app_secret``. See :ref:`WhatsApp Account Fields <conversation-struct-account-whatsapp>`.
* ``message_flow_id`` (UUID): The flow to execute when a message is received on this account. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
* ``tm_create`` (string, ISO 8601): Timestamp when this account was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this account.
* ``tm_delete`` (string, ISO 8601): Timestamp when this account was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. _conversation-struct-account-type:

Type
----

All possible values for the ``type`` field:

========= =============================================
Type      Description
========= =============================================
line      LINE messaging platform account
sms       SMS messaging account
whatsapp  WhatsApp Business account (Meta Cloud API)
========= =============================================

.. _conversation-struct-account-whatsapp:

WhatsApp Account Fields
-----------------------

When creating a ``whatsapp`` account, the following fields are required in addition to ``name``, ``detail``, and ``message_flow_id``.

* ``token`` (string, write-only): The Meta system user access token. Used as the Bearer token when calling the Meta Cloud API to send outbound messages. Never returned in API responses.
* ``secret`` (string, write-only): The webhook verify token. Echoed back verbatim during the Meta hub challenge (``GET`` request from Meta to your webhook URL). Never returned in API responses.
* ``provider_data`` (object): WhatsApp-specific configuration. Returned in ``GET`` responses. Contains the following keys:

  * ``phone_number_id`` (string): The Meta phone number ID associated with your WhatsApp Business phone number. Found in Meta Business Manager under **WhatsApp → Phone numbers**.
  * ``app_secret`` (string): The Meta app secret. Used to validate the ``X-Hub-Signature-256`` header on inbound webhook requests.

.. note::

   ``token`` and ``secret`` are **write-only**. They are accepted on ``POST /conversation_accounts`` and ``PUT /conversation_accounts/{id}`` but are **never included** in ``GET`` responses or webhook payloads. ``provider_data`` is readable and returned in ``GET`` responses.

.. _conversation-struct-account-whatsapp-webhook:

WhatsApp Webhook URL
--------------------

When configuring the webhook in Meta Business Manager, use the following URL pattern::

    https://hook.voipbin.net/v1.0/conversation/accounts/{account_id}

Replace ``{account_id}`` with the ``id`` returned when the account is created.

Meta sends two types of requests to this URL:

1. **Hub challenge** (``GET``): Sent by Meta to verify your endpoint. VoIPBIN responds automatically using the ``secret`` (verify token) you configured.
2. **Inbound messages** (``POST``): WhatsApp messages forwarded by Meta. VoIPBIN validates the ``X-Hub-Signature-256`` header using ``app_secret`` from ``provider_data``.

**Inbound message identifiers**

* ``dialog_id``: Set to the sender's WhatsApp ID (``wa_id``), which is their E.164 phone number without the leading ``+`` (e.g., ``15551234567``).
* ``transaction_id``: Set to the WhatsApp message ID (``wamid``), a unique identifier assigned by Meta to each message.

**Outbound messages**

VoIPBIN sends outbound text messages via the Meta Cloud API using the ``phone_number_id`` from ``provider_data`` and the ``token`` you configured. The ``wamid`` returned by Meta is stored as the conversation message's ``transaction_id``.

Examples
--------

LINE account (no ``provider_data``):

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "line",
        "name": "Customer Support LINE",
        "detail": "LINE account for customer support inquiries",
        "message_flow_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "tm_create": "2024-03-01T10:00:00.000000Z",
        "tm_update": "2024-03-01T10:00:00.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }

WhatsApp account (with ``provider_data``):

.. code::

    {
        "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "whatsapp",
        "name": "WhatsApp Customer Support",
        "detail": "WhatsApp Business account via Meta Cloud API",
        "provider_data": {
            "phone_number_id": "123456789012345",
            "app_secret": "abcdef1234567890abcdef1234567890"
        },
        "message_flow_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "tm_create": "2024-03-01T10:00:00.000000Z",
        "tm_update": "2024-03-01T10:00:00.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }
