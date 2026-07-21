Campaigns & Outbound Dialing
==============================

.. index:: single: Admin console; Campaign
.. index:: single: Admin console; Outdial
.. index:: single: Admin console; Outplan

**Platform -> Campaigns** manages outbound dialing campaigns: bulk
calling or messaging runs against a list of targets, driven by a Flow.

.. image:: _static/images/admin_console_campaign_list.png
   :alt: Campaigns list page
   :width: 700px
   :align: center

The page has four tabs:

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Tab
     - Purpose
   * - **Campaigns**
     - The campaign definitions themselves: type (``call`` or ``flow``),
       status, service level, and which Outplan/Outdial/Queue it uses.
       Click **Create Campaign** to configure a new one.
   * - **Campaign Calls**
     - Individual call attempts a running campaign has made, with their
       outcome.
   * - **Outdials**
     - Named target lists (the phone numbers or destinations a campaign
       dials). A campaign references one Outdial.
   * - **Outplans**
     - Retry/dialing strategy definitions (max attempts, retry interval)
       that a campaign references to decide how hard to retry an
       unanswered target.

.. note:: **AI Implementation Hint**

   These map to four separate REST API resources: campaigns (see
   :ref:`Campaign <campaign-main>`), outdials (see
   :ref:`Outdial <outdial-main>`), and outplans (see
   :ref:`Outplan <outplan-main>`). A Campaign always references exactly
   one Outdial and one Outplan by ID.

.. warning::

   Starting a Campaign against real phone numbers places real outbound
   calls and is a billed action. Review the target Outdial list and the
   Outplan retry settings before starting a campaign at scale.
