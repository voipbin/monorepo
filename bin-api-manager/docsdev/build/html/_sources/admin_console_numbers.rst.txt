Managing Numbers
=================

.. index:: single: Admin console; Numbers
.. index:: single: Admin console; Buy a number

Phone numbers are managed under **Platform -> Numbers**.

.. image:: _static/images/admin_console_number_list.png
   :alt: Numbers list page
   :width: 700px
   :align: center

The list shows every number provisioned to your account: the E.164
number, its assigned Flow, provider, and status.

.. note:: **AI Implementation Hint**

   This maps to the ``numbers`` resource in the REST API (see
   :ref:`Number <number-main>`). Numbers you buy here are immediately
   usable in a Flow's Call or Connect nodes, or as the ``target`` of an
   API-initiated call.

Buying a number
----------------

Click **Buy Number** to open the number search. Filter by country and
area code/prefix, review the available numbers and their monthly cost,
and confirm the purchase. Once purchased, a number appears in the Active
list immediately and can be assigned to a Flow from its detail page.

.. warning::

   Buying a number is a billed action against your account balance. There
   is no cost-free "test buy" -- only purchase numbers you intend to use.

Assigning a number to a Flow
------------------------------

Open a number's detail page and set its **Flow** field to route all
inbound calls to that number through the selected Flow. Until a Flow is
assigned, inbound calls to a newly purchased number have nowhere to go.
