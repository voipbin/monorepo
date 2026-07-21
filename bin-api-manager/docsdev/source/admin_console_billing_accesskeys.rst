Billing & Access Keys
========================

.. index:: single: Admin console; Billing
.. index:: single: Admin console; Access Key

**Operations** in the sidebar (visible to Customer Admin and above) is
where you manage your account's plan, payment method, and API access
keys.

Billing Account
-----------------

.. image:: _static/images/admin_console_billing_account.png
   :alt: Billing account plan and payment configuration
   :width: 700px
   :align: center

**Operations -> Billing Account** shows your current plan, token
balance, and estimated cost, plus two configuration cards:

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Card
     - What it configures
   * - **Plan**
     - Compare the **Free** plan (fixed monthly token allowance, limited
       extensions/agents/queues/trunks/numbers) against **Unlimited**
       (custom, contact support). **Manage Subscription** opens your
       subscription management page; **Compare plans** shows the full
       feature matrix.
   * - **Payment Configuration**
     - Payment Type (for example Prepaid) and Payment Method (for
       example Credit Card).

**Operations -> Billing History** lists past billing entries (usage
charges, top-ups) for the account.

.. note:: **AI Implementation Hint**

   This maps to the ``billing_accounts`` resource in the REST API (see
   :ref:`Billing Account <billing_account-main>`). VoIPBin bills usage in
   tokens; the **Top Up** button in the top bar adds tokens to your
   balance.

Access Keys
------------

**Operations -> Access Keys** manages long-lived API credentials
(accesskeys) you can use instead of a JWT to authenticate REST API
calls, useful for server-to-server integrations. Create one, copy the
secret shown once at creation time, and use it as described in the
:ref:`Quickstart <quickstart-main>` guide's authentication section.

.. note:: **AI Implementation Hint**

   This maps to the ``accesskeys`` resource in the REST API (see
   :ref:`Access Key <accesskey-main>`).
