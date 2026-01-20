.. _agent-struct-agent:

Struct
======

.. _agent-struct-agent-agent:

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
* *ring_method*: Ring method for agent calling. See detail :ref:`here <agent-struct-agent-ring_method>`.
* *status*: Agent's status. See detail :ref:`here <agent-struct-agent-status>`.
* *permission*: Agent's permission.
* *tag_ids*: List of agent's tags.
* *addresses*: List of agent's addresses. See detail :ref:`here <common-struct-address-address>`.

.. _agent-struct-agent-ring_method:

Ring method
-----------
Agent's calling method.

======== ============
Type     Description
======== ============
ringall  Dial to the all addresses.
======== ============

.. _agent-struct-agent-status:

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

Permission
----------
Agent's permission

============== ================
Permission     Description
============== ================
0              (0x0000)None.
65535          (0xFFFF)All permission.
1              (0x0001)Permission for voipbin project super admin.
15             (0x000F)All permission for project level.
16             (0x0010)Permission for customer level agent
32             (0x0020)Permission for customer level admin
64             (0x0040)Permission for customer level manager
240            (0x00F0)All permission for customer level
============== ================
