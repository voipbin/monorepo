.. _email-struct-attachment:

Attachment
==========

.. _email-struct-attachment-attachment:

Attachment
----------

.. code::

    {
        "reference_type": "<string>",
        "reference_id": "<uuid>"
    }

* ``reference_type`` (enum string): The type of resource this attachment references. See :ref:`Type <email-struct-attachment-type>`.
* ``reference_id`` (UUID): The unique identifier of the referenced resource. For example, if ``reference_type`` is ``recording``, this is the recording's ID obtained from ``GET /recordings``.

.. _email-struct-attachment-type:

Type
------
Attachment's type.

+------------+------------------------------------------------------------------+
| Type       | Description                                                      |
+============+==================================================================+
| ``""``     | No type set. Default when no reference is specified.             |
+------------+------------------------------------------------------------------+
| recording  | A call recording file. The ``reference_id`` corresponds to a    |
|            | recording ID from ``GET /recordings``.                           |
+------------+------------------------------------------------------------------+

