.. _flow-struct:

Struct
======

.. _flow-struct-flow:

Flow
----

.. code::

    {
        "id": "<string>",
        "type": "flow",
        "name": "test conference_join",
        "detail": "test flow for conference_join",
        "actions": [
            {
                "id": "e2cafb4a-2f68-46e6-99bd-9cb0d527eca1",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "answer"
            },
            {
                "id": "6596bbc2-6079-4665-8ded-3d8d6fb9fea7",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "conference_join",
                "option": {
                    "conference_id": "99accfb7-c0dd-4a54-997d-dd18af7bc280"
                }
            }
        ],
        "tm_create": "2022-02-03 05:37:48.545532",
        "tm_update": "2022-02-03 06:10:23.604222",
        "tm_delete": "9999-01-01 00:00:00.000000"
    },


action

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

.. _flow-struct-action-type:

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

