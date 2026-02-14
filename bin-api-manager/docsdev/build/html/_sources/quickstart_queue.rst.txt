.. _quickstart_queue:

Queue
=====
Queues let you route incoming calls to available agents. Callers hear hold music or messages while waiting for an agent to become available.

Create a queue
--------------
This example creates a queue that routes calls randomly to agents matching a specific tag. While callers wait, they hear a text-to-speech greeting followed by a 1-second pause (looped):

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/queues?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "support queue",
            "detail": "Customer support queue",
            "routing_method": "random",
            "tag_ids": ["<your-tag-id>"],
            "wait_actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "Thank you for calling. Please wait while we connect you to an agent.",
                        "language": "en-US"
                    }
                },
                {
                    "type": "sleep",
                    "option": {
                        "duration": 1000
                    }
                }
            ],
            "timeout_wait": 100000,
            "timeout_service": 10000000
        }'

The response includes the created queue with its ID and configuration:

.. code::

    {
        "id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "name": "support queue",
        "detail": "Customer support queue",
        "routing_method": "random",
        "tag_ids": ["<your-tag-id>"],
        ...
    }

Key parameters:

- **routing_method**: How calls are distributed to agents (``random``, ``round-robin``).
- **tag_ids**: Agent tags to match. Only agents with matching tags will receive calls from this queue.
- **wait_actions**: Actions executed while the caller waits (e.g., play messages, music).
- **timeout_wait**: Maximum time (ms) a caller waits in the queue before timing out.
- **timeout_service**: Maximum time (ms) for an active call with an agent.

For more details, see the :ref:`Queue tutorial <queue-main>`.
