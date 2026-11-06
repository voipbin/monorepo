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
        "customer_id": "<string>",
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
        "medias": ["<string>"],
        "direction": "outbound",
        "tm_create": "2022-03-13 15:11:05.235717",
        "tm_update": "2022-03-13 15:11:06.497278",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

* ``id`` (UUID): The message's unique identifier. Returned when creating via ``POST /messages`` or listing via ``GET /messages``.
* ``customer_id`` (UUID): The customer's ID. Obtained from the ``id`` field of ``GET /customers``.
* ``type`` (enum string): The message type. See :ref:`Type <message-struct-message-type>`.
* ``source`` (Object): Source address info. See :ref:`Address <common-struct-address-address>`.
* ``targets`` (Array of Object): List of delivery targets with per-destination status. See :ref:`Target <message-struct-message-target>`.
* ``text`` (String): The message body text content.
* ``medias`` (Array of UUID): The ids of the media files attached to the message, from the ``POST /storage_files`` response. Present only when the message has medias (type ``mms``); omitted for an SMS. For an inbound MMS the list is filled after the message was delivered, see :ref:`Receiving an MMS <message-overview-receiving-mms>`.
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
* ``status`` (enum string): Delivery status for this specific destination. One of ``queued``, ``sent``, ``delivered``, ``gw_timeout``, ``dlr_timeout``, ``failed`` (outbound) or ``received``, ``delivered`` (inbound).
* ``parts`` (Integer): Number of segments of the text, counted by VoIPBin from the GSM-7 and UCS-2 rules (160 or 70 characters in one segment, 153 or 67 when concatenated). It is information and not the charged unit: the charge is one message for each destination.
* ``tm_update`` (string, ISO 8601): Timestamp of the last status update for this target.

.. _message-struct-message-type:

Type
----
Message's type.

.. list-table::
   :header-rows: 1

   * - Type
     - Description
   * - sms
     - A text message of one segment: up to 160 characters (GSM-7 encoding) or 70 characters (Unicode encoding).
   * - mms
     - A message with one or more media attachments (images, audio, PDF), listed in ``medias``, or a text without media that is longer than one segment. A message sent with ``medias`` is returned with type ``mms``, and so is a received message that carries media.


.. _message-struct-message-direction:

Direction
---------
Message's direction.

.. list-table::
   :header-rows: 1

   * - Direction
     - Description
   * - inbound
     - Incoming message received from an external sender to your VoIPBin number. Delivered to your application via webhook.
   * - outbound
     - Outgoing message sent from your application via the VoIPBin API to an external recipient.

