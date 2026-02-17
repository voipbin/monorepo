.. _call-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before creating a call, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A source phone number in E.164 format (e.g., ``+15551234567``). This must be a number you own. Obtain your numbers via ``GET /numbers``.
* A destination phone number in E.164 format (e.g., ``+15559876543``) or a SIP endpoint address.
* (Optional) A flow ID (UUID). Create one via ``POST /flows`` or obtain from ``GET /flows``.

.. note:: **AI Implementation Hint**

   All phone numbers must be in E.164 format: start with ``+``, followed by country code and number, no dashes or spaces. For example, ``+15551234567`` (US) or ``+821012345678`` (Korea). If the user provides a local format like ``010-1234-5678``, normalize it to ``+821012345678`` before calling the API.

Simple outbound call with TTS
-----------------------------

Making an outbound call with TTS (Text-to-Speech) action.
When the destination answers the call, it will speak the given text message.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "hello. welcome to voipbin. This is test message. This audio file is generated dynamically by the tts module. Please enjoy the voipbin service. Thank you. Bye",
                        "language": "en-US"
                    }
                }
            ]
        }'


Simple outbound call with media file play
-----------------------------------------

Making an outbound call with media file play action.
When the destination answer the call, it will play the given media file.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+15551234567"
        },
        "destinations": [
            {
                "type": "tel",
                "target": "+15559876543"
            }
        ],
        "actions": [
            {
                "type": "play",
                "option": {
                    "stream_urls": [
                        "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
                    ]
                }
            }
        ]
    }'

    [
        {
            "id": "a023bfa8-1091-4e94-8eaa-7f01fbecc71a",   // Save this as call_id (UUID)
            "user_id": 1,
            "flow_id": "f089791a-ac78-4ea0-be88-8a8e131f9fc5",   // Auto-generated flow (UUID)
            "conf_id": "00000000-0000-0000-0000-000000000000",
            "type": "flow",   // enum: flow, conference, sip-service
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+15551234567",
                "name": ""
            },
            "destination": {
                "type": "tel",
                "target": "+15559876543",
                "name": ""
            },
            "status": "dialing",   // enum: dialing, ringing, progressing, terminating, canceling, hangup
            "direction": "outgoing",   // enum: incoming, outgoing
            "hangup_by": "",
            "hangup_reason": "",
            "tm_create": "2021-02-04 04:44:20.904662",   // ISO 8601 timestamp
            "tm_update": "",
            "tm_progressing": "",
            "tm_ringing": "",
            "tm_hangup": ""
        }
    ]


Simple outbound call with TTS and connect
-------------------------------------------

Making an outbound call with TTS(Text-to-Speech) and connect to other destination.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+15559876543"
        },
        "destinations": [
            {
                "type": "tel",
                "target": "+15559876543"
            }
        ],
        "actions": [
            {
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. This audio file is generated dynamically by the tts module. Please enjoy the voipbin service.",
                    "language": "en-US"
                }
            },
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
                            "target": "+15551111111"
                        }
                    ]
                }
            }
        ]
    }'

    [
        {
            "id": "9f6265bc-6b59-4e80-a906-2679aca11455",
            "user_id": 1,
            "flow_id": "d665fbc0-6dd8-44bc-99ea-2ae54bc59428",
            "conf_id": "00000000-0000-0000-0000-000000000000",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+15559876543",
                "name": ""
            },
            "destination": {
                "type": "tel",
                "target": "+15559876543",
                "name": ""
            },
            "status": "dialing",
            "direction": "outgoing",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_create": "2021-02-06 09:52:49.941865",
            "tm_update": "",
            "tm_progressing": "",
            "tm_ringing": "",
            "tm_hangup": ""
        }
    ]

Simple outbound call with talk and digits_send
----------------------------------------------
Making an outbound call. After answer the call, it will play the TTS and then send the DTMFs.

.. code::

    {
        "source": {
            "type": "tel",
            "target": "+15551234567"
        },
        "destinations": [
            {
                "type": "tel",
                "target": "+15559876543"
            }
        ],
        "actions": [
            {
                "type": "talk",
                "option": {
                    "text": "This is dtmf send test call. Please wait.",
                    "language": "en-US"
                }
            },
            {
                "type": "dtmf_send",
                "option": {
                    "dtmfs": "1234567890",
                    "duration": 500,
                    "interval": 500
                }
            },
            {
                "type": "talk",
                "option": {
                    "text": "Thank you. DTMF send test has done.",
                    "language": "en-US"
                }
            }
        ]
    }

    [
        {
            "id": "d7520a58-0b07-4dd7-ab72-a4e2d1979ec0",
            "user_id": 1,
            "flow_id": "0f4bd9bc-9df5-4a5b-9465-2189822a3019",
            "conf_id": "00000000-0000-0000-0000-000000000000",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+15551234567",
                "name": ""
            },
            "destination": {
                "type": "tel",
                "target": "+15559876543",
                "name": ""
            },
            "status": "dialing",
            "direction": "outgoing",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_create": "2021-02-08 03:59:33.281711",
            "tm_update": "",
            "tm_progressing": "",
            "tm_ringing": "",
            "tm_hangup": ""
        }
    ]



Simple outbound call with Branch
---------------------------------

Making an outbound call with branch. It will get the digits from the call and will execute the branch.

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
    ----goto(loop 2 times)
            |
            |
           Talk(...)
            |
            |
           Hangup

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+15551234567"
        },
        "destinations": [
            {
                "type": "tel",
                "target": "+15559876543"
            }
        ],
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
                    "text": "Mama, Just killed a man. Put a gun against his head, pulled my trigger. Now he'\''s dead. Mama, life had just begun, But now I'\''ve gone and thrown it all away.",
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
                    "text": "Steve walks warily down the street. With his brim pulled way down low. Ain'\''t no sound but the sound of his feet. Machine guns ready to go. Are you ready hey are you ready for this?",
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
                    "text": "You didn'\''t choose the correct number. Default selected.",
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
    }'

    [
        {
            "id": "77517719-ffb9-4583-ba44-737ba991d685",
            "flow_id": "c0827e56-41ef-4fa1-9da0-a8a36fbb76c4",
            "confbridge_id": "00000000-0000-0000-0000-000000000000",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+15551234567",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+15559876543",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "status": "dialing",
            "action": {
                "id": "00000000-0000-0000-0000-000000000001",
                "type": ""
            },
            "direction": "outgoing",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_create": "2022-02-24 02:08:14.469405",
            "tm_update": "9999-01-01 00:00:00.000000",
            "tm_progressing": "9999-01-01 00:00:00.000000",
            "tm_ringing": "9999-01-01 00:00:00.000000",
            "tm_hangup": "9999-01-01 00:00:00.000000"
        }
    ]

Get call list
-------------

Getting a list of calls.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "9a7857ca-73ba-4000-8101-c47d3b48f9d1",
                "user_id": 1,
                "flow_id": "00000000-0000-0000-0000-000000000000",
                "conf_id": "00000000-0000-0000-0000-000000000000",
                "type": "sip-service",
                "master_call_id": "00000000-0000-0000-0000-000000000000",
                "chained_call_ids": [],
                "recording_id": "00000000-0000-0000-0000-000000000000",
                "recording_ids": [],
                "source": {
                    "type": "tel",
                    "target": "109",
                    "name": "109"
                },
                "destination": {
                    "type": "tel",
                    "target": "972595897084",
                    "name": ""
                },
                "status": "hangup",
                "direction": "incoming",
                "hangup_by": "remote",
                "hangup_reason": "normal",
                "tm_create": "2021-02-06 09:47:10.018000",
                "tm_update": "2021-02-06 09:48:14.630000",
                "tm_progressing": "2021-02-06 09:47:10.626000",
                "tm_ringing": "",
                "tm_hangup": "2021-02-06 09:48:14.630000"
            },
            ...
        ],
        "next_page_token": "2021-02-06 08:54:38.361000"
    }


Get specific call
-----------------

Getting a given call uuid's call info.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/calls/f457951b-9918-44af-a834-2216b1cc31bc?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "f457951b-9918-44af-a834-2216b1cc31bc",
        "user_id": 1,
        "flow_id": "246aeabe-fab5-4a1b-8e98-852b50e89dd7",
        "conf_id": "00000000-0000-0000-0000-000000000000",
        "type": "flow",
        "master_call_id": "00000000-0000-0000-0000-000000000000",
        "chained_call_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [
            "142e8ef8-392c-4514-abf0-8656da5d2fdf"
        ],
        "source": {
            "type": "tel",
            "target": "+15551234567",
            "name": ""
        },
        "destination": {
            "type": "tel",
            "target": "+15559876543",
            "name": ""
        },
        "status": "hangup",
        "direction": "outgoing",
        "hangup_by": "remote",
        "hangup_reason": "normal",
        "tm_create": "2021-01-29 03:17:54.349101",
        "tm_update": "2021-01-29 03:18:22.131000",
        "tm_progressing": "2021-01-29 03:18:07.810000",
        "tm_ringing": "2021-01-29 03:17:55.392000",
        "tm_hangup": "2021-01-29 03:18:22.131000"
    }

Make a groupcall
-----------------

Make a groupcall to the multiple destinations.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/groupcalls?token=eyJhbGcslkj' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15552222222"
            },
            "destinations": [
                {
                    "type": "endpoint",
                    "target": "test11@test"
                },
                {
                    "type": "endpoint",
                    "target": "test12@test"
                }
            ],
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "hello. welcome to voipbin. This is test message. This audio file is generated dynamically by the tts module. Please enjoy the voipbin service. Thank you. Bye",
                        "language": "en-US"
                    }
                }
            ]
        }'

    {
        "id": "d8596b14-4d8e-4a86-afde-642b46d59ac7",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "source": {
            "type": "tel",
            "target": "+15551234567",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "destinations": [
            {
                "type": "endpoint",
                "target": "test11@test",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            {
                "type": "endpoint",
                "target": "test12@test",
                "target_name": "",
                "name": "",
                "detail": ""
            }
        ],
        "ring_method": "",
        "answer_method": "",
        "answer_call_id": "00000000-0000-0000-0000-000000000000",
        "call_ids": [
            "3c77eb43-2098-4890-bb6c-5af0707ba4a6",
            "2bcaff64-e05d-11ed-84a6-133172844032"
        ],
        "tm_create": "2023-04-21 15:33:28.569053",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** The ``source.target`` or ``destinations[].target`` phone number is not in E.164 format (contains dashes, spaces, or missing ``+``).
    * **Fix:** Normalize all phone numbers to E.164 format: ``+`` followed by country code and number, no spaces or dashes (e.g., ``+15551234567``).

* **400 Bad Request (source number):**
    * **Cause:** The ``source.target`` phone number is not owned by your VoIPBIN account.
    * **Fix:** Use a number from ``GET /numbers``. Only numbers you own can be used as the source.

* **402 Payment Required:**
    * **Cause:** Insufficient account balance to make a call.
    * **Fix:** Check balance via ``GET /billing-accounts``. Top up before retrying.

* **404 Not Found (flow_id):**
    * **Cause:** The ``flow_id`` does not exist or belongs to a different customer.
    * **Fix:** Verify the flow ID was obtained from ``GET /flows`` or ``POST /flows``.

* **Call immediately hangs up:**
    * **Cause:** No ``answer`` action at the beginning of the flow, or the destination is unreachable.
    * **Fix:** Ensure your actions list starts with ``{"type": "answer"}`` for outbound calls. Check the call's ``hangup_reason`` field via ``GET /calls/{id}`` for details.
