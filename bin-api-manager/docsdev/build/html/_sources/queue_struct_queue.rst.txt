.. _queue-struct-queue:

Queue
======

.. _queue-struct-queue-queue:

Queue
-----
Queue struct

.. code::

    {
        "id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "routing_method": "<string>",
        "tag_ids": [
            "<string>",
            ...
        ],
        "wait_actions": [
            {
                action...
            }
        ],
        "wait_timeout": <number>,
        "service_timeout": <number>,
        "wait_queue_call_ids": [
            "<string>",
            ...
        ],
        "service_queue_call_ids": [
            "<string>",
            ...
        ],
        "total_incoming_count": <number>,
        "total_serviced_count": <number>,
        "total_abandoned_count": <number>,
        "total_waittime": <number>,
        "total_service_duration": <number>,
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>",
    }

* ``id`` (UUID): The queue's unique identifier. Returned when creating a queue via ``POST /queues`` or when listing queues via ``GET /queues``.
* ``name`` (String): Human-readable name for the queue.
* ``detail`` (String): Detailed description of the queue's purpose.
* ``routing_method`` (enum string): The queue's call routing method for selecting agents. See :ref:`Routing Method <queue-struct-queue-routing-method>`.
* ``tag_ids`` (Array of UUID): List of tag IDs that agents must match to receive calls from this queue. Each ID is obtained from ``GET /tags``. Agents must have **all** listed tags (AND logic).
* ``wait_actions`` (Array of Object): List of flow actions executed while callers wait in the queue. Each action follows the :ref:`Action <flow-struct-action>` structure. These loop until an agent is found or the wait timeout is reached.
* ``wait_timeout`` (Integer): Maximum time in milliseconds a caller can wait in the queue before being removed. Set to ``0`` for no timeout (wait indefinitely).
* ``service_timeout`` (Integer): Maximum time in milliseconds a caller and agent can talk before the call is ended. Set to ``0`` for no timeout (talk indefinitely).
* ``wait_queue_call_ids`` (Array of UUID): List of queuecall IDs currently in the waiting state. Each ID can be used with ``GET /queuecalls/{id}`` to retrieve details. Read-only, managed by the system.
* ``service_queue_call_ids`` (Array of UUID): List of queuecall IDs currently in the service state (connected to an agent). Each ID can be used with ``GET /queuecalls/{id}``. Read-only, managed by the system.
* ``total_incoming_count`` (Integer): Total number of calls that have entered this queue. Read-only.
* ``total_serviced_count`` (Integer): Total number of calls that were successfully connected to an agent. Read-only.
* ``total_abandoned_count`` (Integer): Total number of calls that left the queue without being serviced. Read-only.
* ``total_waittime`` (Integer): Sum of all calls' waiting time in milliseconds. Read-only.
* ``total_service_duration`` (Integer): Sum of all calls' service (agent conversation) time in milliseconds. Read-only.
* ``tm_create`` (string, ISO 8601): Timestamp when the queue was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any queue property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the queue was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   The ``wait_timeout`` and ``service_timeout`` fields are in **milliseconds**. A 5-minute wait timeout should be ``300000``, not ``300``. Setting either to ``0`` disables that timeout entirely.

.. _queue-struct-queue-routing-method:

Routing Method
--------------
Defines how the queue selects an agent when multiple matching agents are available.

======== ================
Type     Description
======== ================
random   Selects a random agent from the pool of available agents that match all required tags.
======== ================

