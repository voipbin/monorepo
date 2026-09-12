.. _flow-tutorial-scenario:

Tutorial scenario
=================

Prerequisites
+++++++++++++

Before running these scenarios, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A source phone number in E.164 format (e.g., ``+15559876543``). This must be a number you own. Obtain your numbers via ``GET /numbers``.
* A destination phone number in E.164 format (e.g., ``+15551112222``).
* (Optional) An existing flow to attach to a number. Create one via ``POST /flows``.

.. note:: **AI Implementation Hint**

   These scenarios use ``POST /calls`` with inline ``actions`` arrays. You can also pre-register a flow via ``POST /flows`` and then reference it by ``flow_id`` when creating a call. The inline approach is simpler for one-off calls; the flow approach is better when reusing the same logic across multiple calls or numbers. All phone numbers must be in E.164 format (e.g., ``+15559876543``).

.. _flow-tutorial-scenario-simple_voicemail:

Simple voicemail
----------------

Making an outgoing call for forwarding. If the call is not answered, leave a voicemail.

.. note:: **AI Implementation Hint**

   This scenario uses ``condition_call_status`` (deprecated). For new implementations, use ``condition_variable`` with ``variable: "voipbin.call.status"`` and ``value_string: "progressing"`` instead. The ``next_id`` field set to ``00000000-0000-0000-0000-000000000000`` means "continue to the next action in the array" -- it is the same as omitting ``next_id``.

.. code::

                  Start
                    |
                    |
                Connect(Making an outgoing call for forwarding)
                    |
                    |
                Condition check(check the call's status is Answered)
                    |
                    |
       ------------------------------
       |                            |
   condition false               condition true
       |                            |
       |                          Hangup
     Talk(...)
       |
       |
     Beep
       |
       |
    Recording start
       |
       |
     Sleep(10 sec)
       |
       |
     Hangup

.. code::

    {
        "actions": [
            {
                "type": "connect",
                "option": {
                    "source": {
                        "type": "tel",
                        "target": "+15559876543"
                    },
                    "destinations": [
                        {
                            "type": "tel",
                            "target": "+15551112222"
                        }
                    ]
                }
            },
            {
                "id": "3746e628-8cc1-4ff4-82fe-194b16b9a10e",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "condition_call_status",
                "option": {
                    "status": "progressing",
                    "false_target_id": "cfe0e8ea-991c-11ec-b849-d7fc54168fd5"
                }
            },
            {
                "id": "58f859e9-92d8-4b46-8073-722b9c881ae0",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "hangup"
            },
            {
                "id": "cfe0e8ea-991c-11ec-b849-d7fc54168fd5",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "talk",
                "option": {
                    "text": "Thank you for your calling. We are busy now. Please leave a message after tone.",
                    "language": "en-US"
                }
            },
            {
                "id": "0a9b6f38-ddcd-448b-80a1-ae47ac0e08aa",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "beep"
            },
            {
                "id": "ad969315-6ac4-4339-b300-566eb6352fea",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "recording_start",
                "option": {
                    "beep_start": true
                }
            },
            {
                "id": "8abf3f9d-414c-4a15-aa94-02a799409f48",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "sleep",
                "option": {
                    "duration": 10000
                }
            },
            {
                "id": "e4fc5d9e-9fa8-4b3e-ae77-b55b04c1f2d3",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "hangup"
            }
        ]
    }

.. _flow-tutorial-scenario-simple_branch:

Simple branch
---------------------------------

It will get the digits from the call and will execute the branch.

.. code::

                  Start
                    |
                    |
    ------------>  Talk("Press 1 for show must go on. Press 2 for bohemian rhapsody. Press 3 for another one bites the dust")
    |               |
    |               |
    |              Digit(DTMF) receive
    |               |
    |               |
    |       -----------------------------------------------
    |       |           |                |                |
    |     default      "1"              "2"              "3"
    |       |           |                |                |
    |       |           |                |                |
    |       |          Talk(...)        Talk(...)        Talk(...)
    |       |           |                |                |
    |       |           |                |                |
    |       |          Hangup          Hangup           Hangup
    |       |
    |       |
    |      Talk(...)
    |       |
    ----goto(loop 3 times)
            |
            |
           Talk(...)
            |
            |
           Hangup

.. code::

    {
        "actions": [
            {
                "id": "b8781e56-c524-11ec-889f-d37b0dbb7eb8",
                "type": "talk",
                "option": {
                    "text": "Hello. This is branch test. Press 1 for show must go on. Press 2 for bohemian rhapsody. Press 3 for another one bites the dust",
                    "language": "en-US"
                }
            },
            {
                "type": "digits_receive",
                "option": {
                    "duration": 5000,
                    "length": 1
                }
            },
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
            },
            {
                "id": "c3eb8e62-c524-11ec-94c5-abafec8af561",
                "type": "talk",
                "option": {
                    "text": "Empty spaces, what are we living for? Abandoned places, I guess we know the score, on and on. Does anybody know what we are looking for? Another hero, another mindless crime. Behind the curtain, in the pantomime",
                    "language": "en-US"
                }
            },
            {
                "type": "hangup"
            },
            {
                "id": "dc87123e-c524-11ec-89c6-5fb18da14034",
                "type": "talk",
                "option": {
                    "text": "Mama, Just killed a man. Put a gun against his head, pulled my trigger. Now he's dead. Mama, life had just begun, But now I've gone and thrown it all away.",
                    "language": "en-US"
                }
            },
            {
                "type": "hangup"
            },
            {
                "id": "e70fb030-c524-11ec-b657-ebec72f097ef",
                "type": "talk",
                "option": {
                    "text": "Steve walks warily down the street. With his brim pulled way down low. Ain't no sound but the sound of his feet. Machine guns ready to go. Are you ready hey are you ready for this?",
                    "language": "en-US"
                }
            },
            {
                "type": "hangup"
            },
            {
                "id": "ed9705ca-c524-11ec-a3fb-8feb7731ad45",
                "type": "talk",
                "option": {
                    "text": "You didn't choose the correct number. Default selected.",
                    "language": "en-US"
                }
            },
            {
                "type": "goto",
                "option": {
                    "target_id": "b8781e56-c524-11ec-889f-d37b0dbb7eb8",
                    "loop_count": 2
                }
            },
            {
                "type": "talk",
                "option": {
                    "text": "Loop over. Hangup the call. Thank you, good bye.",
                    "language": "en-US"
                }
            },
            {
                "type": "hangup"
            }
        ]
    }

.. _flow-tutorial-scenario-simple_message_send:

Simple message send
---------------------------------

Send the message to the multiple destinations.

.. code::

                  Start
                    |
                    |
                  Message send
                    |
                    |
                   End

.. code::

    {
        "actions": [
            {
                "type": "message_send",
                "option": {
                    "source": {
                        "type": "tel",
                        "target": "+821100000001"
                    },
                    "destinations": [
                        {
                            "type": "tel",
                            "target": "+821100000002"
                        },
                        {
                            "type": "tel",
                            "target": "+821100000003"
                        },
                        {
                            "type": "tel",
                            "target": "+821100000004"
                        }
                    ],
                    "text": "hello, this is test message."
                }
            }
        ]
    }

.. _flow-tutorial-scenario-simple_message_send_and_make_a_new_outbound_call:

Simple message send and make a new outbound call
------------------------------------------------

Send the message to the destination and start a new outbound call with talk action.

.. code::

                  Start
                    |
                    |
                  Message send
                    |
                    |
                  Call ------------------- Start
                    |                        |
                    |                        |
                   End                      Talk
                                             |
                                             |
                                            End

.. code::

    {
        "actions": [
            {
                "type": "message_send",
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
                    "text": "hello, this is test message."
                }
            },
            {
                "type": "call",
                "option": {
                    "source": {
                        "type": "tel",
                        "target": "+821100000001"
                    },
                    "destinations": [
                        {
                            "type": "tel",
                            "target": "+821100000003"
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
                    ]
                }
            }
        ]
    }


