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

Each plan tier includes a monthly allocation of tokens that cover certain service types (virtual number calls and SMS). When tokens are exhausted, usage overflows to the credit balance. PSTN calls and number purchases are always charged to the credit balance.

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
    +----------+   +----------+   +----------+   +----------+
    | VN Calls |   |   SMS    |   |PSTN Calls|   | Numbers  |
    | 1 tok/min|   |10 tok/msg|   | per min  |   | per num  |
    +----------+   +----------+   +----------+   +----------+
         |              |              |              |
         | token first  | token first  |              |
         | then credit  | then credit  | credit only  | credit only
         v              v              v              v
    +---------+    +---------+    +---------+    +---------+
    |$0.001   |    | $0.008  |    |$0.006 ot|    |  $5.00  |
    | /minute |    | /message|    |$0.0045 in|   | /number |
    +---------+    +---------+    +---------+    +---------+

**Key Components**

- **Account State**: The ``billing_accounts`` table holds the live ``balance_credit`` (int64 micros) and ``balance_token`` (int64). This is the single source of truth for current balances.
- **Billing Ledger**: The ``billing_billings`` table records every transaction as an immutable entry with signed deltas (``amount_token``, ``amount_credit``) and post-transaction snapshots (``balance_token_snapshot``, ``balance_credit_snapshot``).
- **Monthly Token Top-Up**: Tokens are replenished monthly via a cron-driven top-up process. The top-up is recorded as a ``top_up`` transaction in the ledger.
- **Token-Eligible Services**: VN calls (1 token/minute) and SMS (10 tokens/message) consume tokens first, then overflow to credits.
- **Credit-Only Services**: PSTN calls and number purchases always use credits directly.
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
| Tokens per month     |   1,000 |  10,000 |      100,000 | unlimited |
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
| SMS Messages         | 10 tokens        | Per message                            |
+----------------------+------------------+----------------------------------------+

**Credit Rates (Overflow and Credit-Only)**

All credit rates are stored internally as int64 micros.

+----------------------+------------------+------------------+-------------------------+
| Service              | Rate (USD)       | Rate (micros)    | Unit                    |
+======================+==================+==================+=========================+
| VN Calls (overflow)  | $0.001           | 1,000            | Per minute              |
+----------------------+------------------+------------------+-------------------------+
| PSTN Outgoing Calls  | $0.0060          | 6,000            | Per minute              |
+----------------------+------------------+------------------+-------------------------+
| PSTN Incoming Calls  | $0.0045          | 4,500            | Per minute              |
+----------------------+------------------+------------------+-------------------------+
| SMS (overflow)       | $0.008           | 8,000            | Per message             |
+----------------------+------------------+------------------+-------------------------+
| Number Purchase      | $5.00            | 5,000,000        | Per number              |
+----------------------+------------------+------------------+-------------------------+
| Number Renewal       | $5.00            | 5,000,000        | Per number              |
+----------------------+------------------+------------------+-------------------------+
| Extension Calls      | Free             | 0                | No charge               |
+----------------------+------------------+------------------+-------------------------+

**How Token Consumption Works**

When a token-eligible service is used (VN call or SMS):

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
    | Credit cost: 3 x 6,000 = 18,000 micros     |
    | Ledger entry:                               |
    |   amount_token: 0                           |
    |   amount_credit: -18000                     |
    +--------------------------------------------+

    Monthly Token Top-Up (Free plan):
    +--------------------------------------------+
    | Transaction type: top_up                    |
    | Reference type: monthly_allowance           |
    | Ledger entry:                               |
    |   amount_token: +1000                       |
    |   amount_credit: 0                          |
    |   balance_token_snapshot: 1000              |
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
        "balance_token": 650,
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
    |  150,500,000      |     |    1,000          |
    +--------+----------+     +--------+----------+
             |                         |
             | PSTN calls,             | VN calls,
             | numbers,                | SMS
             | overflow                |
             v                         v
    +-------------------+     +-------------------+
    |   Credit Charges  |     |  Token Usage      |
    | - 18000 PSTN call |     | - 3 tokens call   |
    | - 5000000 number  |     | - 10 tokens SMS   |
    +--------+----------+     +--------+----------+
             |                         |
             v                         | exhausted
    +-------------------+              |
    |  Updated Balance  |<-------------+
    |  145,082,000      |   overflow charges
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

    Monthly Usage (Free Plan, 1000 tokens):
    +--------------------------------------------+
    | Week 1:                                    |
    | - 50 VN calls (avg 3 min) = 150 tokens     |
    | - 20 SMS = 200 tokens                      |
    | balance_token: 650                         |
    |                                            |
    | Week 2:                                    |
    | - 40 VN calls (avg 2 min) = 80 tokens      |
    | - 30 SMS = 300 tokens                      |
    | balance_token: 270                         |
    |                                            |
    | Week 3:                                    |
    | - 30 VN calls (avg 3 min) = 90 tokens      |
    | - 15 SMS = 150 tokens                      |
    | balance_token: 30                          |
    |                                            |
    | Week 4 (tokens nearly exhausted):          |
    | - 10 VN calls (avg 3 min) = 30 tokens      |
    |   (uses last tokens)                       |
    | - 5 SMS = overflow to credit               |
    |   5 x 8,000 = 40,000 micros credit charge  |
    +--------------------------------------------+

**Scenario 2: Mixed Token and Credit Usage**

Plan for costs across token-eligible and credit-only services.

::

    Campaign: Customer Outreach
    +--------------------------------------------+
    | VN Calls: 200 calls (avg 3 min)            |
    | - Tokens needed: 200 x 3 = 600             |
    | - If 400 tokens available:                 |
    |   - 400 tokens consumed                    |
    |   - 200 overflow x 3 min x 1,000 micros    |
    |     = 600,000 micros ($0.60)               |
    |                                            |
    | PSTN Calls: 50 calls (avg 2 min)           |
    | - Credit: 50 x 2 x 6,000 = 600,000 micros |
    |   ($0.60)                                  |
    |                                            |
    | SMS: 100 messages (no tokens remaining)    |
    | - Credit: 100 x 8,000 = 800,000 micros    |
    |   ($0.80)                                  |
    |                                            |
    | Total credit: 2,000,000 micros ($2.00)     |
    +--------------------------------------------+


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

- Choose plan tier based on expected VN call and SMS volume
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
|                           | and SMS consume tokens first                   |
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
