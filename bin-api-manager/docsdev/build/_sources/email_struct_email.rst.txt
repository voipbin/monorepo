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

* id: Email's ID.
* customer_id: Customer's ID.
* *source*: Source address info. See detail :ref:`here <common-struct-address-address>`.
* *destinations*: List of destination addresses info. See detail :ref:`here <common-struct-address-address>`.
* status: Email's deliverence status. See detail :ref:`here <email-struct-email-status>`.
* subject: Email's subject.
* content: Email's content.
* attachments: List of attachments. See detail :ref:`here <email-struct-attachment>`.

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

========== ========================
Status     Description
========== ========================
""         None
initiated  The email has been initiated.
processed  The email has been received is being processed.
delivered  The email has been successfully delivered to the recipient's inbox (or spam folder).
========== ========================

