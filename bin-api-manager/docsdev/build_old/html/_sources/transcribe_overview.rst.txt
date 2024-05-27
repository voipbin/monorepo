.. _transcribe-overview:

Overview
========
Voipbin's transcription functionality is designed to cater to a range of communication needs, covering calls, conferences, and recordings. This comprehensive support ensures that users can transcribe various types of interactions accurately and efficiently.

Whether it's a one-on-one conversation, a large conference call, or a recorded discussion, Voipbin's transcription service handles it with ease. By distinguishing between audio input and output, it provides nuanced transcriptions that accurately reflect the dialogue exchanged during communication sessions. This differentiation ensures that users can clearly identify who said what, enhancing the clarity and usefulness of the transcribed content.

Real-Time capability
--------------------
One notable aspect of Voipbin's transcription service is its real-time capability. This feature enables users to transcribe conversations as they occur, providing instant access to written records of ongoing discussions. Real-time transcription not only facilitates live communication but also streamlines documentation processes by eliminating the need for manual transcription after the fact. This functionality is particularly valuable in fast-paced environments where quick access to accurate information is essential.

Additionally, Voipbin offers enhanced flexibility through websocket event subscription. Users can subscribe or unsubscribe to the transcript event using websocket event subscribe, ensuring seamless integration with their applications or systems. This allows for dynamic control over real-time transcription notifications, tailored to specific needs and workflows.

Moreover, Voipbin offers an added feature for enhanced integration and convenience. By including webhook information in your customer settings, you can receive real-time updates through the `transcript_created` event of your transcription process. This enables seamless integration with your existing systems or applications, ensuring that you stay informed of transcription progress without manual intervention.

Overall, Voipbin's transcription service offers a comprehensive solution for capturing and documenting verbal communication across various platforms. Whether users need transcriptions for analysis, reference, or archival purposes, Voipbin's transcription feature delivers accurate and timely results, enhancing communication workflows and productivity.

.. code::

    {
        "type": "transcript_created",
        "data": {
            "id": "9d59e7f0-7bdc-4c52-bb8c-bab718952050",
            "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
            "direction": "out",
            "message": "Hello, this is transcribe test call.",
            "tm_transcript": "0001-01-01 00:00:08.991840",
            "tm_create": "2024-04-04 07:15:59.233415"
        }
    }

.. _transcribe-overview-transcription:

Transcription
-------------
Voipbin's transcription service not only captures the spoken word but also provides additional context by distinguishing between audio input and output. This unique feature enables users to discern the direction of the voice within each transcription, offering valuable insight into the flow of communication.

By indicating whether the audio is incoming or outgoing, Voipbin's transcription service adds an extra layer of clarity to the transcribed content. Users can easily identify who initiated a statement or response, enhancing their understanding of the conversation dynamics.

.. code::

    +--------+                               +-------+
    |Customer|------ Direction: in --------->|Voipbin|
    |        |                               |       |
    |        |<----- Direction: out ---------|       |
    +--------+                               +-------+

For example, in a call or conference scenario, users can quickly determine whether a particular remark was made by the caller or the recipient. Similarly, in recorded discussions, the audio in/out indication helps differentiate between speakers, facilitating more accurate transcription and analysis.

This audio in/out distinguish feature empowers users to gain a deeper understanding of the context and dynamics of communication, leading to more effective collaboration, documentation, and analysis. Whether it's monitoring customer interactions, conducting research, or reviewing meeting minutes, Voipbin's transcription service offers enhanced clarity and insight into verbal communication.

.. code::

    [
        {
            "id": "06af78f0-b063-48c0-b22d-d31a5af0aa88",
            "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
            "direction": "in",
            "message": "Hi, good to see you. How are you today.",
            "tm_transcript": "0001-01-01 00:01:04.441160",
            "tm_create": "2024-04-01 07:22:07.229309"
        },
        {
            "id": "3c95ea10-a5b7-4a68-aebf-ed1903baf110",
            "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
            "direction": "out",
            "message": "Welcome to the transcribe test scenario in this scenario. All your voice will be transcribed and delivered it to the web hook.",
            "tm_transcript": "0001-01-01 00:00:43.116830",
            "tm_create": "2024-04-01 07:17:27.208337"
        }
    ]

Enable transcribe
-----------------
Voipbin provides two different methods to start the transcribe.

Automatic Trigger in the Flow
+++++++++++++++++++++++++++++++
Add the `transcribe_start` action in the action flow. This action automatically triggers transcribe when the flow reaches it. See detail :ref:`here <flow-struct-action-transribe_start>`.

.. code::

    {
        "id": "95c7a67f-9643-4237-8b69-7320a70b382b",
        "next_id": "44e1dabc-a8c1-4647-90ba-16d414231058",
        "type": "transcribe_start",
        "option": {
            "language": "en-US"
        }
    }


Interrupt Trigger(Manual API Request)
+++++++++++++++++++++++++++++++++++++
The client can start the transcribe by API request sending. This allows you to start transcription manually in the middle of a call or conference. However, note that this method requires someone to initiate the API request.

* POST /v1.0/transcribes: See detail `here <https://api.voipbin.net/swagger/index.html#/default/post_v1_0_transcribes>`_.

.. code::

    $ curl -X POST --location 'https://api.voipbin.net/v1.0/transcribes?token=token' \
        --header 'Content-Type: application/json' \
        --data '{
            "reference_type": "call",
            "reference_id": "8c71bcb6-e7e7-4ed2-8aba-44bc2deda9a5",
            "language": "en-US",
            "direction": "both"
        }'