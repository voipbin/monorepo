.. _extension-struct-extension:

Extension
=========

.. _extension-struct-extension-extension:

Extension
---------

.. code::

    {
        "id": "e1491290-c61c-4349-a7ff-5890c796b61b",
        "customer_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
        "name": "office-phone-1",
        "detail": "Main office IP phone",
        "extension": "office1",
        "domain_name": "cc6a05eb-33a4-444b-bf8a-359de7d95499.registrar.voipbin.net",
        "username": "office1",
        "password": "secure-password-123",
        "direct_hash": "a1b2c3d4e5f6",
        "tm_create": "2024-01-15 10:30:00.000000",
        "tm_update": "2024-01-15 11:00:00.000000",
        "tm_delete": ""
    }

* id: Extension's id.
* customer_id: Customer's id.
* name: Extension's name.
* detail: Extension's detail description.
* extension: Extension number/identifier.
* domain_name: SIP domain for registration.
* username: SIP authentication username.
* password: SIP authentication password.
* direct_hash: Hash for direct external access. Empty string if not enabled. When set, the extension is reachable at ``sip:direct.<hash>@sip.voipbin.net``.
* tm_create: Creation timestamp.
* tm_update: Last update timestamp.
* tm_delete: Deletion timestamp.


