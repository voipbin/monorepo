.. _flow-action:

Action
======

.. _flow-action-action:

Action
------

.. code::

    {
        "id": "<string>>",
        "next_id": "<string>>",
        "type": "<string>>",
        "option": {
            ...
        },
        "tm_execute": "<string>>"
    }

* *id*: Action's id.
* *next_id*: Action's next id. If it sets empty, just move on to the next action in the action array.
* *type*: Action's type. See detail :ref:`here <flow-struct-action-type>`.
* *option*: Action's option.

.. _flow-action-action-type:

Action type
-----------

======================= ==================
type                    Description
======================= ==================
agent_call              Make a call to the agent.
amd                     Answering machine detection.
answer                  Answer the call.
beep                    Play the beep sound.
branch                  Branch the flow
condition_call_digits   Condition check(call's digits)
condition_call_status   Condition check(call's status)
confbridge_join         Join to the confbridge.
conference_join         Join to the conference.
connect                 Connect to the other destination.
digits_receive          Receive the digits(dtmfs).
digits_send             Send the digits(dtmfs).
echo                    Echo to stream.
external_media_start    Start the external media.
external_media_stop     Stop the external media.
goto                    Goto.
hangup                  Hangup the call.
message_send            Send a message.
patch                   Patch the actions from endpoint.
patch_flow              Patch the actions from the exist flow.
play                    Play the file.
queue_join              Join to the queue.
recording_start         Startr the record of the given call.
recording_stop          Stop the record of the given call.
sleep                   Sleep.
stream_echo             Echo the steam.
talk                    Generate audio from the given text(ssml or plain text) and play it.
transcribe_start        Start transcribe the call
transcribe_stop         Stop transcribe the call
transcribe_recording    Transcribe the recording and send it to webhook.
======================= ==================

.. _flow-actions:

Actions
=======
List of actions

.. _flow-action-agent_call:

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

.. _flow-action-amd: flow-action-amd

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

* *machine_handle*: hangup,delay,continue if the machine answered a call. See detail :ref:`here <flow-action-amd-machinehandle>`.
* *async*: if it's false, the call flow will be stop until amd done.

.. _flow-action-amd-machinehandle:

Machine handle
++++++++++++++
======== ==============
Type     Description
======== ==============
hangup   Hangup the call.
continue Continue the call.
======== ==============

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

.. _flow-action-answer:

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

.. _flow-action-beep:

Beep
------
Make a beep sound.

Parameters
++++++++++
.. code::

    {
        "type": "beep"
    }

Example
+++++++
.. code::

    {
        "type": "beep"
    }


.. _flow-action-branch:

Branch
------
Branch the flow.

Parameters
++++++++++
.. code::

    {
        "type": "branch",
        "option": {
            "default_target_id": "<string>",
            "target_ids": {
                "<string>": <string>,
            }
        }
    }

* *default_target_id*: action id for default selection. This will be generated automatically by the given default_index.
* *target_ids*: set of input digit and target id fair. This will be generated automatically by the given target_indexes.

Example
+++++++
.. code::

    {
        "type": "branch",
        "option": {
            "default_target_id": "779f2580-9c8b-11ec-ae23-0bb6b66c8a86",
            "target_indexes": {
                "1": "78dcfb2a-9c8b-11ec-b0fd-cb5d94b8bd32",
                "2": "7903829a-9c8b-11ec-8aa5-978e3a1124e9",
                "3": "792b07c0-9c8b-11ec-8407-dbe1a113664c"
            }
        }
    }

.. _flow-action-confbridge_join:

Confbridge Join
----------------
Join to the confbridge.

Parameters
++++++++++
.. code::

    {
        "type": "confbridge_join",
        "option": {
            "confbridge_id": "<string>"
        }
    }

* *confbridge_id*: Target confbridge id.


.. _flow-action-condition_call_digits:

Condition Call Digits
---------------------
Check the condition of received digits.
It checks the received digits and if it matched condition move to the next action. If not, move to the false_target_id.

Parameters
++++++++++
.. code::

    {
        "type": "condition_call_digits",
        "option": {
            "length": <number>,
            "key": "<string>",
            "false_target_id": "<string>"
        }
    }

* *length*: match digits length.
* *key*: match digits contain.
* *false_target_id*: action id for false condition.

Example
+++++++
.. code::

    {
        "type": "condition_call_digits",
        "option": {
            "length": 10,
            "false_target_id": "e3e50e6c-9c8b-11ec-8031-0384a8fcd1e2"
        }
    }

.. _flow-action-condition_call_status:

Condition Call Status
---------------------
Check the condition of call's status.
It checks the call's status and if it matched with condition then move to the next action. If not, move to the false_target_id.

Parameters
++++++++++
.. code::

    {
        "type": "condition_call_status",
        "option": {
            "status": <number>,
            "false_target_id": "<string>"
        }
    }

* *status*: match call's status. See detail :ref:`here <call-struct-status>`.
* *false_target_id*: action id for false condition.

Example
+++++++
.. code::

    {
        "type": "condition_call_status",
        "option": {
            "status": "progressing,
            "false_target_id": "e3e50e6c-9c8b-11ec-8031-0384a8fcd1e2"
        }
    }


.. _flow-action-conference_join:

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

.. _flow-action-connect:

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
                ...
            ]
            "unchained": <boolean>
        }
    }

* *source*: Source address. See detail :ref:`here <call-struct-address>`.
* *destinations*: Array of destination addresses. See detail :ref:`here <call-struct-address>`.
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

.. _flow-action-digits_receive:

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

.. _flow-action-digits_send:

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

.. _flow-action-echo:

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

.. _flow-action-external_media_start:

External Media Start
--------------------
Start the external media.

Parameters
++++++++++
.. code::

    {
        "type": "external_media_start",
        "option": {
            "external_host": "<string>",
            "encapsulation": "<string>",
            "transport": "<string>",
            "connection_type": "<string>",
            "format": "<string>",
            "direction": "<string>",
            "data": "<string>"
        }
    }

* *external_host*: external media target host address.
* *encapsulation*: encapsulation. default: rtp.
* *transport*: transport. default: udp.
* *connection_type*: connection type. default: client
* *format*: format default: ulaw
* *direction*: Direction. default: both.
* *data*: Data. Reserved.

.. _flow-action-external_media_stop:

External Media Stop
--------------------
Stop the external media.

Parameters
++++++++++
.. code::

    {
        "type": "external_media_stop",
    }

.. _flow-action-goto:

Goto
----
Move the action execution.

Parameters
++++++++++
.. code::

    {
        "type": "goto",
        "option": {
            "target_id": "<string>",
            "loop_count": <integer>
        }
    }

* *target_id*: action id for move target.
* *loop_count*: The number of loop.

Example
+++++++
.. code::

    {
        "type": "goto",
        "option": {
            "target_id": "ca4ddd74-9c8d-11ec-818d-d7cf1487e8df",
            "loop_count": 2
        }
    }

.. _flow-action-hangup:

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

.. _flow-action-message_send:

Message send
------------
Send a message.

Parameters
++++++++++
.. code::

    {
        "type": "message_send",
        "option": {
            "source": {
                ...
            },
            "destinations": [
                {
                    ...
                },
                ...
            ],
            "text": "<string>"
        }
    }

* *source*: Source address info. See detail :ref:`here <call-struct-address>`.
* *destinations*: Array of destination addresses. See detail :ref:`here <call-struct-address>`.
* text: Message's text.

.. _flow-action-patch: flow-action-patch

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

.. _flow-action-patch_flow:

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

.. _flow-action-play:

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

.. _flow-action-queue_join:

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

.. _flow-action-recording_start:

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

.. _flow-action-recording_stop:

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

.. _flow-action-sleep:

Sleep
--------------
Sleep the call.

Parameters
++++++++++
.. code::

    {
        "type": "sleep",
        "option": {
            "duration": <number>
        }
    }

* duration: Sleep duration(ms).

.. _flow-action-stream_echo:

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

.. _flow-action-talk:

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

.. _flow-action-transribe_start:

Transcribe Start
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

.. _flow-action-transcribe_stop:

Transcribe Stop
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

