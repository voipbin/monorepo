.. _email-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with emails, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A verified customer identity. VoIPBIN gates email sending on your customer account's identity verification status, not on a sender domain -- unverified accounts cannot send.
* Recipient email addresses for sending.

.. note:: **AI Implementation Hint**

   Sending emails incurs charges. The ``source`` address is fixed to VoIPBIN's own platform address and is not set by the caller. The ``destinations`` field uses the :ref:`Address <common-struct-address-address>` format with ``type`` set to ``email``. ``content`` is plain text only. Email status follows this lifecycle: ``initiated`` -> ``processed`` -> ``delivered``. Additional statuses include ``open``, ``click``, ``bounce``, ``dropped``, ``deferred``, ``unsubscribe``, and ``spamreport``.

Send an email
-------------

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/emails?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "destinations": [
            {
                "type": "email",
                "target": "recipient@example.com"
            }
        ],
        "subject": "Hello from VoIPBIN",
        "content": "This is a test email sent via the VoIPBIN API.",
        "attachments": []
    }'

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
                "target": "recipient@example.com",
                "target_name": "",
                "name": "",
                "detail": ""
            }
        ],
        "status": "initiated",
        "subject": "Hello from VoIPBIN",
        "content": "This is a test email sent via the VoIPBIN API.",
        "attachments": [],
        "tm_create": "2025-03-14 19:04:01.160250",
        "tm_update": "2025-03-14 19:04:01.160250",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Get list of emails
------------------

Example
+++++++

.. code::

    $ curl --location 'https://api.voipbin.net/v1.0/emails?accesskey=your_accesskey'

    {
        "result": [
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
                        "target": "user@example.com",
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
            },
            ...
        ],
        "next_page_token": "2025-03-14 18:04:41.998152"
    }
