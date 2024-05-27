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

* *id*: Queue's ID.
* *name*: Queue's name.
* *detail*: Queue's detail description.
* *routing_method*: Define the queue's call routing method. See detail :ref:`here <queue-struct-queue-routing-method>`.
* *tag_ids*: List of tags.
* *wait_actions*: List of actions for waiting calls.
* *wait_timeout*: Timeout for waiting(ms). If it sets to 0, no timeout.
* *service_timeout*: Timeout for service(talk with agent. ms). If it sets to 0, no timeout.
* *wait_queue_call_ids*: List of waiting call ids.
* *service_queue_call_ids*: List of service call ids.
* *total_incoming_count*: Number of joined calls.
* *total_serviced_count*: Number of serviced calls.
* *total_abandoned_count*: Number of abandoned calls.
* *total_waittime*: Sum of all call's waitting time(ms).
* *total_service_duration*: Sum of all call's service time(ms).

.. _queue-struct-queue-routing-method:

Routing Method
--------------
Define the queue's queued call routing method to the agent if the number of available agent is more than 2.

======== ================
Type     Description
======== ================
random   Pick the agent randomly.
======== ================

