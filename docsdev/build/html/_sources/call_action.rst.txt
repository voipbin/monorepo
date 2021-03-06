.. _call-action: call-action

Action
======

.. _call-action-answer: call-action-answer

Answer
------
Answer the call

Parameters
++++++++++
.. code::

    {
        "type": "answer"
    }

Example
+++++++
.. code::

    {
        "type": "answer"
    }

.. _call-action-conference_join: call-action-conference_join

Conference Join
---------------
Join to the conference

Parameters
++++++++++
.. code::

    {
        "type": "conference_join",
        "option": {
            "conference_id": "<string>"
        }
    }

* *conference_id*: conference's id to join.

Example
+++++++
.. code::

    {
        "type": "conference_join",
        "option": {
            "conference_id": "367e0e7a-3a8c-11eb-bb08-f3c3f059cfbe"
        }
    }

.. _call-action-connect: call-action-connect

Connect
-------
Originate to the other destination(s) and connect to them each other.

Parameters
++++++++++
.. code::

    {
        "type": "connect",
        "option": {
            "source": {...},
            "destinations": [
                {...},
                ...
            ]
            "unchained": <boolean>
        }
    }

* *source*: Source address.
* *destinations*: Destination addresses.
* *unchained*: If it sets to false, connected destination calls will be hungup when the master call is hangup. Default false.

Example
+++++++
.. code::

    {
        "type": "connect",
        "option": {
            "source": {
                "type": "tel",
                "target": "+11111111111111"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+222222222222222"
                }
            ]
        }
    }

DTMF Receive
------------
Receives the DTMFs for given duration or numbers.

Parameters
++++++++++
.. code::

    {
        "type": "dtmf_receive",
        "option": {
            "max_number_key": <number>,
            "duration": <number>,
            "finish_on_key": "<string>"
        }
    }

* *max_number_key*: You can set the number of DTMFs you expect. An optional limit to the number of DTMF events that should be gathered before continuing to the next action. By default, this is set to 1, so any key will trigger the next step. If EndKey is set and MaxNumKeys is unset, no limit for the number of keys that will be gathered will be imposed. It is possible for less keys to be gathered if the EndKey is pressed or the timeout being reached.
* *duration*: The duration allows you to set the limit (in ms) that VoIPBIN will wait for the endpoint to press another digit or say another word before it continue to the next action.
* *finish_on_key*: If set, determines which DTMF triggers the next step. The finish_on_key will be included in the resulting variable. If not set, no key will trigger the next action.

Example
+++++++
.. code::

    {
        "type": "dtmf_receive",
        "option": {
            "max_number_key": 3,
            "duration": 10000,
            "finish_on_key": "#"
        }
    }

DTMF Send
---------
Sends the DTMFs with given duration and interval.

Parameters
++++++++++
.. code::

    {
        "type": "dtmf_send",
        "option": {
            "dtmfs": "<string>",
            "duration": <number>,
            "interval": <number>
        }
    }

* *dtmfs*: The dtmf string to send. Allowed set of characters: 0-9,A-D, #, '*'; with a maximum of 100 keys.
* *duration*: The duration of DTMF tone per key in milliseconds. Allowed values: Between 100 and 1000.
* *finish_on_key*: Interval between sending keys in milliseconds. Allowed values: Between 0 and 5000.

Example
+++++++
.. code::

    {
        "type": "dtmf_send",
        "option": {
            "dtmfs": "1234567890",
            "duration": 500,
            "interval": 500
        }
    },

Echo
----
Echoing the call.

Parameters
++++++++++
.. code::

    {
        "type": "echo",
        "option": {
            "duration": <integer>,
            "dtmf": <boolean>
        }
    }

* *duration*: Echo duration. ms.
* *dtmf*: Sending back the DTMF.

Example
+++++++
.. code::

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
.. code::

    {
        "type": "hangup"
    }

Example
+++++++
.. code::

    {
        "type": "hangup"
    }

.. _call-action-patch: call-action-patch

Patch
-----
Patch the next flow from the remote.

Parameters
++++++++++
.. code::

    {
        "type": "patch",
        "option": {
            "event_url": "<string>",
            "event_method": "<string>"
        }
    }

* *event_url*: The url for flow patching.
* *event_method*: The method for flow patching.

Example
+++++++
.. code::

    {
        "type": "patch".
        "option": {
            "event_url": "https://webhook.site/e47c9b40-662c-4d20-a288-6777360fa211"
        }
    }

Play
----
Plays the linked file.

Parameters
++++++++++
.. code::

    {
        "type": "play",
        "option": {
            "stream_urls": [
                "<string>",
                ...
            ]
        }
    }

* *stream_urls*: Stream url array for media.

Example
+++++++
.. code::

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
.. code::

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

* *format*: Format to encode audio in. wav, mp3, ogg.
* *end_of_silence*: Maximum duration of silence, in seconds. 0 for no limit.
* *end_of_key*: DTMF input to terminate recording. none, any, \*, #.
* *duration*: Maximum duration of the recording, in seconds. 0 for no limit.
* *beep_start*: Play beep when recording begins

Example
+++++++
.. code::

    {
        "type": "recording_start",
        "option": {
            "format": "wav"
        }
    }

Recording Stop
--------------

Parameters
++++++++++
.. code::

    {
        "type": "recording_stop"
    }

Example
+++++++
.. code::

    {
        "type": "recording_stop"
    }

Talk
----
Text to speech. SSML(https://www.w3.org/TR/speech-synthesis/) supported.

Parameters
++++++++++
.. code::

    {
        "type": "talk",
        "option": {
            "text": "<string>",
            "gender": "<string>",
            "language": "<string>"
        }
    }

* *text*: Text to speech. SSML(https://cloud.google.com/text-to-speech/docs/ssml) supported.
* *gender*: male/female.
* *language*: Specifies the language. The value may contain a lowercase, two-letter language code (for example, en), or the language code and uppercase country/region (for example, en-US).

Example
+++++++
.. code::

    {
        "type": "talk",
        "option": {
            "text": "Hello. Welcome to voipbin. This is test message. Please enjoy the voipbin service. Thank you. Bye",
            "gender": "female",
            "language": "en-US"
        }
    }
