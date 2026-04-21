.. _providercall-main:

************
ProviderCall
************
A ProviderCall is an admin-triggered outbound call placed through a specific SIP provider, bypassing normal dialroute selection. It is an audit record that captures the admin's original request plus the IDs of the Call and Groupcall records produced by the underlying call-creation step.

**API Reference:** `ProviderCall endpoints <https://api.voipbin.net/redoc/#tag/ProviderCall>`_

.. include:: providercall_overview.rst
.. include:: providercall_struct_providercall.rst
.. include:: providercall_tutorial.rst
