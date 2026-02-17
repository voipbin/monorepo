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

* ``id`` (UUID): The agent's unique identifier. Returned when creating an agent via ``POST /agents`` or when listing agents via ``GET /agents``.
* ``username`` (String): The agent's login username. Must be unique within the customer account.
* ``name`` (String): The agent's display name.
* ``detail`` (String): An optional description of the agent.
* ``ring_method`` (enum string): The method used to ring the agent's addresses when a call is routed. See :ref:`Ring method <agent-struct-agent-ring_method>`.
* ``status`` (enum string): The agent's current availability status. See :ref:`Status <agent-struct-agent-status>`.
* ``permission`` (Integer): The agent's permission level as a bitmask value. See :ref:`Permission <agent-struct-agent-permission>`.
* ``tag_ids`` (Array of UUID): List of tag IDs assigned to this agent for skill-based routing. Each ID is obtained from the ``id`` field of ``GET /tags``.
* ``addresses`` (Array of Object): List of contact addresses where calls are delivered to this agent. See :ref:`Address <common-struct-address-address>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the agent was created.
* ``tm_update`` (string, ISO 8601): Timestamp when the agent was last updated.
* ``tm_delete`` (string, ISO 8601): Timestamp when the agent was deleted, if applicable.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` indicates the agent has not been deleted. When creating an agent, ``status`` defaults to ``offline`` and ``permission`` defaults to ``0`` (no permissions). The agent must set their status to ``available`` via ``PUT /agents/{id}/status`` before they can receive queue calls.

.. _agent-struct-agent-ring_method:

Ring method
-----------
The method used to ring the agent's contact addresses when a call is routed.

========== ============
Type       Description
========== ============
ringall    Dial all of the agent's addresses simultaneously. The first address to answer is connected; the rest are cancelled.
========== ============

.. _agent-struct-agent-status:

Status
------
The agent's current availability status. Determines whether the agent can receive calls from queues.

========== ============
Type       Description
========== ============
available  Agent is logged in and ready to receive queue calls.
away       Agent is temporarily unavailable (e.g., break, meeting). Cannot receive queue calls.
busy       Agent is currently handling a call. Set automatically by the system. Cannot receive additional queue calls.
offline    Agent is logged out of the system. Cannot receive queue calls.
ringing    A call is being delivered to the agent. Set automatically by the system. Cannot receive additional queue calls.
========== ============

.. _agent-struct-agent-permission:

Permission
----------
The agent's permission level, represented as a bitmask integer.

============== ================
Permission     Description
============== ================
0              (0x0000) No permissions.
65535          (0xFFFF) All permissions.
1              (0x0001) VoIPBIN project super admin.
15             (0x000F) All project-level permissions.
16             (0x0010) Customer-level agent (basic user).
32             (0x0020) Customer-level admin (can manage agents and resources).
64             (0x0040) Customer-level manager.
240            (0x00F0) All customer-level permissions.
============== ================
