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

An allowance represents a single monthly billing cycle for an account. Each cycle tracks a pool of tokens that the account can consume for token-eligible services (VN calls and SMS). When the cycle ends, a new cycle is created automatically with a fresh allocation of tokens based on the account's plan tier. Unused tokens do not carry over between cycles.

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

* id: Unique identifier of this allowance cycle.
* customer_id: Customer that owns this account.
* account_id: Billing account this allowance belongs to.
* cycle_start: Start of the billing cycle. Always the 1st of the month at 00:00:00 UTC.
* cycle_end: End of the billing cycle. Always the 1st of the following month at 00:00:00 UTC.
* tokens_total: Total tokens allocated for this cycle. Determined by the account's plan tier at cycle creation time (Free: 1,000, Basic: 10,000, Professional: 100,000). Can be adjusted by platform admins.
* tokens_used: Number of tokens consumed so far during this cycle. Incremented atomically each time a token-eligible service (VN call or SMS) is used.
* tm_create: Timestamp when this cycle was created.
* tm_update: Timestamp of the last modification (token consumption or admin adjustment).
* tm_delete: Soft-delete timestamp. Active cycles have ``9999-01-01T00:00:00Z``.

Remaining tokens for the cycle can be calculated as ``tokens_total - tokens_used``. When ``tokens_used`` reaches ``tokens_total``, subsequent token-eligible service usage overflows to the credit balance.
