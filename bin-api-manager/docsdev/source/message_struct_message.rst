.. _message-struct-message:

Message
========

.. _message-struct-message-message:

Message
-------
Message struct

.. code::

    {
        "id": "<string>",
        "type": "sms",
        "source": {
            "type": "tel",
            "target": "+15551234567",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "targets": [
            {
                "destination": {
                    "type": "tel",
                    "target": "+15559876543",
                    "target_name": "",
                    "name": "",
                    "detail": ""
                },
                "status": "sent",
                "parts": 1,
                "tm_update": "2022-03-13 15:11:06.497184184"
            }
        ],
        "text": "Hello, this is test message.",
        "direction": "outbound",
        "tm_create": "2022-03-13 15:11:05.235717",
        "tm_update": "2022-03-13 15:11:06.497278",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

* ``id`` (UUID): The message's unique identifier. Returned when creating via ``POST /messages`` or listing via ``GET /messages``.
* ``type`` (enum string): The message type. See :ref:`Type <message-struct-message-type>`.
* ``source`` (Object): Source address info. See :ref:`Address <common-struct-address-address>`.
* ``targets`` (Array of Object): List of delivery targets with per-destination status. See :ref:`Target <message-struct-message-target>`.
* ``text`` (String): The message body text content.
* ``direction`` (enum string): Whether the message is inbound or outbound. See :ref:`Direction <message-struct-message-direction>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the message was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last status update.
* ``tm_delete`` (string, ISO 8601): Timestamp of deletion (soft delete).

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the message has not been deleted.

.. _message-struct-message-target:

Target
------
Target struct

.. code::

    {
        "destination": {
            "type": "tel",
            "target": "+15559876543",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "status": "sent",
        "parts": 1,
        "tm_update": "2022-03-13 15:11:06.497184184"
    }

* ``destination`` (Object): Destination address info. See :ref:`Address <common-struct-address-address>`.
* ``status`` (enum string): Delivery status for this specific destination (e.g., ``sending``, ``sent``, ``delivered``, ``failed``).
* ``parts`` (Integer): Number of message segments. Long SMS messages are split into multiple parts (153 characters each for GSM-7 encoding).
* ``tm_update`` (string, ISO 8601): Timestamp of the last status update for this target.

.. _message-struct-message-type:

Type
----
Message's type.

+------------+------------------------------------------------------------------+
| Type       | Description                                                      |
+============+==================================================================+
| sms        | Standard SMS text message. Limited to 160 characters per segment |
|            | (GSM-7 encoding) or 70 characters (Unicode encoding).           |
+------------+------------------------------------------------------------------+

.. _message-struct-message-direction:

Direction
---------
Message's direction.

+------------+------------------------------------------------------------------+
| Direction  | Description                                                      |
+============+==================================================================+
| inbound    | Incoming message received from an external sender to your        |
|            | VoIPBIN number. Delivered to your application via webhook.       |
+------------+------------------------------------------------------------------+
| outbound   | Outgoing message sent from your application via the VoIPBIN API  |
|            | to an external recipient.                                        |
+------------+------------------------------------------------------------------+
