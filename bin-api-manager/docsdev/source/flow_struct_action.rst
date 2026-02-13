.. _flow-struct-action:

Action
======

.. _flow-struct-action-action:

Action
------

.. code::

    {
        "id": "<string>",
        "next_id": "<string>",
        "type": "<string>",
        "option": {
            ...
        },
        "tm_execute": "<string>"
    }

* *id*: Action's id.
* *next_id*: Action's next id. If it sets empty, just move on to the next action in the action array.
* *type*: Action's type. See detail :ref:`here <flow-struct-action-type>`.
* *option*: Action's option.

.. _flow-struct-action-type:

Type
----

======================= ==================
type                    Description
======================= ==================
agent_call              **Deprecated**. Use the *connect* instead. Creates a call to the agent and connect.
amd                     Answering machine detection.
answer                  Answer the call.
beep                    Play the beep sound.
branch                  Branch gets the variable then execute the correspond action. For example. gets the dtmf input saved variable and jump to the action.
call                    Starts a new independent outgoing call with a given flow.
chatbot_talk            Starts a talk with chatbot.
condition_call_digits   **Deprecated**. Use the `condition_variable` instead. Condition check(call's digits).
condition_call_status   **Deprecated**. Use the `condition_variable` instead. Condition check(call's status).
condition_datetime      Condition check(time)
condition_variable      Condition check(variable).
confbridge_join         Join to the confbridge.
conference_join         Join to the conference.
connect                 Creates a new call to the destinations and connects to them.
conversation_send       Send a message to the conversation.
digits_receive          Receive the digits(dtmfs).
digits_send             Send the digits(dtmfs).
echo                    Echo to stream.
external_media_start    Start the external media.
external_media_stop     Stop the external media.
fetch                   Fetch the actions from endpoint.
fetch_flow              Fetch the actions from the exist flow.
goto                    Goto.
hangup                  Hangup the call.
hangup_relay            Hangs up the call with the same reason of the given reference id.
message_send            Send a message.
play                    Play the file of the given urls.
queue_join              Join to the queue.
recording_start         Start the record of the given call.
recording_stop          Stop the record of the given call.
sleep                   Sleep.
stop                    Stop the flow.
stream_echo             Echo the steam.
talk                    Generate audio from the given text(ssml or plain text) and play it.
transcribe_start        Start transcribe the call
transcribe_stop         Stop transcribe the call
transcribe_recording    Transcribe the recording and send it to webhook.
variable_set            Sets the variable.
webhook_send            Send a webhook.
======================= ==================

.. _flow-struct-action-agent_call:

Agent Call
----------
Calling the agent.
The agent may have various types of addresses or phone numbers, such as a desk phone, mobile phone, or softphone application.

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

.. _flow-struct-action-amd: flow-struct-action-amd

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

* *machine_handle*: hangup,delay,continue if the machine answered a call. See detail :ref:`here <flow-struct-action-amd-machinehandle>`.
* *async*: if it's false, the call flow will be stop until amd done.

.. _flow-struct-action-amd-machinehandle:

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

.. _flow-struct-action-answer:

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

.. _flow-struct-action-beep:

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


.. _flow-struct-action-branch:

Branch
------
Branch the flow.
It gets the variable from the activeflow and move the activeflow cursor to the selected target id.

Parameters
++++++++++
.. code::

    {
        "type": "branch",
        "option": {
            "variable": "<string>",
            "default_target_id": "<string>",
            "target_ids": {
                "<string>": <string>,
            }
        }
    }

* *variable*: Target variable. If this value is empty, default target variable will be selected. Available variables are listed :ref:`here <variable-variable>`. default: voipbin.call.digits
* default_target_id: action id for default selection. This will be generated automatically by the given default_index.
* target_ids: set of input digit and target id fair. This will be generated automatically by the given target_indexes.

Example
+++++++
.. code::

    {
        "type": "branch",
        "option": {
            "default_target_id": "ed9705ca-c524-11ec-a3fb-8feb7731ad45",
            "target_ids": {
                "1": "c3eb8e62-c524-11ec-94c5-abafec8af561",
                "2": "dc87123e-c524-11ec-89c6-5fb18da14034",
                "3": "e70fb030-c524-11ec-b657-ebec72f097ef"
            }
        }
    }

.. _flow-struct-action-call:

Call
----
Make a new outbound call in a new context.

.. image:: _static/images/flow_action_call.png

Parameters
++++++++++
.. code::

    {
        "type": "call",
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
            "flow_id": "<string>"
            "actions": [
                {
                    ...
                }
            ],
            "chained": <boolean>,
            "early_execution": <boolean>
        }
    }

* *source*: Source address. See detail :ref:`here <common-struct-address-type>`.
* *destinations*: Array of destination addresses. See detail :ref:`here <common-struct-address-type>`.
* flow_id: Call's flow id. If this not set, will use the actions array.
* actions: Array of actions. If the flow_id not set, the call flow will be created with this actions.
* chained: If it sets to true, created calls will be hungup when the master call is hangup. Default false.
* early_execution: It it sets to true, the voipbin will execute the flow when then call is ringing.

Example
+++++++
.. code::

    {
        "id": "e34ab97a-c53a-4eb4-aebf-36767a528f00",
        "next_id": "00000000-0000-0000-0000-000000000000",
        "type": "call",
        "option": {
            "source": {
                "type": "tel",
                "target": "+821100000001"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+821100000002"
                }
            ],
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "hello, this is test message.",
                        "language": "en-US"
                    }
                }
            ],
            "chained": false
        }
    }

.. _flow-struct-action-chatbot_talk:

Chatbot Talk
----------------
Start the chatbot talk.

Parameters
++++++++++
.. code::

    {
        "type": "chatbot_talk",
        "option": {
            "chatbot_id": "<string>",
            "gender": "<string>",
            "language": "<string>",
            "duration": <number>
        }
    }

* chatbot_id: Chatbot id.
* gender: Voice gender. male/female/neutral
* language: Specifies the language. BCP47 format.
* duration: Duration. Seconds.



.. _flow-struct-action-confbridge_join:

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

.. _flow-struct-action-condition_call_digits:

Condition Call Digits
---------------------
Deprecated. Use the condition_variable instead.
Check the condition of received digits.
If the conditions are met, the system proceeds to the next action.
If the conditions are not met, the voipbin directs the call to a false_target_id for further processing.

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

* length: match digits length.
* key: match digits contain.
* false_target_id: action id for false condition.

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

.. _flow-struct-action-condition_call_status:

Condition Call Status
---------------------
Deprecated. Use the condition_variable instead.
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

* *status*: match call's status. See detail :ref:`here <call-struct-call-status>`.
* false_target_id: action id for false condition.

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

.. _flow-struct-action-condition_datetime:

Condition Datetime
---------------------
Check the condition of the time.
It checks the current time(UTC) and if it matched with condition then move to the next action. If not, move to the false_target_id.

Parameters
++++++++++
.. code::

    {
        "type": "condition_datetime",
        "option": {
            "condition": <number>,

            "minute": <number>,
            "hour": <number>,
            "day": <number>,
            "month": <number>,
            "weekdays": [
                <number>,
                ...
            ],


            "false_target_id": "<string>"
        }
    }

* *condition*: Match condition. One of "==", "!=", ">", ">=", "<", "<=".
* minute: Minutes. -1 for all minutes.
* hour: Hour. -1 for all hours.
* day: Day. -1 for all days.
* month: Month. 0 for all months.
* weekdays: List of weekdays. Sunday: 0, Monday: 1, Tuesday: 2, Wednesday: 3, Thursday: 4, Friday: 5, Saturday: 6
* false_target_id: action id for false condition.

Example
+++++++
.. code::

    {
        "type": "condition_datetime",
        "option": {
            "condition": ">=,
            "minute": 0
            "hour": 8,
            "day": -1,
            "month": 0,
            "weekdays": [],
            "false_target_id": "d08582ee-1b3d-11ed-a43e-9379f27c3f7f"
        }
    }

.. _flow-struct-action-condition_variable:

Condition Variable
---------------------
Check the condition of the given variable.
It checks the call's status and if it matched with condition then move to the next action. If not, move to the false_target_id.

Parameters
++++++++++
.. code::

    {
        "type": "condition_variable",
        "option": {
            "condition": "<string>",
            "variable": "<string>",
            "value_type": "<string>",
            "value_string": "<string>",
            "value_number": <number>,
            "value_length": <number>,
            "false_target_id": "<string>"
        }
    }

* *condition*: Match condition. One of "==", "!=", ">", ">=", "<", "<=".
* *variable*: Target variable. See detail :ref:`here <variable-variable>`.
* value_type: Type of value. string/number/length.
* value_string: Value. Valid only if the value_type is string.
* value_number: Value. Valid only if the value_type is number.
* value_length: Value. Valid only if the value_type is length.
* false_target_id: action id for false condition.

Example
+++++++
.. code::

    {
        "type": "condition_variable",
        "option": {
            "condition": "==",
            "variable": "voipbin.call.source.target",
            "value_type": "string",
            "value_string": "+821100000001",
            "false_target_id": "fb2f4e2a-b030-11ed-bddb-976af892f5a3"
        }
    }

.. _flow-struct-action-conference_join:

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

* conference_id: Conference's id to join.

Example
+++++++
.. code::

    {
        "type": "conference_join",
        "option": {
            "conference_id": "367e0e7a-3a8c-11eb-bb08-f3c3f059cfbe"
        }
    }

.. _flow-struct-action-connect:

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
            ],
            "early_media": <boolean>,
            "relay_reason": <boolean>
        }
    }

* *source*: Source address. See detail :ref:`here <common-struct-address-address>`.
* *destinations*: Array of destination addresses. See detail :ref:`here <common-struct-address-address>`.
* early_media: Support early media.
* relay_reason: relay the hangup reason to the master call.

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

.. _flow-struct-action-conversation_send:

Conversation send
-----------------
Send the message to the conversation.

Parameters
++++++++++
.. code::

    {
        "type": "conversation_send",
        "option": {
            "conversation_id": "<string>",
            "text": "<string>",
            "sync": <boolean>
        }
    }

* conversation_id: Target conversation id.
* text: Send text message.
* sync: If this set to true, waits until this action done.

Example
+++++++
.. code::

    {
        "type": "conversation_send",
        "option": {
            "conversation_id": "b5ef5e64-f7ca-11ec-bbe9-9f74186a2a72",
            "text": "hello world, this is test message.",
            "sync": false
        }
    }

.. _flow-struct-action-digits_receive:

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

* duration: The duration allows you to set the limit (in ms) that VoIPBIN will wait for the endpoint to press another digit or say another word before it continue to the next action.
* length: You can set the number of DTMFs you expect. An optional limit to the number of DTMF events that should be gathered before continuing to the next action. By default, this is set to 1, so any key will trigger the next step. If EndKey is set and MaxNumKeys is unset, no limit for the number of keys that will be gathered will be imposed. It is possible for less keys to be gathered if the EndKey is pressed or the timeout being reached.
* key: If set, determines which DTMF triggers the next step. The finish_on_key will be included in the resulting variable. If not set, no key will trigger the next action.

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

.. _flow-struct-action-digits_send:

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

* digits: The digit string to send. Allowed set of characters: 0-9,A-D, #, '*'; with a maximum of 100 keys.
* duration: The duration of DTMF tone per key in milliseconds. Allowed values: Between 100 and 1000.
* interval: Interval between sending keys in milliseconds. Allowed values: Between 0 and 5000.

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

.. _flow-struct-action-echo:

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

.. _flow-struct-action-external_media_start:

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

* external_host: external media target host address.
* encapsulation: encapsulation. default: rtp.
* transport: transport. default: udp.
* connection_type: connection type. default: client
* format: format default: ulaw
* direction: Direction. default: both.
* data: Data. Reserved.

.. _flow-struct-action-external_media_stop:

External Media Stop
--------------------
Stop the external media.

Parameters
++++++++++
.. code::

    {
        "type": "external_media_stop",
    }

.. _flow-struct-action-fetch: flow-struct-action-fetch

Fetch
-----
Fetch the next flow from the remote.

Parameters
++++++++++
.. code::

    {
        "type": "fetch",
        "option": {
            "event_url": "<string>",
            "event_method": "<string>"
        }
    }

* event_url: The url for flow fetching.
* event_method: The method for flow fetching.

Example
+++++++
.. code::

    {
        "type": "fetch",
        "option": {
            "event_method": "POST",
            "event_url": "https://webhook.site/e47c9b40-662c-4d20-a288-6777360fa211"
        }
    }

.. _flow-struct-action-fetch_flow:

Fetch Flow
----------
Fetch the next flow from the existing flow.

Parameters
++++++++++
.. code::

    {
        "type": "fetch_flow",
        "option": {
            "flow_id": "<string>"
        }
    }

* *flow_id*: The id of flow.

Example
+++++++
.. code::

    {
        "type": "fetch_flow",
        "option": {
            "flow_id": "212a32a8-9529-11ec-8bf0-8b89df407b6e"
        }
    }

.. _flow-struct-action-goto:

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

* target_id: action id for move target.
* loop_count: The number of loop.

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

.. _flow-struct-action-hangup:

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

.. _flow-struct-action-hangup_relay:

Hangup Relay
-------------
Hangup the call and relay the hangup cause to the reference id.

Parameters
++++++++++
.. code::

    {
        "type": "hangup_relay",
        "option": {
            "reference_id": "<string>"
        }
    }

Example
+++++++
.. code::

    {
        "type": "hangup_relay",
        "option": {
            "reference_id": "b8573f30-b031-11ed-ac05-3bc9a62e64c3"
        }
    }

.. _flow-struct-action-message_send:

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

* *source*: Source address info. See detail :ref:`here <common-struct-address-address>`.
* *destinations*: Array of destination addresses. See detail :ref:`here <common-struct-address-address>`.
* text: Message's text.

.. _flow-struct-action-play:

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

* stream_urls: Stream url array for media.

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

.. _flow-struct-action-queue_join:

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

* queue_id: Target queue id.

Example
+++++++
.. code::

    {
        "type": "queue_join",
        "option": {
            "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb"
        }
    }

.. _flow-struct-action-recording_start:

Recording Start
---------------
Starts the call recording.

Parameters
++++++++++
.. code::

    {
        "type": "recording_start",
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
.. code::

    {
        "type": "recording_start",
        "option": {
            "format": "wav"
        }
    }

.. _flow-struct-action-recording_stop:

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

.. _flow-struct-action-sleep:

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

.. _flow-struct-action-stream_echo:

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
        "type": "stream_echo",
        "option": {
            "duration": 10000
        }
    }

.. _flow-struct-action-talk:

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
            "language": "<string>",
            "provider": "<string>",
            "voice_id": "<string>",
            "digits_handle": "<string>"
        }
    }

* text: Text to speech. SSML(https://cloud.google.com/text-to-speech/docs/ssml) supported.
* language: Specifies the language. The value may contain a lowercase, two-letter language code (for example, en), or the language code and uppercase country/region (for example, en-US).
* provider: TTS provider. Optional. ``gcp`` (Google Cloud TTS) or ``aws`` (AWS Polly). If omitted, defaults to GCP. If the selected provider fails, the system automatically falls back to the alternative provider with the default voice for the language.
* voice_id: Provider-specific voice identifier. Optional. For example, ``en-US-Wavenet-D`` (GCP) or ``Joanna`` (AWS). On fallback, the voice_id is reset to the alternative provider's default voice.
* digits_handle: See detail :ref:`here <flow-struct-action-talk-digits_handle>`.

.. _flow-struct-action-talk-digits_handle:

Digits handle
++++++++++++++
======== ==============
Type     Description
======== ==============
next     If digits received in talk, move the action to the next.
======== ==============


Example
+++++++
.. code::

    {
        "type": "talk",
        "option": {
            "text": "Hello. Welcome to voipbin. This is test message. Please enjoy the voipbin service. Thank you. Bye",
            "language": "en-US"
        }
    }

.. _flow-struct-action-transcribe_recording:

Transcribe recording
--------------------
Transcribe the call's recordings.

Parameters
++++++++++
.. code::

    {
        "type": "transcribe_recording",
        "option": {
            "language": "<string>"
        }
    }

* language: Specifies the language. BCP47 format.

Example
+++++++
.. code::

    {
        "type": "transcribe_recording",
        "option": {
            "language": "en-US"
        }
    }

.. _flow-struct-action-transribe_start:

Transcribe start
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

.. _flow-struct-action-transcribe_stop:

Transcribe stop
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

.. _flow-struct-action-variable_set:

Variable Set
---------------
Set a variable value for use in the flow.

Parameters
++++++++++
.. code::

    {
        "type": "variable_set",
        "option": {
            "key": "<string>",
            "value": "<string>"
        }
    }

* key: Variable name.
* value: Variable value.

Example
+++++++
.. code::

    {
        "type": "variable_set",
        "option": {
            "key": "Provider name",
            "value": "voipbin"
        }
    }

.. _flow-struct-action-webhook_send:

Webhook send
------------
Send a webhook.

Parameters
++++++++++
.. code::

    {
        "type": "webhook_send",
        "option": {
            "sync": boolean,
            "uri": "<string>",
            "method": "<string>",
            "data_type": "<string>",
            "data": "<string>"
        }
    }

* sync: If this set to true, waits until this action done.
* uri: Destination uri.
* method: HTTP method. Supported values: POST, GET, PUT, DELETE.
* data_type: Content type of the data. For example: application/json.
* data: Data string. Variable can be used.

Example
+++++++
.. code::

    {
        "type": "webhook_send",
        "option": {
            "sync": true,
            "uri": "https://test.com",
            "method": "POST",
            "data_type": "application/json",
            "data": "{\"destination_number\": \"${voipbin.call.destination.target}\", \"source_number\": \"${voipbin.call.source.target}\"}"
        }
    }


