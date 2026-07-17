.. _webchat-struct-session:

Session
=======

.. _webchat-struct-session-session:

Session
-------
Session struct

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "widget_id": "<string>",
        "status": "<string>",
        "tm_last_activity": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_end": "<string>"
    }

* ``id`` (UUID): The session's unique identifier, and the visitor's continuity token for this browsing session. Returned from ``POST /webchat_sessions`` or ``GET /webchat_sessions``.
* ``customer_id`` (UUID): The customer who owns the parent widget. Obtained from the ``id`` field of ``GET /customers``.
* ``widget_id`` (UUID): The widget this session belongs to. Obtained from the ``id`` field of ``GET /webchat_widgets``.
* ``status`` (enum string): The session's status. See :ref:`Status <webchat-struct-session-status>`.
* ``tm_last_activity`` (string, ISO 8601): Timestamp of the most recent message on this session. Used to determine idle timeout.
* ``tm_create`` (string, ISO 8601): Timestamp when the session was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any session property.
* ``tm_end`` (string, ISO 8601): Timestamp when the session was ended (explicitly via ``POST /webchat_sessions/{id}/end`` or automatically via idle timeout).

.. note:: **AI Implementation Hint**

   ``Session.id`` is the value your frontend must persist (e.g. in browser storage) and reuse across page loads within the same visit -- it is the sole visitor identity webchat uses; there is no separate customer-facing "visitor" or "contact" resource for anonymous webchat traffic.

.. _webchat-struct-session-status:

Status
------
Defines the session's lifecycle state.

============= ================
Type          Description
============= ================
active        The session is live; ``POST /webchat_messages`` is accepted.
ended         The session has ended; a new session must be created to continue the conversation.
============= ================
