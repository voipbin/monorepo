.. _billing_account-struct:

Struct
======

.. _billing_account-struct-billing_account:

Billing account
---------------

.. code::

    {
        "id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "",
        "detail": "",
        "plan_type": "free",
        "balance": 69.77263,
        "payment_type": "",
        "payment_method": "",
        "tm_create": "2013-06-17 00:00:00.000000",
        "tm_update": "2023-06-30 19:18:08.466742",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

* id: Billing account's id.
* customer_id: Customer's id.
* name: Billing account's name.
* detail: Billing account's detail.
* plan_type: Plan tier of the account. Determines resource creation limits. Available values: ``free``, ``basic``, ``professional``, ``unlimited``.
* balance: Left balance. USD.
* payment_type: payment type. Reserved.
* payment_method: payment method. Reserved.
