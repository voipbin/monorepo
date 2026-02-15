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

.. _billing_account-struct-allowance:

Allowance
---------

.. code::

    {
        "id": "a1b2c3d4-1234-5678-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "account_id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        "cycle_start": "2024-01-01T00:00:00Z",
        "cycle_end": "2024-02-01T00:00:00Z",
        "tokens_total": 1000,
        "tokens_used": 350,
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z",
        "tm_delete": "9999-01-01T00:00:00Z"
    }

* id: Allowance cycle's id.
* customer_id: Customer's id.
* account_id: Billing account's id.
* cycle_start: Start of the billing cycle (beginning of the month).
* cycle_end: End of the billing cycle (beginning of the next month).
* tokens_total: Total tokens allocated for this cycle (determined by plan tier).
* tokens_used: Tokens consumed so far this cycle.
