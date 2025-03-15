.. _email-tutorial: email-tutorial

Tutorial
========

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
            },
            ...
        ],
        "next_page_token": "2025-03-14 18:04:41.998152"
    }
