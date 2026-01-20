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

* id: Talk's unique identifier.
* customer_id: Customer's unique identifier.
* type: Talk's type. See detail :ref:`here <talk-struct-talk-type>`.
* tm_create: Timestamp when the talk was created.
* tm_update: Timestamp when the talk was last updated.
* tm_delete: Timestamp when the talk was deleted (soft delete).

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

=========== ============
Type        Description
=========== ============
normal      1:1 talk between two agents.
group       Group talk with multiple participants.
=========== ============
