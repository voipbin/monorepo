.. _conference-struct-conferencecall:

conferencecall
==============

Conferencecall
--------------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "conference_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The conferencecall's unique identifier. Returned when a participant joins a conference or when listing via ``GET /conferencecalls``.
* ``customer_id`` (UUID): The customer who owns this conferencecall. Obtained from ``GET /customers``.
* ``conference_id`` (UUID): The conference this participant belongs to. Obtained from ``GET /conferences``.
* ``reference_type`` (enum string): The type of the referenced resource. See :ref:`Reference type <conference-struct-conferencecall-reference_type>`.
* ``reference_id`` (UUID): The ID of the referenced resource (e.g., a call ID). Obtained from ``GET /calls/{id}`` when ``reference_type`` is ``call``.
* ``status`` (enum string): The conferencecall's current status. See :ref:`Status <conference-struct-conferencecall-status>`.
* ``tm_create`` (String, ISO 8601): Timestamp when the conferencecall was created.
* ``tm_update`` (String, ISO 8601): Timestamp of the last update to any conferencecall property.
* ``tm_delete`` (String, ISO 8601): Timestamp when the conferencecall was deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the participant record still exists. To remove a participant from a conference, use ``DELETE /conferencecalls/{id}``.


Example
+++++++

.. code::

    {
        "id": "b8aa51f6-5cc1-40ba-9737-45ca24dab153",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "conference_id": "99accfb7-c0dd-4a54-997d-dd18af7bc280",
        "reference_type": "call",
        "reference_id": "7cb70145-a20a-4070-8b23-9131410d301d",
        "status": "leaved",
        "tm_create": "2022-08-06 16:57:12.247946",
        "tm_update": "2022-08-06 19:09:47.349667",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _conference-struct-conferencecall-reference_type:

Reference type
--------------
The type of resource participating in the conference (enum string).

========== ==============
Type       Description
========== ==============
call       The participant is a call. The ``reference_id`` is a call ID obtainable from ``GET /calls``.
========== ==============

.. _conference-struct-conferencecall-status:

Status
--------------
The conferencecall's current status (enum string). States only move forward, never backward.

========== ==============
Status     Description
========== ==============
joining    The call is connecting to the conference. Pre-conference flow actions may be executing (e.g., greeting message).
joined     The call is active in the conference. Audio is flowing. The participant can hear and speak with others.
leaving    The call is being disconnected from the conference. Triggered by hangup or ``DELETE /conferencecalls/{id}``.
leaved     The call has fully left the conference. This is the final state. No further changes are possible.
========== ==============
