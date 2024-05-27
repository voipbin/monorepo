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
            "target": "+821028286521",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "targets": [
            {
                "destination": {
                    "type": "tel",
                    "target": "+821021656521",
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

* id: Message's ID.
* *type*: Message's type. See detail :ref:`here <message-struct-message-type>`.
* *source*: Source address info. See detail :ref:`here <common-struct-address-address>`.
* *targets*: List of targets. See detail :ref:`here <message-struct-message-target>`.
* *destinations*: List of destination addresses info. See detail :ref:`here <common-struct-address-address>`.
* *targets*: List of targets. See detail :ref:`here <message-struct-message-target>`.
* text: Message's text.
* *direction*: Message's direction. See detail :ref:`here <message-struct-message-direction>`.

.. _message-struct-message-target:

Target
------
Target struct

.. code::

    {
        "destination": {
            "type": "tel",
            "target": "+821021656521",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "status": "sent",
        "parts": 1,
        "tm_update": "2022-03-13 15:11:06.497184184"
    }

* *destination*: Destination address info. See detail :ref:`here <common-struct-address-address>`.
* status: Message's status for this destination.
* parts: Number of parted message.

.. _message-struct-message-type:

Type
----
Message's type.

=========== ============
Type        Description
=========== ============
sms         SMS.
=========== ============

.. _message-struct-message-direction:

Direction
---------
Message's direction.

=========== ============
Type        Description
=========== ============
inbound     Incoming message.
outbound    Outgoing message.
=========== ============
