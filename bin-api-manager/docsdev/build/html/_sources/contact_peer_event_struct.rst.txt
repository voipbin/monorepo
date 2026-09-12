.. _contact-peer-event-struct:

Structures
==========

.. _contact-peer-event-struct-peerevent:

PeerEvent
---------

.. code::

    {
        "timestamp": "<string>",
        "customer_id": "<string>",
        "publisher": "<string>",
        "event_type": "<string>",
        "reference_id": "<string>",
        "direction": "<string>",
        "peer": {<Address>},
        "local": {<Address>},
        "data": {}
    }

* ``timestamp`` (string, ISO 8601): The event's origin timestamp. Used as the pagination cursor.
* ``customer_id`` (UUID): The customer who owns this row. Obtained from ``GET /customers``.
* ``publisher`` (enum string): Synthetic derived label for the originating channel. See :ref:`Publisher <contact-peer-event-struct-publisher>`.
* ``event_type`` (String): The originating event type (e.g. ``call_hangup``, ``conversation_message_created``).
* ``reference_id`` (UUID): The ``call_id``, ``conversation_message_id``, or ``conversation_id`` this row was projected from.
* ``direction`` (enum string): Direction of the interaction from the platform's perspective. See :ref:`Direction <contact-peer-event-struct-direction>`. Empty string for rows with no direction concept (e.g. conversation-parent rows).
* ``peer`` (Object): Remote endpoint, structurally identical to ``commonaddress.Address``, the platform's standard cross-service address type (``type``/``target``/``target_name``/``name``/``detail``) — not the same shape as :ref:`Contact's Address <contact-struct-contact-address>` (which additionally carries ``id``/``is_primary``/``tm_create`` and lacks ``target_name``/``name``/``detail``). ``type`` may be an internal-resource value (``agent``, ``ai``, ``conference``, ``sip``) not present in Contact's address types.
* ``local`` (Object): The customer's own endpoint, same shape as ``peer``.
* ``data`` (Object): The original webhook payload, verbatim.

.. _contact-peer-event-struct-publisher:

Publisher
^^^^^^^^^

+-----------------------+-------------------------------------------+
| Value                 | Description                               |
+=======================+===========================================+
| call                  | Originated from call-manager              |
+-----------------------+-------------------------------------------+
| conversation_message  | A single message within a conversation    |
+-----------------------+-------------------------------------------+
| conversation          | The conversation-parent record            |
+-----------------------+-------------------------------------------+

.. _contact-peer-event-struct-direction:

Direction
^^^^^^^^^

+----------+---------------------------------------------------------------------+
| Value    | Description                                                         |
+==========+=====================================================================+
| incoming | Received by the platform                                            |
+----------+---------------------------------------------------------------------+
| outgoing | Sent by the platform                                                |
+----------+---------------------------------------------------------------------+
| (empty)  | No direction concept for this row (e.g. conversation-parent rows)   |
+----------+---------------------------------------------------------------------+
