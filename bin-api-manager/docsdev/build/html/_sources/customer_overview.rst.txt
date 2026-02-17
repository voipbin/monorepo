.. _customer-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free (customer management operations incur no charges)
   * **Async:** No. All customer operations are synchronous and return immediately.

VoIPBIN's Customer API provides account-level management for your organization within the platform. A customer represents a tenant account that owns resources like agents, numbers, flows, and billing. The Customer API enables you to manage account settings, view usage, and configure organization-wide preferences.

With the Customer API you can:

- View and update customer account information
- Manage account-level settings and preferences
- Monitor resource usage and limits
- Configure default behaviors for the account
- Access account metadata and timestamps


How Customers Work
------------------
A customer is the top-level organizational unit in VoIPBIN that owns all other resources.

**Customer Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Customer Account                              |
    +-----------------------------------------------------------------------+

    +-------------------+
    |     Customer      |
    |  (organization)   |
    +--------+----------+
             |
             | owns
             v
    +--------+----------+--------+----------+--------+----------+
    |                   |                   |                   |
    v                   v                   v                   v
    +----------+   +----------+   +----------+   +----------+
    |  Agents  |   | Numbers  |   |  Flows   |   | Billing  |
    +----------+   +----------+   +----------+   +----------+
         |              |              |              |
         v              v              v              v
    +---------+    +---------+    +---------+    +---------+
    | Users   |    | Phone   |    | Call    |    | Account |
    | Skills  |    | Lines   |    | Logic   |    | Balance |
    +---------+    +---------+    +---------+    +---------+

**Key Components**

- **Customer**: The tenant account that owns all resources
- **Agents**: Users who handle calls and messages
- **Numbers**: Phone numbers provisioned for the account
- **Flows**: Call and message handling logic
- **Billing**: Account balance and payment information


Customer Properties
-------------------
Key properties of a customer account.

**Core Properties**

+------------------+----------------------------------------------------------------+
| Property         | Description                                                    |
+==================+================================================================+
| id               | Unique identifier for the customer account                     |
+------------------+----------------------------------------------------------------+
| name             | Display name of the organization                               |
+------------------+----------------------------------------------------------------+
| detail           | Additional description or notes                                |
+------------------+----------------------------------------------------------------+
| status           | Account status (active, suspended, etc.)                       |
+------------------+----------------------------------------------------------------+
| permission_ids   | Default permissions for new agents                             |
+------------------+----------------------------------------------------------------+

**Timestamps**

+------------------+----------------------------------------------------------------+
| Property         | Description                                                    |
+==================+================================================================+
| tm_create        | When the account was created                                   |
+------------------+----------------------------------------------------------------+
| tm_update        | When the account was last modified                             |
+------------------+----------------------------------------------------------------+
| tm_delete        | When the account was deleted (if applicable)                   |
+------------------+----------------------------------------------------------------+


Managing Customers
------------------
Access and update customer account information.

**Get Customer Information**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/customers/<customer-id>?token=<token>'

**Response:**

.. code::

    {
        "id": "customer-uuid-123",
        "name": "Acme Corporation",
        "detail": "Enterprise customer account",
        "status": "active",
        "permission_ids": [...],
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z"
    }

**Update Customer Information**

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/customers/<customer-id>?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Acme Corporation Inc.",
            "detail": "Updated enterprise account"
        }'


Resource Ownership
------------------
All resources in VoIPBIN are scoped to a customer.

**Resource Hierarchy**

::

    Customer: "Acme Corp"
    +-----------------------------------------------------------------------+
    |                                                                       |
    |  Agents (10)                                                          |
    |  +-- John Smith (agent-001)                                          |
    |  +-- Jane Doe (agent-002)                                            |
    |  +-- ... (8 more)                                                    |
    |                                                                       |
    |  Numbers (5)                                                          |
    |  +-- +15551234567 (main line)                                        |
    |  +-- +15559876543 (support)                                          |
    |  +-- ... (3 more)                                                    |
    |                                                                       |
    |  Flows (8)                                                            |
    |  +-- IVR Main Menu                                                   |
    |  +-- Support Queue Router                                            |
    |  +-- ... (6 more)                                                    |
    |                                                                       |
    |  Queues (3)                                                           |
    |  +-- Sales Queue                                                     |
    |  +-- Support Queue                                                   |
    |  +-- Billing Queue                                                   |
    |                                                                       |
    +-----------------------------------------------------------------------+

**Resource Isolation**

::

    +-------------------+        +-------------------+
    |   Customer A      |        |   Customer B      |
    +-------------------+        +-------------------+
    | Agents: A1, A2    |        | Agents: B1, B2    |
    | Numbers: +1555... |        | Numbers: +1666... |
    | Flows: Flow-A     |        | Flows: Flow-B     |
    +-------------------+        +-------------------+
           |                            |
           |  ISOLATED                  |
           |  (cannot access            |
           |   each other's resources)  |
           +----------------------------+


Guest Agent
-----------
Every customer account automatically has a guest agent created for administrative access.

::

    Customer Created
           |
           v
    +-------------------+
    | Auto-create       |
    | Guest Agent       |
    +--------+----------+
             |
             v
    +-------------------+
    | Properties:       |
    | o Admin permission|
    | o Cannot delete   |
    | o Cannot change   |
    |   password        |
    +-------------------+

The guest agent ensures every account has at least one administrator for recovery purposes.

.. note:: **AI Implementation Hint**

   The customer ``id`` is the top-level scoping identifier for all resources in VoIPBIN. When creating agents, numbers, flows, or any other resource, they are automatically associated with the customer of the authenticated user. You do not need to pass ``customer_id`` explicitly in most creation requests -- it is derived from the authentication token.


Common Scenarios
----------------

**Scenario 1: Account Setup**

Initial configuration of a new customer account.

::

    1. Customer account created
       +--------------------------------------------+
       | Customer: "New Company LLC"               |
       | Status: active                            |
       | Guest agent: auto-created                 |
       +--------------------------------------------+

    2. Create additional agents
       POST /agents { "username": "admin@..." }

    3. Provision numbers
       POST /numbers { "number": "+1555..." }

    4. Create flows
       POST /flows { "name": "Main IVR" }

    5. Configure resources
       Link numbers to flows, agents to queues

**Scenario 2: Multi-Department Setup**

Organize resources by department.

::

    Customer: "Enterprise Corp"
    +--------------------------------------------+
    |                                            |
    | Sales Department:                          |
    | - Agents: sales-1, sales-2, sales-3       |
    | - Queue: Sales Queue (tags: sales)        |
    | - Number: +18005551111                    |
    |                                            |
    | Support Department:                        |
    | - Agents: support-1, support-2            |
    | - Queue: Support Queue (tags: support)    |
    | - Number: +18005552222                    |
    |                                            |
    | Billing Department:                        |
    | - Agents: billing-1                       |
    | - Queue: Billing Queue (tags: billing)    |
    | - Number: +18005553333                    |
    |                                            |
    +--------------------------------------------+

**Scenario 3: Account Review**

Periodic review of account resources.

::

    Review Checklist:
    +--------------------------------------------+
    | 1. Check active agents                     |
    |    GET /agents -> count, statuses          |
    |                                            |
    | 2. Review provisioned numbers              |
    |    GET /numbers -> count, usage            |
    |                                            |
    | 3. Audit flows                             |
    |    GET /flows -> active, last updated      |
    |                                            |
    | 4. Check billing status                    |
    |    GET /billing_accounts -> balance        |
    +--------------------------------------------+


Best Practices
--------------

**1. Account Organization**

- Use descriptive customer names
- Document account purpose in detail field
- Keep resource naming consistent

**2. Security**

- Limit admin permissions to necessary agents
- Regularly audit agent access
- Use strong passwords for all agents

**3. Resource Management**

- Clean up unused numbers and flows
- Monitor agent activity
- Archive inactive resources

**4. Monitoring**

- Track account usage metrics
- Set up alerts for unusual activity
- Review billing regularly


Troubleshooting
---------------

**Access Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Cannot access customer    | Verify API token is valid; check agent has     |
| data                      | admin permission                               |
+---------------------------+------------------------------------------------+
| Resources not visible     | Ensure accessing correct customer ID; verify   |
|                           | resource exists                                |
+---------------------------+------------------------------------------------+

**Account Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Account suspended         | Check billing status; contact support for      |
|                           | reactivation                                   |
+---------------------------+------------------------------------------------+
| Cannot update account     | Verify admin permissions; check for locked     |
|                           | fields                                         |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Agent Overview <agent_overview>` - Managing agents
- :ref:`Billing Account Overview <billing_account_overview>` - Account billing
- :ref:`Number Overview <number-overview>` - Phone number management
- :ref:`Flow Overview <flow-overview>` - Call flow configuration

