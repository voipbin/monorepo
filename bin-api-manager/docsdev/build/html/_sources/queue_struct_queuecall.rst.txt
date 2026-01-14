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

* id: Queuecall's ID.
* reference_type: Referenced resource's type. See detail :ref:`here <queue-struct-queuecall-reference_type>`.
* status: Queuecall's status. See detail :ref:`here <queue-struct-queuecall-type>`.
* service_agent_id: Connected agent_id.

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
Referenced resource's type.

======== ================
Type     Description
======== ================
call     Call reference resource
======== ================

.. _queue-struct-queuecall-type:

Status
------
Queuecall's status.

========= ================
Type      Description
========= ================
wait      Queue is looking for an available agent and the queuecall is looping the waiting actions.
kicking   A queuecall is being kicked.
service   A queuecall is talking with agent.
done      A queuecall is done.
abandoned A queuecall has abandoned before connect to the agent.
========= ================

