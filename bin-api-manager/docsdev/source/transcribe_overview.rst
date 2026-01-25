.. _transcribe-overview:

Overview
========
VoIPBIN's transcription functionality is designed to cater to a range of communication needs, covering calls, conferences, and recordings. This comprehensive support ensures that users can transcribe various types of interactions accurately and efficiently.

Whether it's a one-on-one conversation, a large conference call, or a recorded discussion, VoIPBIN's transcription service handles it with ease. By distinguishing between audio input and output, it provides nuanced transcriptions that accurately reflect the dialogue exchanged during communication sessions. This differentiation ensures that users can clearly identify who said what, enhancing the clarity and usefulness of the transcribed content.

Real-Time capability
--------------------
One notable aspect of VoIPBIN's transcription service is its real-time capability. This feature enables users to transcribe conversations as they occur, providing instant access to written records of ongoing discussions. Real-time transcription not only facilitates live communication but also streamlines documentation processes by eliminating the need for manual transcription after the fact. This functionality is particularly valuable in fast-paced environments where quick access to accurate information is essential.

Additionally, VoIPBIN offers enhanced flexibility through websocket event subscription. Users can subscribe or unsubscribe to the transcript event using websocket event subscribe, ensuring seamless integration with their applications or systems. This allows for dynamic control over real-time transcription notifications, tailored to specific needs and workflows.

Moreover, VoIPBIN offers an added feature for enhanced integration and convenience. By including webhook information in your customer settings, you can receive real-time updates through the `transcript_created` event of your transcription process. This enables seamless integration with your existing systems or applications, ensuring that you stay informed of transcription progress without manual intervention.

Overall, VoIPBIN's transcription service offers a comprehensive solution for capturing and documenting verbal communication across various platforms. Whether users need transcriptions for analysis, reference, or archival purposes, VoIPBIN's transcription feature delivers accurate and timely results, enhancing communication workflows and productivity.

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
VoIPBIN's transcription service not only captures the spoken word but also provides additional context by distinguishing between audio input and output. This unique feature enables users to discern the direction of the voice within each transcription, offering valuable insight into the flow of communication.

By indicating whether the audio is incoming or outgoing, VoIPBIN's transcription service adds an extra layer of clarity to the transcribed content. Users can easily identify who initiated a statement or response, enhancing their understanding of the conversation dynamics.

.. code::

    +--------+                               +-------+
    |Customer|------ Direction: in --------->|VoIPBIN|
    |        |                               |       |
    |        |<----- Direction: out ---------|       |
    +--------+                               +-------+

For example, in a call or conference scenario, users can quickly determine whether a particular remark was made by the caller or the recipient. Similarly, in recorded discussions, the audio in/out indication helps differentiate between speakers, facilitating more accurate transcription and analysis.

This audio in/out distinguish feature empowers users to gain a deeper understanding of the context and dynamics of communication, leading to more effective collaboration, documentation, and analysis. Whether it's monitoring customer interactions, conducting research, or reviewing meeting minutes, VoIPBIN's transcription service offers enhanced clarity and insight into verbal communication.

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
VoIPBIN provides two different methods to start the transcribe.

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

.. _transcribe-overview-supported_languages:

Supported Languages
-------------------
VoIPBIN supports transcription in over 70 languages and regional variants, enabling global communication scenarios. You can specify the desired language using the language option (e.g., "en-US", "ko-KR"). Below is a non-exhaustive list of available language codes:

============== =============================
Language Code  Language
============== =============================
af-ZA          Afrikaans (South Africa)
-------------- -----------------------------
am-ET          Amharic (Ethiopia)
-------------- -----------------------------
ar-AE          Arabic (U.A.E.)
-------------- -----------------------------
ar-BH          Arabic (Bahrain)
-------------- -----------------------------
ar-DZ          Arabic (Algeria)
-------------- -----------------------------
ar-EG          Arabic (Egypt)
-------------- -----------------------------
ar-IQ          Arabic (Iraq)
-------------- -----------------------------
ar-IL          Arabic (Israel)
-------------- -----------------------------
ar-JO          Arabic (Jordan)
-------------- -----------------------------
ar-KW          Arabic (Kuwait)
-------------- -----------------------------
ar-LB          Arabic (Lebanon)
-------------- -----------------------------
ar-MA          Arabic (Morocco)
-------------- -----------------------------
ar-OM          Arabic (Oman)
-------------- -----------------------------
ar-PS          Arabic (Palestinian Territories)
-------------- -----------------------------
ar-QA          Arabic (Qatar)
-------------- -----------------------------
ar-SA          Arabic (Saudi Arabia)
-------------- -----------------------------
ar-TN          Arabic (Tunisia)
-------------- -----------------------------
ar-YE          Arabic (Yemen)
-------------- -----------------------------
az-AZ          Azerbaijani (Azerbaijan)
-------------- -----------------------------
bg-BG          Bulgarian (Bulgaria)
-------------- -----------------------------
bn-BD          Bengali (Bangladesh)
-------------- -----------------------------
bn-IN          Bengali (India)
-------------- -----------------------------
bs-BA          Bosnian (Bosnia and Herzegovina)
-------------- -----------------------------
ca-ES          Catalan (Spain)
-------------- -----------------------------
cs-CZ          Czech (Czech Republic)
-------------- -----------------------------
da-DK          Danish (Denmark)
-------------- -----------------------------
de-AT          German (Austria)
-------------- -----------------------------
de-CH          German (Switzerland)
-------------- -----------------------------
de-DE          German (Germany)
-------------- -----------------------------
el-GR          Greek (Greece)
-------------- -----------------------------
en-AU          English (Australia)
-------------- -----------------------------
en-CA          English (Canada)
-------------- -----------------------------
en-GB          English (United Kingdom)
-------------- -----------------------------
en-GH          English (Ghana)
-------------- -----------------------------
en-HK          English (Hong Kong)
-------------- -----------------------------
en-IE          English (Ireland)
-------------- -----------------------------
en-IN          English (India)
-------------- -----------------------------
en-KE          English (Kenya)
-------------- -----------------------------
en-NG          English (Nigeria)
-------------- -----------------------------
en-NZ          English (New Zealand)
-------------- -----------------------------
en-PH          English (Philippines)
-------------- -----------------------------
en-SG          English (Singapore)
-------------- -----------------------------
en-TZ          English (Tanzania)
-------------- -----------------------------
en-US          English (United States)
-------------- -----------------------------
en-ZA          English (South Africa)
-------------- -----------------------------
es-AR          Spanish (Argentina)
-------------- -----------------------------
es-BO          Spanish (Bolivia)
-------------- -----------------------------
es-CL          Spanish (Chile)
-------------- -----------------------------
es-CO          Spanish (Colombia)
-------------- -----------------------------
es-CR          Spanish (Costa Rica)
-------------- -----------------------------
es-DO          Spanish (Dominican Republic)
-------------- -----------------------------
es-EC          Spanish (Ecuador)
-------------- -----------------------------
es-ES          Spanish (Spain)
-------------- -----------------------------
es-GT          Spanish (Guatemala)
-------------- -----------------------------
es-HN          Spanish (Honduras)
-------------- -----------------------------
es-MX          Spanish (Mexico)
-------------- -----------------------------
es-NI          Spanish (Nicaragua)
-------------- -----------------------------
es-PA          Spanish (Panama)
-------------- -----------------------------
es-PE          Spanish (Peru)
-------------- -----------------------------
es-PR          Spanish (Puerto Rico)
-------------- -----------------------------
es-PY          Spanish (Paraguay)
-------------- -----------------------------
es-SV          Spanish (El Salvador)
-------------- -----------------------------
es-US          Spanish (United States)
-------------- -----------------------------
es-UY          Spanish (Uruguay)
-------------- -----------------------------
es-VE          Spanish (Venezuela)
-------------- -----------------------------
et-EE          Estonian (Estonia)
-------------- -----------------------------
eu-ES          Basque (Spain)
-------------- -----------------------------
fa-IR          Persian (Iran)
-------------- -----------------------------
fi-FI          Finnish (Finland)
-------------- -----------------------------
fil-PH         Filipino (Philippines)
-------------- -----------------------------
fr-BE          French (Belgium)
-------------- -----------------------------
fr-CA          French (Canada)
-------------- -----------------------------
fr-CH          French (Switzerland)
-------------- -----------------------------
fr-FR          French (France)
-------------- -----------------------------
gl-ES          Galician (Spain)
-------------- -----------------------------
gu-IN          Gujarati (India)
-------------- -----------------------------
he-IL          Hebrew (Israel)
-------------- -----------------------------
hi-IN          Hindi (India)
-------------- -----------------------------
hr-HR          Croatian (Croatia)
-------------- -----------------------------
hu-HU          Hungarian (Hungary)
-------------- -----------------------------
hy-AM          Armenian (Armenia)
-------------- -----------------------------
id-ID          Indonesian (Indonesia)
-------------- -----------------------------
is-IS          Icelandic (Iceland)
-------------- -----------------------------
it-CH          Italian (Switzerland)
-------------- -----------------------------
it-IT          Italian (Italy)
-------------- -----------------------------
ja-JP          Japanese (Japan)
-------------- -----------------------------
jv-ID          Javanese (Indonesia)
-------------- -----------------------------
ka-GE          Georgian (Georgia)
-------------- -----------------------------
kk-KZ          Kazakh (Kazakhstan)
-------------- -----------------------------
km-KH          Khmer (Cambodia)
-------------- -----------------------------
kn-IN          Kannada (India)
-------------- -----------------------------
ko-KR          Korean (South Korea)
-------------- -----------------------------
lo-LA          Lao (Laos)
-------------- -----------------------------
lt-LT          Lithuanian (Lithuania)
-------------- -----------------------------
lv-LV          Latvian (Latvia)
-------------- -----------------------------
mk-MK          Macedonian (North Macedonia)
-------------- -----------------------------
ml-IN          Malayalam (India)
-------------- -----------------------------
mn-MN          Mongolian (Mongolia)
-------------- -----------------------------
mr-IN          Marathi (India)
-------------- -----------------------------
ms-MY          Malay (Malaysia)
-------------- -----------------------------
my-MM          Burmese (Myanmar)
-------------- -----------------------------
ne-NP          Nepali (Nepal)
-------------- -----------------------------
nl-BE          Dutch (Belgium)
-------------- -----------------------------
nl-NL          Dutch (Netherlands)
-------------- -----------------------------
no-NO          Norwegian (Norway)
-------------- -----------------------------
pa-Guru-IN     Punjabi (Gurmukhi, India)
-------------- -----------------------------
pl-PL          Polish (Poland)
-------------- -----------------------------
pt-BR          Portuguese (Brazil)
-------------- -----------------------------
pt-PT          Portuguese (Portugal)
-------------- -----------------------------
ro-RO          Romanian (Romania)
-------------- -----------------------------
ru-RU          Russian (Russia)
-------------- -----------------------------
si-LK          Sinhala (Sri Lanka)
-------------- -----------------------------
sk-SK          Slovak (Slovakia)
-------------- -----------------------------
sl-SI          Slovenian (Slovenia)
-------------- -----------------------------
sq-AL          Albanian (Albania)
-------------- -----------------------------
sr-RS          Serbian (Serbia)
-------------- -----------------------------
su-ID          Sundanese (Indonesia)
-------------- -----------------------------
sv-SE          Swedish (Sweden)
-------------- -----------------------------
sw-KE          Swahili (Kenya)
-------------- -----------------------------
sw-TZ          Swahili (Tanzania)
-------------- -----------------------------
ta-IN          Tamil (India)
-------------- -----------------------------
ta-LK          Tamil (Sri Lanka)
-------------- -----------------------------
ta-MY          Tamil (Malaysia)
-------------- -----------------------------
ta-SG          Tamil (Singapore)
-------------- -----------------------------
te-IN          Telugu (India)
-------------- -----------------------------
th-TH          Thai (Thailand)
-------------- -----------------------------
tr-TR          Turkish (Turkey)
-------------- -----------------------------
uk-UA          Ukrainian (Ukraine)
-------------- -----------------------------
ur-IN          Urdu (India)
-------------- -----------------------------
ur-PK          Urdu (Pakistan)
-------------- -----------------------------
uz-UZ          Uzbek (Uzbekistan)
-------------- -----------------------------
vi-VN          Vietnamese (Vietnam)
-------------- -----------------------------
zh-CN          Chinese (Mandarin, Simplified)
-------------- -----------------------------
zh-HK          Chinese (Cantonese, Traditional)
-------------- -----------------------------
zh-TW          Chinese (Mandarin, Traditional)
-------------- -----------------------------
zu-ZA          Zulu (South Africa)
============== =============================

To ensure optimal transcription results, choose the correct code that best matches your speaker's language and dialect.
