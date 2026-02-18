.. _quickstart_queue:

Queue
=====
Queues let you route incoming calls to available agents. Callers hear hold music or messages while waiting for an agent to become available.

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart_authentication>`.
* At least one tag ID (UUID). Create tags via ``POST /tags`` or obtain from ``GET /tags``. Tags are used to match agents to queues.
* At least one agent with the matching tag assigned. Create agents via ``POST /agents`` and assign tags.

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

- ``routing_method`` (enum string): How calls are distributed to agents. One of: ``random`` (random agent selection) or ``round-robin`` (sequential agent selection).
- ``tag_ids`` (Array of UUID): Tag IDs to match agents. Obtained from the ``id`` field of ``GET /tags``. Only agents with at least one matching tag will receive calls from this queue.
- ``wait_actions`` (Array of Object): Flow actions executed while the caller waits (e.g., play messages, music). These actions loop until an agent becomes available.
- ``timeout_wait`` (Integer, milliseconds): Maximum time a caller waits in the queue before timing out. Example: ``100000`` = 100 seconds.
- ``timeout_service`` (Integer, milliseconds): Maximum time for an active call with an agent before automatic disconnect.

.. note:: **AI Implementation Hint**

   ``timeout_wait`` and ``timeout_service`` are in milliseconds, not seconds. A common mistake is setting ``timeout_wait: 100`` (0.1 seconds) instead of ``timeout_wait: 100000`` (100 seconds). Always verify the unit when setting timeouts.

For more details, see the :ref:`Queue tutorial <queue-tutorial>`.

Troubleshooting
+++++++++++++++

* **400 Bad Request:**
    * **Cause:** ``tag_ids`` contains an invalid UUID, or required fields are missing.
    * **Fix:** Verify tag IDs via ``GET /tags``. Ensure ``tag_ids`` is a non-empty array of valid UUIDs.

* **Queue created but calls are not routed to agents:**
    * **Cause:** No agents with matching tags are online or in ``available`` status.
    * **Fix:** Verify agents have matching tags via ``GET /agents``. Check that at least one agent is in ``available`` status.

* **Callers timing out immediately:**
    * **Cause:** ``timeout_wait`` is set too low. The value is in **milliseconds**, not seconds.
    * **Fix:** For a 100-second wait, set ``timeout_wait: 100000``. Setting ``100`` means only 0.1 seconds.
