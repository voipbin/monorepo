Getting Started
================

.. index:: single: Admin console; Login
.. index:: single: Admin console; Permission levels
.. index:: single: Admin console; Try Demo Account

Signing in
----------

Go to https://admin.voipbin.net and sign in with your VoIPBin account
email and password. If you just want to explore the console without
creating an account, click **Try Demo Account** on the login page; this
signs you into a shared read/write demo account so you can click around
before committing to a real signup.

.. image:: _static/images/admin_console_getting_started_home.png
   :alt: Admin console home page after login
   :width: 700px
   :align: center

Permission levels
-----------------

Every agent (user) in a VoIPBin customer account holds exactly one
permission level, and the console hides menu items and buttons the current
level cannot use.

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Level
     - What it can do
   * - **Customer Agent**
     - Base level. Handle calls, conversations, and messages assigned to
       the agent through the agent-facing tools (for example
       talk.voipbin.net). A Customer Agent account cannot sign in to
       admin.voipbin.net -- the console blocks it with an
       "Access Denied" message and signs the account back out, since this
       guide covers manager/admin-level tooling only.
   * - **Customer Manager**
     - Everything an Agent can do, plus manage Flows, Numbers, Agents,
       Contacts, Queues, AI Assistants, and other resource configuration
       for the account.
   * - **Customer Admin**
     - Everything a Manager can do, plus billing, access keys, and
       customer-level settings.
   * - **Project Admin**
     - Platform operator level. Not used by customer accounts.

.. note:: **AI Implementation Hint**

   If a menu item or a "Create" button described in this guide does not
   appear for you, the most common cause is a permission level below what
   that page requires, not a bug. Ask your account's Customer Admin to
   raise your permission level.

Console layout
--------------

.. image:: _static/images/admin_console_getting_started_sidebar.png
   :alt: Admin console sidebar with resource groups expanded
   :width: 700px
   :align: center

The left sidebar groups resources by function:

- **Voice** -- Calls, Conferences, Queues, Recordings, Transcribes.
- **Messaging** -- SMS/MMS Messages, Email, Conversations, Webchat.
- **AI Services** -- AI Assistants, Teams, RAGs, AI Calls, Summaries.
- **Timeline** -- Execution history for flows and calls.
- **Platform** -- Flows, Numbers, Campaigns, Agents, Contacts, Unresolved
  Cases, Extensions, Trunks, Tags, Storage Files.
- **Operations** -- Billing, access keys, and account-level configuration.

Every list page (for example Flows or Numbers) has the same three parts:
a **Create** button in the top right, a search box to filter rows, and a
table with sortable columns. Clicking a row opens that resource's detail
page.
