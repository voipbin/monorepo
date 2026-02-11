.. _extension-struct-extension:

Extension
=========

.. _extension-struct-extension-extension:

Extension
---------

.. code::

    {
        "id": "e1491290-c61c-4349-a7ff-5890c796b61b",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "test domain",
        "detail": "test domain creation",
        "domain_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
        "extension": "test12",
        "domain_name": "5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
        "username": "test12",
        "password": "27a4d0f2-757c-11eb-bc8f-4f045857b89c",
        "direct_hash": "",
        "tm_create": "2021-03-21 11:24:26.485161",
        "tm_update": "",
        "tm_delete": ""
    },

* id: Extension's id.
* customer_id: Customer's id.
* name: Extension's name.
* detail: Extension's detail description.
* domain_id: Domain's id.
* extension: Extension name/number.
* domain_name: SIP registration domain.
* username: SIP authentication username.
* password: SIP authentication password.
* direct_hash: Hash for direct extension access. Empty string when direct is disabled. When enabled, this hash forms the direct SIP URI: ``sip:direct.<hash>@sip.voipbin.net``.
* tm_create: Timestamp of creation.
* tm_update: Timestamp of last update.
* tm_delete: Timestamp of deletion.


