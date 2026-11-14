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
* ``reference_id`` (UUID): The unique identifier of the referenced resource. If ``reference_type`` is ``recording``, this is the recording's ID obtained from ``GET /recordings``. If it is ``file``, this is the storage file's ID obtained from ``POST /storage_files`` or ``GET /storage_files``. The resource must belong to the customer that sends the email.

.. _email-struct-attachment-type:

Type
------
Attachment's type.

.. list-table::
   :header-rows: 1

   * - Type
     - Description
   * - ``""``
     - No type set. Default when no reference is specified.
   * - recording
     - A call recording file. The ``reference_id`` corresponds to a recording ID from ``GET /recordings``. The files of the recording are attached as one zip file.
   * - file
     - A storage file of type ``talk``, ``media`` or ``recording``. The ``reference_id`` corresponds to a storage file ID from ``GET /storage_files``. Files of type ``rag`` are refused.

An email can have at most 5 attachments, and the total size of the original files can be at most 18 MiB.


