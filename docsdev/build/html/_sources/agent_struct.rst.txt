.. _agent-struct:

Struct
======

.. _agent-struct-agent:

Agent
-----

.. code::

    {
        "id": "<string>",
        "username": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "ring_method": "<string>",
        "status": "<string>",
        "permission": <number>,
        "tag_ids": [
            "<string>"
        ],
        "addresses": [
            ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    },

* *id*: Agent's id.
* *username*: Agent's login username.
* *name*: Agent's name.
* *detail*: Agent's detail description.
* *ring_method*: Ring method for agent calling. See detail :ref:`here <agent-struct-ring_method>`.
* *status*: Agent's status. See detail :ref:`here <agent-struct-status>`.
* *permission*: Agent's permission.
* *tag_ids*: List of agent's tags.
* *addresses*: List of agent's addresses. See detail :ref:`here <call-struct-address>`.

.. _agent-struct-ring_method:

Ring method
-----------
Agent's calling method.

======== ============
Type     Description
======== ============
ringall  Dial to the all addresses.
======== ============

.. _agent-struct-status:

Status
------
Agent's status.

========== ============
Type       Description
========== ============
available  Available.
away       Away.
busy       Busy.
offline    Offline.
ringing    Ringing.
========== ============

