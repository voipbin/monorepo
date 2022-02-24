.. _call-action: call-action

Action
======

.. _call-action-agent_call: call-action-agent_call

Agent Call
----------
Call to agent.

Parameters
++++++++++
.. code::

    {
        "type": "agent_call",
        "option": {
            "agent_id": "<string>"
        }
    }

* *agent_id*: target agent id.

Example
+++++++
.. code::

    {
        "type": "agent_call",
        "option": {
            "agent_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"
        }
    }

.. _call-action-amd: call-action-amd

AMD
---
Answering machine detection.

Parameters
++++++++++
.. code::

    {
        "type": "amd",
        "option": {
            "machine_handle": "<string>",
            "async": <boolean>
        }
    }

* *machine_handle*: hangup,delay,continue if the machine answered a call.
* *async*: if it's false, the call flow will be stop until amd done.

Example
+++++++
.. code::

    {
        "type": "amd",
        "option": {
            "machine_handle": "hangup",
            "sync": true
        }
    }

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

.. _call-action-branch: call-action-branch

Branch
------
Branch the flow.

Parameters
++++++++++
.. code::

    {
        "type": "branch",
        "option": {
            "default_index": <number>,
            "default_id": "<string>",
            "target_indexes": {
                "<string>": <string>,
                ...
            },
            "target_ids": {
                "<string>": <string>,
            }
        }
    }

* *default_index*: action index for default selection.
* *default_id*: action id for default selection. This will be generated automatically by the given default_index.
* *target_indexes*: set of input digit and target index fair.
* *target_ids*: set of input digit and target id fair. This will be generated automatically by the given target_indexes.

Example
+++++++
.. code::

    {
        "type": "branch",
        "option": {
            "default_index": 9,
            "target_indexes": {
                "1": 3,
                "2": 5,
                "3": 7
            }
        }
    }

.. _call-action-condition_digits: call-action-condition_digits

Condition Digits
----------------
Check the condition of received digits.
It checks the received digits and if it matched condition move to the next action. If not, move to the false_target_id.

Parameters
++++++++++
.. code::

    {
        "type": "condition_digits",
        "option": {
            "length": <number>,
            "key": "<string>",
            "false_target_index": <number>,
            "false_target_id": "<string>"
        }
    }

* *length*: match digits length.
* *key*: match digits contain.
* *false_target_index*: action index for false condition.
* *false_target_id*: action id for false condition. This will be generated automatically by the given false_target_index.

Example
+++++++
.. code::

    {
        "type": "condition_digits",
        "option": {
            "length": 10,
            "false_target_index": 3
        }
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

* *conference_id*: Conference's id to join.

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

Digits Receive
--------------
Receives the digits for given duration or numbers.

Parameters
++++++++++
.. code::

    {
        "type": "digits_receive",
        "option": {
            "duration": <number>,
            "length": <number>,
            "key": "<string>"
        }
    }

* *duration*: The duration allows you to set the limit (in ms) that VoIPBIN will wait for the endpoint to press another digit or say another word before it continue to the next action.
* *length*: You can set the number of DTMFs you expect. An optional limit to the number of DTMF events that should be gathered before continuing to the next action. By default, this is set to 1, so any key will trigger the next step. If EndKey is set and MaxNumKeys is unset, no limit for the number of keys that will be gathered will be imposed. It is possible for less keys to be gathered if the EndKey is pressed or the timeout being reached.
* *key*: If set, determines which DTMF triggers the next step. The finish_on_key will be included in the resulting variable. If not set, no key will trigger the next action.

Example
+++++++
.. code::

    {
        "type": "digits_receive",
        "option": {
            "duration": 10000,
            "length": 3,
            "key": "#"
        }
    }

Digits Send
-----------
Sends the digits with given duration and interval.

Parameters
++++++++++
.. code::

    {
        "type": "digits_send",
        "option": {
            "digits": "<string>",
            "duration": <number>,
            "interval": <number>
        }
    }

* *digits*: The digit string to send. Allowed set of characters: 0-9,A-D, #, '*'; with a maximum of 100 keys.
* *duration*: The duration of DTMF tone per key in milliseconds. Allowed values: Between 100 and 1000.
* *interval*: Interval between sending keys in milliseconds. Allowed values: Between 0 and 5000.

Example
+++++++
.. code::

    {
        "type": "digits_send",
        "option": {
            "digits": "1234567890",
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
            "duration": <number>,
        }
    }

* *duration*: Echo duration. ms.

Example
+++++++
.. code::

    {
        "type": "echo",
        "option": {
            "duration": 30000
        }
    }

Goto
----
Move the action execution.

Parameters
++++++++++
.. code::

    {
        "type": "goto",
        "option": {
            "target_index": <integer>,
            "target_id": "<string>",
            "loop": <boolean>,
            "loop_count": <integer>
        }
    }

* *target_index*: action index for move target.
* *target_id*: action id for move target. This will be generated automatically by the given default_index.
* *loop*: It this set to true, will loop only number of loop_count.
* *loop_count*: The number of loop.

Example
+++++++
.. code::

    {
        "type": "goto",
        "option": {
            "target_index": 0,
            "loop": true,
            "loop_count": 2
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

Patch Flow
----------
Patch the next flow from the existed flow.

Parameters
++++++++++
.. code::

    {
        "type": "patch_flow",
        "option": {
            "flow_id": "<string>"
        }
    }

* *flow_id*: The id of flow.

Example
+++++++
.. code::

    {
        "type": "patch_flow".
        "option": {
            "flow_id": "212a32a8-9529-11ec-8bf0-8b89df407b6e"
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

Queue Join
----------
Join to the queue.

Parameters
++++++++++
.. code::

    {
        "type": "queue_join",
        "option": {
            "queue_id": "<string>"
        }
    }

* *queue_id*: Target queue id.

Example
+++++++
.. code::

    {
        "type": "queue_join",
        "option": {
            "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb"
        }
    }

Recording Start
---------------
Starts the call recording.

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
Stops the call recording.

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

Stream Echo
-----------
Echoing the RTP stream including the digits receive.

Parameters
++++++++++
.. code::

    {
        "type": "stream_echo",
        "option": {
            "duration": <number>
        }
    }

* *duration*: Echo duration. ms.

Example
+++++++
.. code::

    {
        "type": "stream_echo"
        "option": {
            "duration": 10000
        }
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

Transcribe_start
----------------
Start the STT(Speech to text) transcribe in realtime.

Parameters
++++++++++
.. code::

    {
        "type": "transcribe_start",
        "option": {
            "language": "<string>",
        }
    }

* *language*: Specifies the language. BCP47 format. The value may contain a lowercase, two-letter language code (for example, en), or the language code and uppercase country/region (for example, en-US).

Example
+++++++
.. code::

    {
        "type": "transcribe_start",
        "option": {
            "language": "en-US",
        }
    }

Transcribe_stop
---------------
Stop the transcribe talk in realtime.

Parameters
++++++++++
.. code::

    {
        "type": "transcribe_stop"
    }

Example
+++++++
.. code::

    {
        "type": "transcribe_stop"
    }
