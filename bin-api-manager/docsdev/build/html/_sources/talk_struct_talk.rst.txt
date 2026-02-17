.. _talk-struct-talk:

Talk
====

.. _talk-struct-talk-talk:

Talk
----

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "type": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The talk's unique identifier. Returned when creating via ``POST /service_agents/talk_chats`` or listing via ``GET /service_agents/talk_chats``.
* ``customer_id`` (UUID): The customer's unique identifier. Obtained from ``GET /customers``.
* ``type`` (enum string): The talk's type. See :ref:`Type <talk-struct-talk-type>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the talk was created.
* ``tm_update`` (string, ISO 8601): Timestamp when the talk was last updated.
* ``tm_delete`` (string, ISO 8601): Timestamp when the talk was deleted (soft delete).

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` means the talk has not been deleted. An empty string ``""`` also indicates the talk is active.

Example
+++++++

.. code::

    {
        "id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "normal",
        "tm_create": "2024-01-17 10:30:00.000000",
        "tm_update": "2024-01-17 10:30:00.000000",
        "tm_delete": ""
    }

.. _talk-struct-talk-type:

Type
----
Talk's type determines whether it's a one-on-one or group conversation.

=========== ============================================================
Type        Description
=========== ============================================================
normal      1:1 talk between two agents. Private direct conversation.
group       Group talk with multiple participants. Team discussions.
=========== ============================================================
