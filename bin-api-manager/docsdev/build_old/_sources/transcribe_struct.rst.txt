.. _transcribe-struct-transcribe:

Transcribe
==========

Transcribe
----------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "language": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>",
    }

* id: Transcribe's ID.
* customer_id: Customer's ID.
* reference_type: Reference type. See detail :ref:`here <transcribe-struct-transcribe-reference_type>`.
* reference_id: Reference ID.
* status: Transcribe's status. See detail :ref:`here <transcribe-struct-transcribe-status>`.
* language: Transcribe's language. BCP47 format.

example
+++++++

.. code::

    {
        "id": "bbf08426-3979-41bc-a544-5fc92c237848",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "reference_type": "call",
        "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f",
        "status": "done",
        "language": "en-US",
        "tm_create": "2024-04-01 07:17:04.091019",
        "tm_update": "2024-04-01 13:25:32.428602",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _transcribe-struct-transcribe-reference_type:

reference_type
--------------
Reference's type

=========== ============
Type        Description
=========== ============
call        Reference type is call.
recording   Reference type is recording.
confbridge  Reference type is confbridge.
=========== ============

.. _transcribe-struct-transcribe-status:

status
--------------
Transcribe's status

=========== ============
Type        Description
=========== ============
progressing Transcribe is on progress.
done        Transcribe is done.
=========== ============

.. _transcribe-struct-transcription:

Transcription
=============

Transcription
-------------

.. code::

    {
        "id": "<string>",
        "transcribe_id": "<string>",
        "direction": "<string>",
        "message": "<string>",
        "tm_transcript": "<string>",
        "tm_create": "<string>",
    },

* id: Transcription's id.
* transcribe_id: Transcribe's id.
* direction: Transcription's direction. See detail :ref:`here <transcribe-struct-transcription-direction>`.
* message: Transcription's message.
* tm_transcript: Transcription's timestamp. "0001-01-01 00:00:00" is the beginning of the transcribe.

example
+++++++

.. code::

    {
        "id": "06af78f0-b063-48c0-b22d-d31a5af0aa88",
        "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
        "direction": "in",
        "message": "Hi, good to see you. How are you today.",
        "tm_transcript": "0001-01-01 00:05:04.441160",
        "tm_create": "2024-04-01 07:22:07.229309"
    }

.. _transcribe-struct-transcription-direction:

direction
---------
Transcription's direction

=========== ============
Type        Description
=========== ============
in          Incoming voice toward to the voipbin.
out         Outgoing voice from the voipbin.
=========== ============
