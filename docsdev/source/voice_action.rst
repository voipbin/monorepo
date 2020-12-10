.. voice_action:

Action
======

Answer
------
Answer the call

Parameters
++++++++++
::

    {
        "type": "answer"
    }

Example
+++++++
::

    {
        "type": "answer"
    }

Conference Join
---------------
Join to the conference

Parameters
++++++++++
::

    {
        "type": "conference_join",
        "option": {
            "conference_id": "<string>"
        }
    }

* conference_id<string>: conference's id to join.

Example
+++++++
::

    {
        "type": "conference_join",
        "option": {
            "conference_id": "367e0e7a-3a8c-11eb-bb08-f3c3f059cfbe"
        }
    }

Connect
-------
Originate to the other destination(s) and connect to them each other.

Parameters
++++++++++
::

    {
        "type": "connect",
        "option": {
            "from": "<string>",
            "destinations": [
                "<string>",
                ...
            ]
            "unchained": <boolean>
        }
    }

* from: From number.
* destinations: Target destinations.
* unchained: If it sets to false, connected destination calls will be hungup when the master call is hangup. Default false.

Example
+++++++
::

    {
        "type": "connect",
        "option": {
            "from": "+1111111111111",
            "destinations": [
                {
                    "type": "tel",
                    "target": "+222222222222222"
                }
            ]
        }
    }

Echo
----
Echoing the voice.

Parameters
++++++++++
::

    {
        "type": "echo",
        "option": {
            "duration": <integer>,
            "dtmf": <boolean>
        }
    }

* duration: Echo duration. ms.
* dtmf: Sending back the DTMF.

Example
+++++++
::

    {
        "type": "echo",
        "option": {
            "duration": 30000
        }
    }

Hangup
------
Hangup the call.

Parameters
++++++++++
::

    {
        "type": "hangup"
    }

Example
+++++++
::

    {
        "type": "hangup"
    }

Play
----
Plays the linked file.

Parameters
++++++++++
::

    {
        "type": "play",
        "option": {
            "stream_urls": [
                "<string>",
                ...
            ]
        }
    }

* stream_urls: Stream url array for media.

Example
+++++++
::

    {
        "type": "play",
        "option": {
            "stream_urls": [
                "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
            ]
        }
    }

Recording Start
---------------

Parameters
++++++++++
::

    {
        "type": "recording_start"
        "option": {
            "format": "<string>",
            "end_of_silence": <integer>,
            "end_of_key": "<string>",
            "duration": <integer>,
            "beep_start": <boolean>
        }
    }

* format: Format to encode audio in. wav, mp3, ogg.
* end_of_silence: Maximum duration of silence, in seconds. 0 for no limit.
* end_of_key: DTMF input to terminate recording. none, any, \*, #.
* duration: Maximum duration of the recording, in seconds. 0 for no limit.
* beep_start: Play beep when recording begins

Example
+++++++
::

    {
        "type": "record_start",
        "option": {
            "format": "wav"
        }
    }

Recording Stop
--------------

Parameters
++++++++++
::

    {
        "type": "recording_stop"
    }

Example
+++++++
::

    {
        "type": "recording_stop"
    }

Talk
----
Text to speech. SSML(https://www.w3.org/TR/speech-synthesis/) supported.

Parameters
++++++++++
::

    {
        "type": "talk",
        "option": {
            "text": "<string>",
            "gender": "<string>",
            "language": "<string>"
        }
    }

* text: Text to speech. SSML(https://cloud.google.com/text-to-speech/docs/ssml) supported.
* gender: male/female.
* language: Specifies the language. The value may contain a lowercase, two-letter language code (for example, en), or the language code and uppercase country/region (for example, en-US).

Example
+++++++
::

    {
        "type": "talk",
        "option": {
            "text": "Hello. Welcome to voipbin. This is test message. Please enjoy the voipbin service. Thank you. Bye",
            "gender": "female",
            "language": "en-US"
        }
    }
