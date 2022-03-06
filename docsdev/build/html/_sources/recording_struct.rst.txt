.. _recording-struct:

Struct
======

.. _recording-struct-recording:

Recording
---------

.. code::

    {
        "id": "<string>",
        "type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "format": "<string>",
        "tm_start": "<string>",
        "tm_end": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Recording's ID.
* type: Recording's type. See detail :ref:`here <reocording-struct-type>`.
* reference_id: Reference's ID. It shows call/conference's ID.
* status: Recording status. See detail :ref:`here <recording-struct-status>`.
* format: Recording file format. See detail :ref:`here <recording-struct-format>`.

.. _reocording-struct-type:

Type
----

========== ===========
Type       Description
========== ===========
call       Call recording.
conference Conference recording.
========== ===========

.. _recording-struct-status:

Status
------

========== ===========
Type       Description
========== ===========
initiating Preparing the recording.
recording  Recording now.
ended      Recording ended.
========== ===========

.. _recording-struct-format:

Format
------

========== ===========
Type       Description
========== ===========
wav        Wav format.
========== ===========
