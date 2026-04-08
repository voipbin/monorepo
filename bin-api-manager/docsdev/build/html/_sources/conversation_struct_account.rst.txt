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
* ``message_flow_id`` (UUID): The flow to execute when a message is received on this account. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
* ``tm_create`` (string, ISO 8601): Timestamp when this account was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this account.
* ``tm_delete`` (string, ISO 8601): Timestamp when this account was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. _conversation-struct-account-type:

Type
----

All possible values for the ``type`` field:

====== ===========
Type   Description
====== ===========
line   LINE messaging platform account
sms    SMS messaging account
====== ===========

Example
-------

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
