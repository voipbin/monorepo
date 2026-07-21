Webchat Widgets
================

.. index:: single: Admin console; Webchat

**Messaging -> Webchat** manages embeddable webchat widgets, small chat
bubbles you place on your own website to let visitors talk to your team
or an AI Assistant without leaving the page.

.. image:: _static/images/admin_console_webchat_list.png
   :alt: Webchat widgets list page
   :width: 700px
   :align: center

The page has two tabs:

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Tab
     - Purpose
   * - **Widgets**
     - Every widget configured for your account: name, status (Active or
       Inactive), and idle timeout in seconds. Click **Create Widget** to
       configure a new one and get its embed snippet.
   * - **Sessions**
     - Live and past visitor chat sessions across all your widgets, for
       reviewing conversations or reopening one.

.. note:: **AI Implementation Hint**

   This maps to the ``webchat`` resource family in the REST API (see
   :ref:`Webchat <webchat-main>`). A widget can be wired to a Flow (for
   an AI Assistant or a human agent handoff) the same way a phone number
   is.
