.. _billing_account_overview:

Overview
========
VoIPBIN's Billing Account API provides balance management, token allowance tracking, and usage monitoring for your account. The billing system uses a hybrid model combining monthly token allowances with credit-based overflow. The API enables you to check balances, view allowance usage, add funds, and monitor consumption.

With the Billing Account API you can:

- Check current account balance and token allowance
- View monthly token usage and remaining allowance
- Add funds to your account (admin only)
- View rate information for different service types
- Monitor usage and charges
- Track billing history


How Billing Works
-----------------
VoIPBIN uses a hybrid billing model with two cost mechanisms: **token allowances** and **credit balance**.

Each plan tier includes a monthly pool of tokens that cover certain service types (virtual number calls and SMS). When tokens are exhausted, usage overflows to the credit balance. PSTN calls and number purchases are always charged to the credit balance.

**Billing Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Billing System                                |
    +-----------------------------------------------------------------------+

    +-------------------+     +-------------------+
    |  Token Allowance  |     |  Credit Balance   |
    |  (monthly pool)   |     |  (prepaid USD)    |
    +--------+----------+     +--------+----------+
             |                         |
             | covers                  | covers
             v                         v
    +--------+----------+     +--------+----------+--------+----------+
    |                   |     |                   |                   |
    v                   v     v                   v                   v
    +----------+   +----------+   +----------+   +----------+   +----------+
    | VN Calls |   |   SMS    |   |PSTN Calls|   | Numbers  |   |  Other   |
    | 1 tok/min|   |10 tok/msg|   | per min  |   | per num  |   | services |
    +----------+   +----------+   +----------+   +----------+   +----------+
         |              |              |              |              |
         | overflow     | overflow     |              |              |
         v              v              v              v              v
    +---------+    +---------+    +---------+    +---------+    +---------+
    |$0.0045  |    | $0.008  |    |$0.006 ot|    |  $5.00  |    |  varies |
    | /minute |    | /message|    |$0.0045 in|   | /number |    |         |
    +---------+    +---------+    +---------+    +---------+    +---------+

**Key Components**

- **Token Allowance**: A monthly pool of tokens included with your plan tier. Tokens cover VN (virtual number) calls and SMS messages. Each billing cycle is represented by an **allowance** record that tracks the token allocation and consumption for that month.
- **Credit Balance**: Prepaid USD balance used for PSTN calls, number purchases, and overflow when tokens are exhausted.
- **Token-Eligible Services**: VN calls (1 token/minute) and SMS (10 tokens/message) consume tokens first, then overflow to credits.
- **Credit-Only Services**: PSTN calls and number purchases always use credits directly.
- **Free Services**: Extension-to-extension calls and direct extension calls incur no charges.

**Allowance Cycle Lifecycle**

Each billing account has one active allowance cycle at a time. The cycle follows this lifecycle:

1. **Creation**: A new allowance cycle is created on the 1st of each month (or when the account is first used). The ``tokens_total`` is set based on the account's current plan tier.
2. **Consumption**: As token-eligible services are used, ``tokens_used`` increments atomically. The system checks the allowance before each service call.
3. **Overflow**: When ``tokens_used`` reaches ``tokens_total``, further token-eligible usage is charged to the credit balance at the overflow rate.
4. **Expiry**: When the cycle end date passes, the cycle becomes inactive. A new cycle is created for the next month with a fresh token allocation.

Unused tokens do **not** carry over between cycles. Each month starts with the full allocation defined by the plan tier.


Plan Tiers
----------
Each billing account is assigned a plan tier that determines both resource creation limits and monthly token allowances. New accounts default to the **free** tier.

**Monthly Token Allowances**

+----------------------+---------+---------+--------------+-----------+
| Plan                 | Free    | Basic   | Professional | Unlimited |
+======================+=========+=========+==============+===========+
| Tokens per month     |   1,000 |  10,000 |      100,000 | unlimited |
+----------------------+---------+---------+--------------+-----------+

Tokens reset at the start of each monthly billing cycle. Unused tokens do not carry over.

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

+----------------------+------------------+----------------------------------------+
| Service              | Rate (USD)       | Unit                                   |
+======================+==================+========================================+
| VN Calls (overflow)  | $0.0045          | Per minute (ceiling-rounded)           |
+----------------------+------------------+----------------------------------------+
| PSTN Outgoing Calls  | $0.0060          | Per minute (ceiling-rounded)           |
+----------------------+------------------+----------------------------------------+
| PSTN Incoming Calls  | $0.0045          | Per minute (ceiling-rounded)           |
+----------------------+------------------+----------------------------------------+
| SMS (overflow)       | $0.008           | Per message                            |
+----------------------+------------------+----------------------------------------+
| Number Purchase      | $5.00            | Per number                             |
+----------------------+------------------+----------------------------------------+
| Number Renewal       | $5.00            | Per number                             |
+----------------------+------------------+----------------------------------------+
| Extension Calls      | Free             | No charge                              |
+----------------------+------------------+----------------------------------------+

**How Token Consumption Works**

When a token-eligible service is used (VN call or SMS):

1. The system checks the current month's token allowance.
2. If tokens are available, they are consumed first.
3. If tokens are partially available, the available tokens are consumed and the remainder overflows to credits.
4. If no tokens remain, the full cost is charged to credits.

**Rate Calculation Examples**

::

    VN Call (2 minutes 15 seconds) with tokens available:
    +--------------------------------------------+
    | Duration: 2 min 15 sec -> 3 minutes        |
    | (ceiling-rounded to next whole minute)      |
    | Token cost: 3 x 1 = 3 tokens               |
    | Credit cost: $0.00 (covered by tokens)      |
    +--------------------------------------------+

    VN Call (5 minutes) with NO tokens remaining:
    +--------------------------------------------+
    | Duration: 5 minutes                         |
    | Token cost: 0 (exhausted)                   |
    | Credit cost: 5 x $0.0045 = $0.0225         |
    +--------------------------------------------+

    PSTN Outgoing Call (2 minutes 30 seconds):
    +--------------------------------------------+
    | Duration: 2 min 30 sec -> 3 minutes        |
    | (ceiling-rounded to next whole minute)      |
    | Credit cost: 3 x $0.0060 = $0.018          |
    | (always credit-only, no token deduction)    |
    +--------------------------------------------+

    SMS Campaign (100 messages) with 500 tokens remaining:
    +--------------------------------------------+
    | Token cost per message: 10 tokens           |
    | Total tokens needed: 100 x 10 = 1,000      |
    | Tokens available: 500                       |
    | Tokens consumed: 500 (50 messages)          |
    | Overflow to credit: 50 x $0.008 = $0.40    |
    +--------------------------------------------+

    Number Provisioning (1 number):
    +--------------------------------------------+
    | Count: 1 number                            |
    | Credit cost: 1 x $5.00 = $5.00             |
    | (always credit-only, no token deduction)    |
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
        "balance": 150.50,
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z"
    }

**Check Current Token Allowance**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/billing_accounts/<account-id>/allowance?token=<token>'

**Response:**

.. code::

    {
        "id": "allowance-uuid-123",
        "customer_id": "customer-uuid-456",
        "account_id": "billing-uuid-123",
        "cycle_start": "2024-01-01T00:00:00Z",
        "cycle_end": "2024-02-01T00:00:00Z",
        "tokens_total": 1000,
        "tokens_used": 350,
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z"
    }

The ``tokens_total - tokens_used`` gives you the remaining tokens for the current billing cycle.

**List All Allowance Cycles**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/billing_accounts/<account-id>/allowances?token=<token>'

Returns a paginated list of all allowance cycles (current and past) for the account.

**Add Balance (Admin Only)**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/billing_accounts/<account-id>/balance?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "amount": 100.00
        }'

Note: Balance addition is restricted to users with admin permissions for security.


Balance Lifecycle
-----------------
Account balance changes through specific operations. Token allowances reset each billing cycle.

**Balance and Token Flow**

::

    +-------------------+     +-------------------+
    | Add Balance       |     | Monthly Cycle     |
    | (admin only)      |     | (automatic)       |
    +--------+----------+     +--------+----------+
             |                         |
             v                         v
    +-------------------+     +-------------------+
    |  Credit Balance   |     |  Token Allowance  |
    |     $150.50       |     |   650 / 1000      |
    +--------+----------+     +--------+----------+
             |                         |
             | PSTN calls,             | VN calls,
             | numbers,                | SMS
             | overflow                |
             v                         v
    +-------------------+     +-------------------+
    |   Credit Charges  |     |  Token Usage      |
    | - $0.018 PSTN call|     | - 3 tokens call   |
    | - $5.00 number    |     | - 10 tokens SMS   |
    | - $0.40 overflow  |     |                   |
    +--------+----------+     +--------+----------+
             |                         |
             v                         | exhausted
    +-------------------+              |
    |  Updated Balance  |<-------------+
    |     $145.08       |   overflow charges
    +-------------------+


Balance Monitoring
------------------
Monitor balance and token usage to avoid service interruption.

**Balance Check Flow**

::

    Before Service Execution:
    +--------------------------------------------+
    | 1. Identify service type                   |
    | 2. If token-eligible: check token balance  |
    |    - If tokens available -> proceed         |
    |    - If no tokens -> check credit balance   |
    | 3. If credit-only: check credit balance    |
    |    - If balance >= cost -> proceed          |
    |    - If balance < cost -> deny service      |
    +--------------------------------------------+

**Low Balance Handling**

::

    +-------------------+     +-------------------+
    | Balance: $10.00   |     | Tokens: 0 / 1000  |
    +--------+----------+     +--------+----------+
             |                         |
             | Attempt VN call         |
             | (no tokens left)        |
             v                         |
    +-------------------+              |
    | Check credit      |<-------------+
    | for overflow      |  overflow
    +--------+----------+
             |
             | $10.00 >= $0.0045/min
             v
    +-------------------+
    | Call proceeds      |
    | (credit charged)   |
    +-------------------+


Common Scenarios
----------------

**Scenario 1: Token-Based Monthly Usage**

Track token consumption throughout the month.

::

    Monthly Usage (Free Plan, 1000 tokens):
    +--------------------------------------------+
    | Week 1:                                    |
    | - 50 VN calls (avg 3 min) = 150 tokens     |
    | - 20 SMS = 200 tokens                      |
    | Tokens remaining: 650                      |
    |                                            |
    | Week 2:                                    |
    | - 40 VN calls (avg 2 min) = 80 tokens      |
    | - 30 SMS = 300 tokens                      |
    | Tokens remaining: 270                      |
    |                                            |
    | Week 3:                                    |
    | - 30 VN calls (avg 3 min) = 90 tokens      |
    | - 15 SMS = 150 tokens                      |
    | Tokens remaining: 30                       |
    |                                            |
    | Week 4 (tokens nearly exhausted):          |
    | - 10 VN calls (avg 3 min) = 30 tokens      |
    |   (uses last tokens)                       |
    | - 5 SMS = overflow to credit               |
    |   5 x $0.008 = $0.04 credit charge         |
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
    |   - 200 overflow x 3 min x $0.0045 = $2.70|
    |                                            |
    | PSTN Calls: 50 calls (avg 2 min)           |
    | - Credit: 50 x 2 x $0.006 = $0.60         |
    | (always credit, no token deduction)         |
    |                                            |
    | SMS: 100 messages                          |
    | - Tokens needed: 100 x 10 = 1,000          |
    | - If 0 tokens remaining:                   |
    |   - Credit: 100 x $0.008 = $0.80          |
    |                                            |
    | Total credit needed: $2.70 + $0.60 + $0.80 |
    |                    = $4.10                  |
    +--------------------------------------------+

**Scenario 3: Low Balance Alert**

Handle low balance situations.

::

    1. Balance drops below threshold
       +--------------------------------------------+
       | Balance: $15.00                           |
       | Threshold: $50.00                         |
       | Tokens: 0 / 1000 (exhausted)              |
       | Status: LOW BALANCE                       |
       +--------------------------------------------+

    2. Add funds (admin action)
       POST /billing_accounts/{id}/balance
       { "amount": 200.00 }

    3. Balance restored
       +--------------------------------------------+
       | Balance: $215.00                          |
       | Status: OK                                |
       +--------------------------------------------+


Best Practices
--------------

**1. Token Monitoring**

- Check token usage regularly via the allowance endpoint
- Plan upgrades to higher tiers if tokens are consistently exhausted early in the cycle
- Track which services consume the most tokens

**2. Balance Monitoring**

- Maintain credit balance for PSTN calls, number purchases, and token overflow
- Set up low balance alerts
- Plan for buffer above minimum needed

**3. Cost Estimation**

- Separate estimates into token-eligible and credit-only services
- Account for token overflow in budget planning
- Include retry costs in estimates

**4. Security**

- Restrict balance add permissions to admins
- Monitor for unusual usage patterns
- Review billing regularly for anomalies

**5. Plan Selection**

- Choose plan tier based on expected VN call and SMS volume
- Compare token allowance cost vs. credit-only cost at each tier
- Consider upgrading if monthly overflow charges are significant


Troubleshooting
---------------

**Balance Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Services being denied     | Check credit balance and token allowance;      |
|                           | add funds or upgrade plan if needed            |
+---------------------------+------------------------------------------------+
| Cannot add balance        | Verify admin permissions; check API token      |
|                           | validity                                       |
+---------------------------+------------------------------------------------+
| Balance not updating      | Allow time for transaction processing; check   |
|                           | for API errors in response                     |
+---------------------------+------------------------------------------------+

**Token Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Tokens exhausted early    | Review usage patterns via allowance endpoint;   |
|                           | consider upgrading plan tier for more tokens    |
+---------------------------+------------------------------------------------+
| Unexpected overflow       | Check token balance via allowance endpoint;     |
|                           | VN calls and SMS consume tokens first          |
+---------------------------+------------------------------------------------+
| Tokens not resetting      | Verify billing cycle dates; tokens reset       |
|                           | at the start of each monthly cycle             |
+---------------------------+------------------------------------------------+

**Billing Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Unexpected charges        | Review call/message logs; check if tokens      |
|                           | were exhausted causing credit overflow          |
+---------------------------+------------------------------------------------+
| Rate seems wrong          | Verify current rate structure; note calls are   |
|                           | billed per minute (ceiling-rounded)            |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Customer Overview <customer-overview>` - Account management
- :ref:`Call Overview <call-overview>` - Voice call costs
- :ref:`Message Overview <message-overview>` - SMS costs
- :ref:`Number Overview <number-overview>` - Number provisioning costs
