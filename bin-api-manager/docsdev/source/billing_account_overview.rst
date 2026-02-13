.. _billing_account_overview:

Overview
========
VoIPBIN's Billing Account API provides balance management and usage tracking for your account. With prepaid billing, customers maintain an account balance that is debited as services are used. The API enables you to check balances, add funds, and monitor consumption.

With the Billing Account API you can:

- Check current account balance
- Add funds to your account (admin only)
- View rate information
- Monitor usage and charges
- Track billing history


How Billing Works
-----------------
VoIPBIN uses a prepaid billing model where services consume account balance.

**Billing Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Billing System                                |
    +-----------------------------------------------------------------------+

    +-------------------+
    |  Billing Account  |
    |    (balance)      |
    +--------+----------+
             |
             | debited by
             v
    +--------+----------+--------+----------+--------+----------+
    |                   |                   |                   |
    v                   v                   v                   v
    +----------+   +----------+   +----------+   +----------+
    |  Calls   |   | Messages |   | Numbers  |   |  Other   |
    | per sec  |   | per msg  |   | monthly  |   | services |
    +----------+   +----------+   +----------+   +----------+
         |              |              |              |
         v              v              v              v
    +---------+    +---------+    +---------+    +---------+
    | $0.020  |    | $0.008  |    |  $5.00  |    |  varies |
    | /second |    | /message|    | /number |    |         |
    +---------+    +---------+    +---------+    +---------+

**Key Components**

- **Billing Account**: The prepaid balance for your customer account
- **Balance**: Current available funds
- **Charges**: Debits for services used
- **Rate**: Cost per unit for each service type


Plan Tiers
----------
Each billing account is assigned a plan tier that determines resource creation limits. New accounts default to the **free** tier.

**Available Tiers**

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
VoIPBIN operates on a transparent fixed-rate system.

**Current Rates**

+----------------------+------------------+----------------------------------------+
| Service              | Rate (USD)       | Unit                                   |
+======================+==================+========================================+
| Number Purchase      | $5.00            | Per number                             |
+----------------------+------------------+----------------------------------------+
| Voice Calls          | $0.020           | Per second                             |
+----------------------+------------------+----------------------------------------+
| SMS Messages         | $0.008           | Per message                            |
+----------------------+------------------+----------------------------------------+

**Rate Calculation Examples**

::

    Voice Call (2 minutes):
    +--------------------------------------------+
    | Duration: 120 seconds                      |
    | Rate: $0.020/second                        |
    | Total: 120 x $0.020 = $2.40               |
    +--------------------------------------------+

    SMS Campaign (100 messages):
    +--------------------------------------------+
    | Count: 100 messages                        |
    | Rate: $0.008/message                       |
    | Total: 100 x $0.008 = $0.80               |
    +--------------------------------------------+

    Number Provisioning (1 number):
    +--------------------------------------------+
    | Count: 1 number                            |
    | Rate: $5.00/number                         |
    | Total: 1 x $5.00 = $5.00                  |
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
        "currency": "USD",
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z"
    }

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
Account balance changes through specific operations.

**Balance Flow**

::

    +-------------------+
    | Add Balance       |
    | (admin only)      |
    +--------+----------+
             |
             v
    +-------------------+
    |  Current Balance  |<-----------------+
    |     $150.50       |                  |
    +--------+----------+                  |
             |                             |
             | Use services                | Add more funds
             v                             |
    +-------------------+                  |
    |   Charges         |                  |
    | - $2.40 call      |                  |
    | - $0.80 SMS       |                  |
    | - $5.00 number    |                  |
    +--------+----------+                  |
             |                             |
             v                             |
    +-------------------+     Low          |
    |  Updated Balance  |----------------->+
    |     $142.30       |   balance
    +-------------------+   alert


Balance Monitoring
------------------
Monitor balance to avoid service interruption.

**Balance Check Flow**

::

    Before Service Execution:
    +--------------------------------------------+
    | 1. Check current balance                   |
    | 2. Estimate service cost                   |
    | 3. If balance >= cost -> proceed           |
    | 4. If balance < cost -> deny service       |
    +--------------------------------------------+

**Low Balance Handling**

::

    +-------------------+
    | Balance: $10.00   |
    +--------+----------+
             |
             | Attempt 5-min call
             | (estimated: $6.00)
             v
    +-------------------+     Sufficient
    | Balance >= Cost?  |----------------------> Call proceeds
    +--------+----------+
             |
             | Insufficient
             v
    +-------------------+
    | Service denied    |
    | or limited        |
    +-------------------+


Common Scenarios
----------------

**Scenario 1: Prepaid Balance Management**

Maintain adequate balance for operations.

::

    Daily Operations:
    +--------------------------------------------+
    | Morning: Check balance                     |
    | GET /billing_accounts                      |
    | Balance: $500.00                           |
    |                                            |
    | Throughout day:                            |
    | - 50 calls (avg 3 min) = $180.00          |
    | - 200 SMS = $1.60                          |
    | - 2 numbers = $10.00                       |
    |                                            |
    | Evening: Check balance                     |
    | Balance: $308.40                           |
    +--------------------------------------------+

**Scenario 2: Campaign Budget Planning**

Estimate costs before running a campaign.

::

    Campaign: Customer Survey
    +--------------------------------------------+
    | Targets: 1,000 customers                   |
    | Method: Voice call (avg 45 seconds)        |
    |                                            |
    | Cost estimate:                             |
    | - Assumed answer rate: 50%                 |
    | - Calls to complete: 500                   |
    | - Cost per call: 45 x $0.020 = $0.90      |
    | - Total: 500 x $0.90 = $450.00            |
    |                                            |
    | Required balance: >= $450.00               |
    +--------------------------------------------+

**Scenario 3: Low Balance Alert**

Handle low balance situations.

::

    1. Balance drops below threshold
       +--------------------------------------------+
       | Balance: $15.00                           |
       | Threshold: $50.00                         |
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

**1. Balance Monitoring**

- Check balance regularly before operations
- Set up low balance alerts
- Plan for buffer above minimum needed

**2. Cost Estimation**

- Calculate expected costs before campaigns
- Include retry costs in estimates
- Account for peak usage periods

**3. Security**

- Restrict balance add permissions to admins
- Monitor for unusual usage patterns
- Review billing regularly for anomalies

**4. Budget Planning**

- Track monthly spending trends
- Set usage budgets per department
- Review rate changes periodically


Troubleshooting
---------------

**Balance Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Services being denied     | Check balance is sufficient; add funds if      |
|                           | needed; verify rate calculations               |
+---------------------------+------------------------------------------------+
| Cannot add balance        | Verify admin permissions; check API token      |
|                           | validity                                       |
+---------------------------+------------------------------------------------+
| Balance not updating      | Allow time for transaction processing; check   |
|                           | for API errors in response                     |
+---------------------------+------------------------------------------------+

**Billing Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Unexpected charges        | Review call/message logs; check for failed     |
|                           | attempts that still incur charges              |
+---------------------------+------------------------------------------------+
| Rate seems wrong          | Verify current rate structure; check if        |
|                           | special rates apply                            |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Customer Overview <customer-overview>` - Account management
- :ref:`Call Overview <call-overview>` - Voice call costs
- :ref:`Message Overview <message-overview>` - SMS costs
- :ref:`Number Overview <number-overview>` - Number provisioning costs

