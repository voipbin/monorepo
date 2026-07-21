Trunks & Extensions
=====================

.. index:: single: Admin console; Trunk
.. index:: single: Admin console; Extension

**Platform -> Trunks** and **Platform -> Extensions** connect VoIPBin to
your own SIP infrastructure, separate from the numbers VoIPBin provisions
for you.

Trunks
------

.. image:: _static/images/admin_console_trunk_list.png
   :alt: Trunks list page
   :width: 700px
   :align: center

A Trunk registers a SIP carrier or PBX domain so VoIPBin can send and
receive calls through it instead of (or alongside) VoIPBin-purchased
numbers. The list shows each trunk's domain name, name, and
authentication type. Click **Create Trunk** to register a new carrier
domain and its credentials.

.. note:: **AI Implementation Hint**

   This maps to the ``trunks`` resource in the REST API (see
   :ref:`Trunk <trunk-main>`).

Extensions
----------

**Platform -> Extensions** manages internal SIP extensions, endpoints
that register directly to VoIPBin (for example a desk phone or a
softphone) rather than routing through the PSTN. Use an Extension when
you want an internal line reachable from a Flow's Connect or Call nodes
without going through a public phone number.

.. note:: **AI Implementation Hint**

   This maps to the ``extensions`` resource in the REST API (see
   :ref:`Extension <extension-main>`).
