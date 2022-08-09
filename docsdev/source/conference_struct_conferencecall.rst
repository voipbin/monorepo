.. _conference-struct-conferencecall:

conferencecall
==============

Conferencecall
--------------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "conference_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Conferencecall's ID.
* customer_id: Customer's ID.
* conference_id: Conference's ID.
* *reference_type*: Reference type. See detail :ref:`here <conference-struct-conferencecall-reference_type>`.
* reference_id: Reference ID.
* *status*: Conferencecall's status. See detail :ref:`here <conference-struct-conferencecall-status>`.


Example
+++++++

.. code::

    {
        "id": "b8aa51f6-5cc1-40ba-9737-45ca24dab153",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "conference_id": "99accfb7-c0dd-4a54-997d-dd18af7bc280",
        "reference_type": "call",
        "reference_id": "7cb70145-a20a-4070-8b23-9131410d301d",
        "status": "leaved",
        "tm_create": "2022-08-06 16:57:12.247946",
        "tm_update": "2022-08-06 19:09:47.349667",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _conference-struct-conferencecall-reference_type:

Reference type
--------------
Reference's type

========== ==============
Type       Description
========== ==============
call       call
========== ==============

.. _conference-struct-conferencecall-status:

Status
--------------
Status

========== ==============
Type       Description
========== ==============
joining    The call is joining to the conference.
joined     The call is joined to the conference.
leaving    The call is leaving from the conference.
leaved     The call is leaved from the conference.
========== ==============
