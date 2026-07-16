.. _webchat-struct-message:

Message
=======

.. _webchat-struct-message-message:

Message
-------
Message struct

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "widget_id": "<string>",
        "session_id": "<string>",
        "direction": "<string>",
        "status": "<string>",
        "sender_id": "<string>",
        "text": "<string>",
        "tm_create": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The message's unique identifier. Returned from ``POST /webchat_messages`` or ``GET /webchat_messages``.
* ``customer_id`` (UUID): The customer who owns the parent widget. Obtained from the ``id`` field of ``GET /customers``.
* ``widget_id`` (UUID): The widget this message belongs to (denormalized from the session). Obtained from the ``id`` field of ``GET /webchat_widgets``.
* ``session_id`` (UUID): The session this message belongs to. Obtained from the ``id`` field of ``GET /webchat_sessions``.
* ``direction`` (enum string): The message's direction. See :ref:`Direction <webchat-struct-message-direction>`.
* ``status`` (enum string): The message's delivery status. See :ref:`Status <webchat-struct-message-status>`.
* ``sender_id`` (UUID, optional): The agent ID for an agent-authored outbound reply. Empty for visitor-authored inbound messages and for Flow/AI-originated outbound messages.
* ``text`` (String): The text content of the message.
* ``tm_create`` (string, ISO 8601): Timestamp when the message was created.
* ``tm_delete`` (string, ISO 8601): Timestamp when the message was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Only ``inbound`` messages can trigger the widget's Flow, and only the **first** inbound message on a given session does so -- subsequent inbound messages on the same session are delivered to the already-running Flow/conversation without re-triggering it. ``outbound`` messages (agent replies, Flow/AI responses) never trigger a Flow.

.. _webchat-struct-message-direction:

Direction
---------
Defines who sent the message.

============= ================
Type          Description
============= ================
inbound       Sent by the visitor to VoIPBin.
outbound      Sent by VoIPBin (Flow, AI, or an agent) to the visitor.
============= ================

.. _webchat-struct-message-status:

Status
------
Defines the message's delivery status.

============= ================
Type          Description
============= ================
sent          The message has been persisted and (for outbound) delivered over the WebSocket connection.
delivered     The message was confirmed received (reserved for future delivery-confirmation support).
failed        The message could not be delivered.
============= ================
