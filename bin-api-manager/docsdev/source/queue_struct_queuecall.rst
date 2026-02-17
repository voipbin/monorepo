.. _queue-struct-queuecall:

Queuecall
=========

.. _queue-struct-queuecall-queuecall:

Queuecall
---------
Queuecall struct

.. code::

    {
        "id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "service_agent_id": "<string>",
        "tm_create": "<string>",
        "tm_service": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The queuecall's unique identifier. Returned when a call enters a queue or when listing queuecalls via ``GET /queuecalls``.
* ``reference_type`` (enum string): The type of the referenced resource. See :ref:`Reference type <queue-struct-queuecall-reference_type>`.
* ``reference_id`` (UUID): The ID of the referenced resource (e.g., the call). Obtained from ``GET /calls`` when reference_type is ``call``.
* ``status`` (enum string): The queuecall's current status. See :ref:`Status <queue-struct-queuecall-type>`.
* ``service_agent_id`` (UUID): The ID of the agent connected to this queuecall. Obtained from ``GET /agents``. Set to ``00000000-0000-0000-0000-000000000000`` if no agent is connected yet.
* ``tm_create`` (string, ISO 8601): Timestamp when the queuecall was created (call entered the queue).
* ``tm_service`` (string, ISO 8601): Timestamp when the agent was connected and service began. Set to ``9999-01-01 00:00:00.000000`` if service has not started.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this queuecall.
* ``tm_delete`` (string, ISO 8601): Timestamp when the queuecall ended. Set to ``9999-01-01 00:00:00.000000`` if still active.

.. note:: **AI Implementation Hint**

   To calculate a caller's wait time, subtract ``tm_create`` from ``tm_service``. Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred.

Example
+++++++

.. code::

    {
        "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
        "reference_type": "call",
        "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
        "status": "done",
        "service_agent_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "tm_create": "2022-03-29 15:07:46.111715",
        "tm_service": "2022-03-29 15:08:04.811442",
        "tm_update": "2022-03-29 15:08:25.814885",
        "tm_delete": "2022-03-29 15:08:25.814885"
    }

.. _queue-struct-queuecall-reference_type:

Reference type
--------------
The type of the resource that this queuecall references.

======== ================
Type     Description
======== ================
call     The queuecall references a call resource. Use the ``reference_id`` with ``GET /calls/{id}`` to retrieve the associated call.
======== ================

.. _queue-struct-queuecall-type:

Status
------
The queuecall's current lifecycle status. Transitions follow: ``wait`` -> ``entering`` -> ``service`` -> ``done`` (success path) or ``wait`` -> ``abandoned`` (failure path).

=========== ================
Type        Description
=========== ================
wait        The system is searching for an available agent. The caller hears the queue's wait actions (hold music, announcements) in a loop.
entering    The queuecall is connecting to an available agent's conference room. This is a brief transitional state between ``wait`` and ``service``.
kicking     The queuecall is being removed from the queue (e.g., due to wait timeout). This is a brief transitional state.
service     An agent has been connected. The caller and agent are in conversation.
done        The queuecall completed successfully. The agent finished helping the caller.
abandoned   The queuecall ended without service. The caller hung up, the wait timeout was exceeded, or the call was otherwise terminated before an agent connected.
=========== ================

