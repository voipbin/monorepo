.. _email-struct-email:

Email
========

.. _email-struct-email-email:

Email
--------

.. code::

    {
        "id": "<uuid>",
        "customer_id": "<uuid>",
        "source": {
            ...
        },
        "destinations": [
            {
                ...
            },
            ...
        ],
        "status": "<string>",
        "subject": "<string>",
        "content": "<string>",
        "attachments": [
            {
                ...
            },
            ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    },

* ``id`` (UUID): The email's unique identifier. Returned when creating via ``POST /emails`` or listing via ``GET /emails``.
* ``customer_id`` (UUID): The customer who owns this email. Obtained from ``GET /customers``.
* ``source`` (Object): Source address info. See :ref:`Address <common-struct-address-address>`.
* ``destinations`` (Array of Object): List of destination addresses. See :ref:`Address <common-struct-address-address>`.
* ``status`` (enum string): The email's delivery status. See :ref:`Status <email-struct-email-status>`.
* ``subject`` (String): The email's subject line.
* ``content`` (String): The email's body content.
* ``attachments`` (Array of Object): List of attachments. See :ref:`Attachment <email-struct-attachment>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the email was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last status update.
* ``tm_delete`` (string, ISO 8601): Timestamp of deletion (soft delete).

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the email has not been deleted.

Example
+++++++

.. code::

    {
        "id": "1f25e6c9-6709-44d1-b93e-a5f1c5f80411",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "source": {
            "type": "email",
            "target": "service@voipbin.net",
            "target_name": "voipbin service",
            "name": "",
            "detail": ""
        },
        "destinations": [
            {
                "type": "email",
                "target": "pchero21@gmail.com",
                "target_name": "",
                "name": "",
                "detail": ""
            }
        ],
        "status": "delivered",
        "subject": "test email 7",
        "content": "test email from voipbin.",
        "attachments": [],
        "tm_create": "2025-03-14 19:04:01.160250",
        "tm_update": "2025-03-14 19:04:11.509512",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _email-struct-email-status:

Status
------
Email's status.

+------------+------------------------------------------------------------------+
| Status     | Description                                                      |
+============+==================================================================+
| ``""``     | No status set. Initial default before processing begins.         |
+------------+------------------------------------------------------------------+
| initiated  | The email has been created and accepted for processing.          |
+------------+------------------------------------------------------------------+
| processed  | The email is being processed and routed for delivery.            |
+------------+------------------------------------------------------------------+
| delivered  | The email has been successfully delivered to the recipient's     |
|            | mail server (may end up in inbox or spam folder).                |
+------------+------------------------------------------------------------------+

