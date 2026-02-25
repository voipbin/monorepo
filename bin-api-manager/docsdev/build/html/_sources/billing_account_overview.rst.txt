.. _billing_account_overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Medium
   * **Cost:** Free (reading billing data incurs no charges; adding balance is an admin-only operation)
   * **Async:** No. All billing account operations are synchronous and return immediately. Balance changes from service usage (calls, SMS) happen asynchronously as services are consumed.

VoIPBIN's Billing Account API provides balance management, token tracking, and usage monitoring for your account. The billing system uses a State+Ledger architecture where the **account** holds the live state (current balance and tokens) and the **billings** table records every transaction as an immutable ledger entry with signed deltas and post-transaction snapshots.

With the Billing Account API you can:

- Check current credit balance and token balance
- View billing history as a ledger of transactions
- Add funds to your account (admin only)
- View rate information for different service types
- Monitor usage and charges
- Track all transactions via the immutable billing ledger


How Billing Works
-----------------
VoIPBIN uses a hybrid billing model with two cost mechanisms: **token balance** and **credit balance**.

Each plan tier includes a monthly allocation of tokens that cover certain service types (virtual number calls and TTS). When tokens are exhausted, usage overflows to the credit balance. PSTN calls, SMS, email, and number purchases are always charged to the credit balance.

All monetary values are stored as **int64 micros** (1 USD = 1,000,000 micros) to prevent floating-point rounding errors.

.. note:: **AI Implementation Hint**

   The ``balance_credit`` field is in micros (int64), not dollars. To convert: divide by 1,000,000 for USD (e.g., ``69772630`` micros = $69.77). When displaying balance to users, always convert from micros to the currency unit. When estimating costs, multiply the rate in micros by the number of billable units.

**Billing Architecture**

::

    +-----------------------------------------------------------------------+
    |                     State + Ledger Architecture                       |
    +-----------------------------------------------------------------------+

    +-------------------+     +-------------------+
    |  Account (State)  |     |  Billings (Ledger)|
    |  balance_token    |     |  Immutable entries |
    |  balance_credit   |     |  with deltas and   |
    +--------+----------+     |  snapshots         |
             |                +--------+----------+
             |                         |
             | live state              | transaction history
             v                         v
    +--------+----------+     +--------+----------+
    |                   |     |                   |
    v                   v     v                   v
    +----------+  +----------+  +----------+  +----------+  +----------+  +----------+
    | VN Calls |  |   TTS    |  |PSTN Calls|  |   SMS    |  |  Email   |  | Numbers  |
    | 1 tok/min|  | 3 tok/min|  | per min  |  | per msg  |  | per msg  |  | per num  |
    +----------+  +----------+  +----------+  +----------+  +----------+  +----------+
         |             |             |             |              |             |
         | token first | token first |             |              |             |
         | then credit | then credit | credit only | credit only  | credit only | credit only
         v             v             v             v              v             v
    +---------+   +---------+   +---------+   +---------+   +---------+   +---------+
    | $0.001  |   |  $0.03  |   |  $0.01  |   |  $0.01  |   |  $0.01  |   |  $5.00  |
    | /minute |   | /minute |   | /minute |   | /message|   | /message|   | /number |
    +---------+   +---------+   +---------+   +---------+   +---------+   +---------+

**Key Components**

- **Account State**: The ``billing_accounts`` table holds the live ``balance_credit`` (int64 micros) and ``balance_token`` (int64). This is the single source of truth for current balances.
- **Billing Ledger**: The ``billing_billings`` table records every transaction as an immutable entry with signed deltas (``amount_token``, ``amount_credit``) and post-transaction snapshots (``balance_token_snapshot``, ``balance_credit_snapshot``).
- **Monthly Token Top-Up**: Tokens are replenished monthly via a cron-driven top-up process. The top-up is recorded as a ``top_up`` transaction in the ledger.
- **Token-Eligible Services**: VN calls (1 token/minute) and TTS (3 tokens/minute) consume tokens first, then overflow to credits.
- **Credit-Only Services**: PSTN calls, SMS, email, and number purchases always use credits directly.
- **Free Services**: Extension-to-extension calls and direct extension calls incur no charges.

**Token Top-Up Process**

Token replenishment happens via an automated monthly cron process:

1. **Selection**: The system selects accounts where ``tm_next_topup <= NOW()``.
2. **State Update**: The account's ``balance_token`` is set to the plan's allocation. ``tm_last_topup`` and ``tm_next_topup`` are updated.
3. **Ledger Entry**: A billing record is inserted with ``transaction_type: top_up`` and ``reference_type: monthly_allowance``, recording the positive token delta and the resulting balance snapshot.

The ``tm_next_topup`` field on the account indicates when the next top-up is scheduled.


Plan Tiers
----------
Each billing account is assigned a plan tier that determines both resource creation limits and monthly token allocations. New accounts default to the **free** tier.

**Monthly Token Allocations**

+----------------------+---------+---------+--------------+-----------+
| Plan                 | Free    | Basic   | Professional | Unlimited |
+======================+=========+=========+==============+===========+
| Tokens per month     |     100 |   1,000 |       10,000 | unlimited |
+----------------------+---------+---------+--------------+-----------+

Tokens are replenished at the scheduled top-up date. The current token balance is available in the ``balance_token`` field of the account.

**Resource Limits**

+----------------------+-------+-------+--------------+-----------+
| Resource             | Free  | Basic | Professional | Unlimited |
+======================+=======+=======+==============+===========+
| Extensions           |     5 |    50 |          500 | unlimited |
+----------------------+-------+-------+--------------+-----------+
| Agents               |     5 |    50 |          500 | unlimited |
+----------------------+-------+-------+--------------+-----------+
| Queues               |     2 |    10 |          100 | unlimited |
+----------------------+-------+-------+--------------+-----------+
| Conferences          |     2 |    10 |          100 | unlimited |
+----------------------+-------+-------+--------------+-----------+
| Trunks               |     1 |     5 |           50 | unlimited |
+----------------------+-------+-------+--------------+-----------+
| Virtual Numbers      |     5 |    50 |          500 | unlimited |
+----------------------+-------+-------+--------------+-----------+

- When a resource limit is reached, further creation of that resource type is denied.
- Only platform admins can change a customer's plan tier.
- The current plan tier is returned in the ``plan_type`` field of the billing account.


.. _billing_account_rate_structure:

Rate Structure
--------------
VoIPBIN uses per-minute billing for calls (rounded up to the next whole minute) and per-unit billing for other services.

**Token Rates**

+----------------------+------------------+----------------------------------------+
| Service              | Token Cost       | Unit                                   |
+======================+==================+========================================+
| VN Calls             | 1 token          | Per minute (ceiling-rounded)           |
+----------------------+------------------+----------------------------------------+
| TTS (Text-to-Speech) | 3 tokens         | Per minute (ceiling-rounded)           |
+----------------------+------------------+----------------------------------------+

**Credit Rates (Overflow and Credit-Only)**

All credit rates are stored internally as int64 micros.

+----------------------+------------------+------------------+-------------------------+
| Service              | Rate (USD)       | Rate (micros)    | Unit                    |
+======================+==================+==================+=========================+
| VN Calls (overflow)  | $0.001           | 1,000            | Per minute              |
+----------------------+------------------+------------------+-------------------------+
| PSTN Outgoing Calls  | $0.01            | 10,000           | Per minute              |
+----------------------+------------------+------------------+-------------------------+
| PSTN Incoming Calls  | $0.01            | 10,000           | Per minute              |
+----------------------+------------------+------------------+-------------------------+
| SMS                  | $0.01            | 10,000           | Per message             |
+----------------------+------------------+------------------+-------------------------+
| Email                | $0.01            | 10,000           | Per message             |
+----------------------+------------------+------------------+-------------------------+
| Number Purchase      | $5.00            | 5,000,000        | Per number              |
+----------------------+------------------+------------------+-------------------------+
| Number Renewal       | $5.00            | 5,000,000        | Per number              |
+----------------------+------------------+------------------+-------------------------+
| TTS (overflow)       | $0.03            | 30,000           | Per minute              |
+----------------------+------------------+------------------+-------------------------+
| Extension Calls      | Free             | 0                | No charge               |
+----------------------+------------------+------------------+-------------------------+

**How Token Consumption Works**

When a token-eligible service is used (VN call or TTS):

1. The system checks the account's ``balance_token``.
2. If tokens are available, they are consumed first.
3. If tokens are partially available, the available tokens are consumed and the remainder overflows to credits.
4. If no tokens remain, the full cost is charged to credits.

Each transaction is recorded in the billing ledger with the token and credit amounts as signed deltas.

**Rate Calculation Examples**

::

    VN Call (2 minutes 15 seconds) with tokens available:
    +--------------------------------------------+
    | Duration: 2 min 15 sec -> 3 minutes        |
    | (ceiling-rounded to next whole minute)      |
    | Token cost: 3 x 1 = 3 tokens               |
    | Credit cost: 0 micros (covered by tokens)   |
    | Ledger entry:                               |
    |   amount_token: -3                          |
    |   amount_credit: 0                          |
    +--------------------------------------------+

    VN Call (5 minutes) with NO tokens remaining:
    +--------------------------------------------+
    | Duration: 5 minutes                         |
    | Token cost: 0 (exhausted)                   |
    | Credit cost: 5 x 1,000 = 5,000 micros      |
    | Ledger entry:                               |
    |   amount_token: 0                           |
    |   amount_credit: -5000                      |
    +--------------------------------------------+

    PSTN Outgoing Call (2 minutes 30 seconds):
    +--------------------------------------------+
    | Duration: 2 min 30 sec -> 3 minutes        |
    | (ceiling-rounded to next whole minute)      |
    | Credit cost: 3 x 10,000 = 30,000 micros    |
    | Ledger entry:                               |
    |   amount_token: 0                           |
    |   amount_credit: -30000                     |
    +--------------------------------------------+

    TTS Session (1 minute 15 seconds) with tokens available:
    +--------------------------------------------+
    | Duration: 1 min 15 sec -> 2 minutes        |
    | (ceiling-rounded to next whole minute)      |
    | Token cost: 2 x 3 = 6 tokens               |
    | Credit cost: 0 micros (covered by tokens)   |
    | Ledger entry:                               |
    |   amount_token: -6                          |
    |   amount_credit: 0                          |
    +--------------------------------------------+

    Monthly Token Top-Up (Free plan):
    +--------------------------------------------+
    | Transaction type: top_up                    |
    | Reference type: monthly_allowance           |
    | Ledger entry:                               |
    |   amount_token: +100                        |
    |   amount_credit: 0                          |
    |   balance_token_snapshot: 100               |
    +--------------------------------------------+

Note: Rates are subject to change. Check the API for current pricing.


Managing Balance
----------------
Check and manage your account balance.

**Check Balance**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/billing_accounts?token=<token>'

**Response:**

.. code::

    {
        "id": "billing-uuid-123",
        "customer_id": "customer-uuid-456",
        "plan_type": "free",
        "balance_credit": 150500000,
        "balance_token": 70,
        "tm_last_topup": "2024-01-01T00:00:00Z",
        "tm_next_topup": "2024-02-01T00:00:00Z",
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z"
    }

The ``balance_credit`` is in micros (150500000 = $150.50). The ``balance_token`` is the current token count.

**View Billing Ledger**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/billings?token=<token>&page_size=10'

Returns a paginated list of billing ledger entries showing all transactions (usage, top-ups, adjustments).

**Add Balance (Admin Only)**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/billing_accounts/<account-id>/balance_add_force?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "balance": 100.00
        }'

Note: Balance addition is restricted to users with admin permissions for security.


Balance Lifecycle
-----------------
Account balance changes through specific operations. Token balances are replenished monthly via the automated top-up process.

**Balance and Token Flow**

::

    +-------------------+     +-------------------+
    | Add Balance       |     | Monthly Top-Up    |
    | (admin only)      |     | (automated cron)  |
    +--------+----------+     +--------+----------+
             |                         |
             v                         v
    +-------------------+     +-------------------+
    |  balance_credit   |     |  balance_token    |
    |  150,500,000      |     |      100          |
    +--------+----------+     +--------+----------+
             |                         |
             | PSTN calls, SMS,        | VN calls,
             | email, numbers,         | TTS
             | overflow                |
             v                         v
    +-------------------+     +-------------------+
    |   Credit Charges  |     |  Token Usage      |
    | - 30000 PSTN call |     | - 3 tokens call   |
    | - 10000 SMS       |     | - 6 tokens TTS    |
    | - 5000000 number  |
    +--------+----------+     +--------+----------+
             |                         |
             v                         | exhausted
    +-------------------+              |
    |  Updated Balance  |<-------------+
    |  145,460,000      |   overflow charges
    +-------------------+

    All transactions recorded in billing ledger
    with signed deltas and balance snapshots.


Balance Monitoring
------------------
Monitor balance and token usage to avoid service interruption.

**Balance Check Flow**

::

    Before Service Execution:
    +--------------------------------------------+
    | 1. Identify service type                   |
    | 2. If token-eligible: check balance_token  |
    |    - If tokens available -> proceed         |
    |    - If no tokens -> check balance_credit   |
    | 3. If credit-only: check balance_credit    |
    |    - If balance >= cost -> proceed          |
    |    - If balance < cost -> deny service      |
    +--------------------------------------------+


Common Scenarios
----------------

**Scenario 1: Token-Based Monthly Usage**

Track token consumption via the billing ledger.

::

    Monthly Usage (Free Plan, 100 tokens):
    +--------------------------------------------+
    | Week 1:                                    |
    | - 10 VN calls (avg 3 min) = 30 tokens      |
    | - 2 TTS sessions (avg 5 min) = 30 tokens   |
    | - 5 SMS = 50,000 micros credit (credit-only)|
    | balance_token: 40                          |
    |                                            |
    | Week 2:                                    |
    | - 20 VN calls (avg 2 min) = 40 tokens      |
    | balance_token: 0                           |
    |                                            |
    | Week 3 (tokens exhausted):                 |
    | - 5 VN calls (avg 3 min) = overflow        |
    |   5 x 3 x 1,000 = 15,000 micros credit    |
    | - 2 SMS = credit                           |
    |   2 x 10,000 = 20,000 micros credit        |
    +--------------------------------------------+

**Scenario 2: Mixed Token and Credit Usage**

Plan for costs across token-eligible and credit-only services.

::

    Campaign: Customer Outreach (Basic plan, 1000 tokens)
    +----------------------------------------------+
    | VN Calls: 200 calls (avg 3 min)              |
    | - Tokens needed: 200 x 3 = 600               |
    | - If 400 tokens remaining:                   |
    |   - 400 tokens consumed (= 400 min covered)  |
    |   - 200 min overflow x 1,000 micros/min      |
    |     = 200,000 micros ($0.20)                 |
    |                                              |
    | PSTN Calls: 50 calls (avg 2 min)             |
    | - Credit: 50 x 2 x 10,000 = 1,000,000 micros|
    |   ($1.00)                                    |
    |                                              |
    | SMS: 100 messages (credit-only)              |
    | - Credit: 100 x 10,000 = 1,000,000 micros   |
    |   ($1.00)                                    |
    |                                              |
    | Total credit: 2,200,000 micros ($2.20)       |
    +----------------------------------------------+


Best Practices
--------------

**1. Token Monitoring**

- Check ``balance_token`` on the account regularly
- Plan upgrades to higher tiers if tokens are consistently exhausted before the next top-up
- Review billing ledger entries to understand consumption patterns

**2. Balance Monitoring**

- Maintain credit balance for PSTN calls, number purchases, and token overflow
- Set up low balance alerts
- Plan for buffer above minimum needed

**3. Cost Estimation**

- Separate estimates into token-eligible and credit-only services
- Account for token overflow in budget planning
- Include retry costs in estimates
- Note all credit amounts are in micros (divide by 1,000,000 for USD)

**4. Security**

- Restrict balance add permissions to admins
- Monitor for unusual usage patterns via the billing ledger
- Review billing history regularly for anomalies

**5. Plan Selection**

- Choose plan tier based on expected VN call and TTS volume
- Compare token allocation cost vs. credit-only cost at each tier
- Consider upgrading if monthly overflow charges are significant


Troubleshooting
---------------

**Balance Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Services being denied     | Check ``balance_credit`` and                   |
|                           | ``balance_token`` on the account;              |
|                           | add funds or upgrade plan if needed            |
+---------------------------+------------------------------------------------+
| Cannot add balance        | Verify admin permissions; check API token       |
|                           | validity                                       |
+---------------------------+------------------------------------------------+
| Balance not updating      | Allow time for transaction processing; check    |
|                           | billing ledger for recent entries               |
+---------------------------+------------------------------------------------+

**Token Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Tokens exhausted early    | Review billing ledger for usage patterns;       |
|                           | consider upgrading plan tier for more tokens    |
+---------------------------+------------------------------------------------+
| Unexpected overflow       | Check ``balance_token`` on account; VN calls    |
|                           | and TTS consume tokens first                   |
+---------------------------+------------------------------------------------+
| Tokens not replenishing   | Check ``tm_next_topup`` on the account;         |
|                           | tokens are replenished by the automated cron    |
+---------------------------+------------------------------------------------+

**Billing Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Unexpected charges        | Review billing ledger entries; check if tokens  |
|                           | were exhausted causing credit overflow          |
+---------------------------+------------------------------------------------+
| Rate seems wrong          | Verify current rate structure; note all credit  |
|                           | amounts are in micros (divide by 1,000,000)    |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Customer Overview <customer-overview>` - Account management
- :ref:`Call Overview <call-overview>` - Voice call costs
- :ref:`Message Overview <message-overview>` - SMS costs
- :ref:`Number Overview <number-overview>` - Number provisioning costs
