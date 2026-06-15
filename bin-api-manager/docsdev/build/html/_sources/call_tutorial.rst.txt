.. _call-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before creating a call, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A source phone number in E.164 format (e.g., ``+15551234567``). This must be a **normal** (non-virtual) number you own with **active** status. Obtain your numbers via ``GET https://api.voipbin.net/v1.0/numbers`` and use one where ``type`` is ``normal`` and ``status`` is ``active``. Virtual numbers cannot be used as the source for outgoing PSTN calls.
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


Simple outbound call with an existing flow
-------------------------------------------

If you have already created a flow, you can reference it by ``flow_id`` instead of defining actions inline:

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "<your-source-number>"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "<your-destination-number>"
                }
            ],
            "flow_id": "<your-flow-id>"
        }'

.. note:: **AI Implementation Hint**

   The ``flow_id`` (UUID) must reference an existing flow. Obtain one from the ``id`` field of ``GET /flows`` or create one via ``POST /flows``. If the flow does not exist, the API returns ``404 Not Found``.

For more details on flows, see the :ref:`Flow tutorial <flow-main>`.

Passing custom variables into the call's flow
---------------------------------------------

You can seed per-call context into the flow by adding an optional ``variables`` object to the ``POST /calls`` request. Each key becomes referenceable inside the flow as ``${<key>}`` (for example, in a ``talk`` action's text or a ``webhook`` action's body).

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+155****4567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+155****6543"
                }
            ],
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "Hello ${customer_name}, this is a call about campaign ${campaign_id}.",
                        "language": "en-US"
                    }
                }
            ],
            "variables": {
                "campaign_id": "summer-2026",
                "customer_name": "Jane Doe"
            }
        }'

At runtime the destination hears "Hello Jane Doe, this is a call about campaign summer-2026."

.. note:: **Variable limits and reserved keys**

   Values must be strings. The injection accepts at most 100 keys, 64KB total (keys plus values), and 32KB per individual value. Keys beginning with ``voipbin.`` are reserved for system variables and are silently ignored. See the :ref:`Variables overview <variable-overview>` for the full reference.

Anonymous outbound call (API-initiated)
---------------------------------------

You can hide your caller ID on outgoing PSTN calls by setting ``"anonymous": "yes"`` in the ``POST /calls`` request. The destination sees "Anonymous" or "Private number" instead of your real phone number.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "anonymous": "yes",
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
                        "text": "Hello. This is an anonymous call from voipbin.",
                        "language": "en-US"
                    }
                }
            ]
        }'

The ``anonymous`` parameter accepts three values:

- ``"yes"`` — Always hide caller ID. The destination sees "Anonymous".
- ``"no"`` — Always show real caller ID.
- ``"auto"`` — (Default) Inherit from incoming call's Privacy header. If there is no incoming call, behaves like ``"no"``.

.. note:: **AI Implementation Hint**

   The ``anonymous`` parameter only affects PSTN destinations (``type: "tel"``). It has no effect on SIP or extension destinations. Some carriers or countries may reject anonymous calls — if the call fails, retry with ``"no"`` or omit the parameter.


Anonymous outbound call within a flow (connect action)
------------------------------------------------------

A common scenario is receiving an incoming call on a registered extension and then connecting it to an external PSTN number with anonymous caller ID. For example, a customer calls your VoIPBIN number, your flow answers and plays a greeting, then connects to a mobile phone — but you want the mobile phone to see "Anonymous" instead of your VoIPBIN number.

Use the ``connect`` action with ``"anonymous": "yes"`` in the flow:

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
                        "text": "Please wait while we connect your call.",
                        "language": "en-US"
                    }
                },
                {
                    "type": "connect",
                    "option": {
                        "anonymous": "yes",
                        "source": {
                            "type": "tel",
                            "target": "+15551234567"
                        },
                        "destinations": [
                            {
                                "type": "tel",
                                "target": "+15557778888"
                            }
                        ]
                    }
                }
            ]
        }'

In this example:

1. An outbound call is made to ``+15559876543``.
2. When answered, VoIPBIN plays a greeting message.
3. The ``connect`` action creates a second outbound leg to ``+15557778888`` with anonymous caller ID.
4. ``+15557778888`` sees "Anonymous" instead of ``+15551234567``.
5. Once ``+15557778888`` answers, both parties are bridged together.

The same pattern works for incoming calls. If you assign this flow to a VoIPBIN number, incoming callers hear the greeting and are connected anonymously to the PSTN destination.

You can also use ``"anonymous": "auto"`` in the ``connect`` action. In that case, if the *incoming* call already had a Privacy header (i.e., the original caller was anonymous), the outbound leg preserves that anonymity. Otherwise, the real caller ID is shown.

.. note:: **AI Implementation Hint**

   The ``anonymous`` option is available on both the ``connect`` and ``call`` flow actions. Use ``connect`` when bridging the current call to another destination (1:1 call forwarding). Use ``call`` when creating an independent outbound call from within a flow. Both support the same ``"yes"``/``"no"``/``"auto"`` values.


Anonymous outbound call within a flow (call action)
----------------------------------------------------

The ``call`` action creates an **independent** outbound call from within a flow. Unlike ``connect`` (which bridges the current call to another party), ``call`` spawns a separate call that runs its own flow or actions independently.

Use this when you want to trigger an anonymous notification call to a third party without bridging it to the current call.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/flows?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Notify third party anonymously",
            "actions": [
                {
                    "type": "answer"
                },
                {
                    "type": "talk",
                    "option": {
                        "text": "Incoming customer request detected. Triggering anonymous notification.",
                        "language": "en-US"
                    }
                },
                {
                    "type": "call",
                    "option": {
                        "anonymous": "yes",
                        "source": {
                            "type": "tel",
                            "target": "+15551234567"
                        },
                        "destinations": [
                            {
                                "type": "tel",
                                "target": "+15557778888"
                            }
                        ],
                        "actions": [
                            {
                                "type": "talk",
                                "option": {
                                    "text": "This is an automated notification. A customer request has been received.",
                                    "language": "en-US"
                                }
                            }
                        ]
                    }
                },
                {
                    "type": "talk",
                    "option": {
                        "text": "Notification sent. Thank you.",
                        "language": "en-US"
                    }
                }
            ]
        }'

In this example:

1. The flow answers an incoming call and speaks a greeting to the caller.
2. The ``call`` action spawns an **independent** outbound call to ``+15557778888`` with anonymous caller ID.
3. ``+15557778888`` sees "Anonymous" and hears the automated notification message.
4. Meanwhile, the original call continues to the next action (the "Notification sent" message) — it does **not** wait for the ``call`` action's outbound call to finish.

Key differences from the ``connect`` action:

- ``connect`` bridges two calls together (the caller hears the destination and vice versa). ``call`` creates a separate, independent call.
- With ``connect``, the flow pauses until the connected call ends. With ``call``, the flow continues immediately to the next action.
- Use ``connect`` for call forwarding scenarios. Use ``call`` for fire-and-forget notifications or spawning parallel calls.


Anonymous outbound call from a registered endpoint (SIP phone)
--------------------------------------------------------------

When a SIP phone registered with VoIPBIN dials an external PSTN number, the system automatically handles anonymous caller ID based on the phone's SIP ``Privacy`` header. There is no API parameter to set — the behavior is determined entirely by the SIP phone's own settings.

**How it works:**

::

    SIP Phone                    VoIPBIN                     PSTN Destination
       |                           |                              |
       |  INVITE +15559876543      |                              |
       |  Privacy: id              |                              |
       +-------------------------->|                              |
       |                           |  INVITE (anonymous From)     |
       |                           |  P-Asserted-Identity: ...    |
       |                           |  Privacy: id                 |
       |                           +----------------------------->|
       |                           |                              |
       |                           |  180 Ringing                 |
       |                           |<-----------------------------+
       |  180 Ringing              |                              |
       |<--------------------------+                              |
       |                           |                              |
       |  (Destination sees "Anonymous" on their phone)           |

- If the SIP phone includes ``Privacy: id`` in the outgoing INVITE, VoIPBIN forwards the call as anonymous — the PSTN destination sees "Anonymous" or "Private number".
- If the SIP phone does **not** include a Privacy header, VoIPBIN uses the real caller ID — the PSTN destination sees the actual source number.

**Setting up anonymous calls on your SIP phone:**

Most SIP phones and softphones have a "Hide Caller ID", "Anonymous Call", or "CLIR" (Calling Line Identification Restriction) setting. Enable this to send the ``Privacy: id`` header automatically. The exact steps vary by phone manufacturer:

- **Softphones** (Ooh La La, Ooh, LinPhone, etc.): Look for "Privacy" or "Anonymous" settings in the SIP account configuration.
- **Desk phones** (Yealink, Polycom, Grandstream, etc.): Typically under Account > Advanced > Anonymous Call or Send Anonymous.
- **Mobile SIP apps**: Usually in Settings > Account > Caller ID / Privacy.

.. note:: **AI Implementation Hint**

   Registered endpoint outbound calls always use ``anonymous: "auto"`` internally. The user cannot set ``"yes"`` or ``"no"`` from the SIP phone — the system inherits the phone's own Privacy header setting. If a user asks "how do I make anonymous calls from my SIP phone", instruct them to enable the "Anonymous Call" or "Hide Caller ID" setting on their SIP phone/softphone. There is no API call needed.


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
            "id": "a023bfa8-1091-4e94-8eaa-7f01fbecc71a",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "flow_id": "f089791a-ac78-4ea0-be88-8a8e131f9fc5",
            "activeflow_id": "00000000-0000-0000-0000-000000000000",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "groupcall_id": "00000000-0000-0000-0000-000000000000",
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
            "direction": "outgoing",
            "mute_direction": "",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_progressing": null,
            "tm_ringing": null,
            "tm_hangup": null,
            "tm_create": "2021-02-04T04:44:20Z",
            "tm_update": null,
            "tm_delete": null
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
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "flow_id": "d665fbc0-6dd8-44bc-99ea-2ae54bc59428",
            "activeflow_id": "00000000-0000-0000-0000-000000000000",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "groupcall_id": "00000000-0000-0000-0000-000000000000",
            "source": {
                "type": "tel",
                "target": "+15559876543",
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
            "direction": "outgoing",
            "mute_direction": "",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_progressing": null,
            "tm_ringing": null,
            "tm_hangup": null,
            "tm_create": "2021-02-06T09:52:49Z",
            "tm_update": null,
            "tm_delete": null
        }
    ]

Simple outbound call with talk and digits_send
----------------------------------------------
Making an outbound call. After answer the call, it will play the TTS and then send the DTMFs.

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
    }'

    [
        {
            "id": "d7520a58-0b07-4dd7-ab72-a4e2d1979ec0",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "flow_id": "0f4bd9bc-9df5-4a5b-9465-2189822a3019",
            "activeflow_id": "00000000-0000-0000-0000-000000000000",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "groupcall_id": "00000000-0000-0000-0000-000000000000",
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
            "direction": "outgoing",
            "mute_direction": "",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_progressing": null,
            "tm_ringing": null,
            "tm_hangup": null,
            "tm_create": "2021-02-08T03:59:33Z",
            "tm_update": null,
            "tm_delete": null
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
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "flow_id": "c0827e56-41ef-4fa1-9da0-a8a36fbb76c4",
            "activeflow_id": "00000000-0000-0000-0000-000000000000",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "groupcall_id": "00000000-0000-0000-0000-000000000000",
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
            "mute_direction": "",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_progressing": null,
            "tm_ringing": null,
            "tm_hangup": null,
            "tm_create": "2022-02-24T02:08:14Z",
            "tm_update": null,
            "tm_delete": null
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
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "flow_id": "00000000-0000-0000-0000-000000000000",
                "activeflow_id": "00000000-0000-0000-0000-000000000000",
                "type": "sip-service",
                "master_call_id": "00000000-0000-0000-0000-000000000000",
                "chained_call_ids": [],
                "recording_id": "00000000-0000-0000-0000-000000000000",
                "recording_ids": [],
                "groupcall_id": "00000000-0000-0000-0000-000000000000",
                "source": {
                    "type": "tel",
                    "target": "+15551234567",
                    "target_name": "",
                    "name": "109",
                    "detail": ""
                },
                "destination": {
                    "type": "tel",
                    "target": "+972595897084",
                    "target_name": "",
                    "name": "",
                    "detail": ""
                },
                "status": "hangup",
                "direction": "incoming",
                "mute_direction": "",
                "hangup_by": "remote",
                "hangup_reason": "normal",
                "tm_progressing": "2021-02-06T09:47:10Z",
                "tm_ringing": null,
                "tm_hangup": "2021-02-06T09:48:14Z",
                "tm_create": "2021-02-06T09:47:10Z",
                "tm_update": "2021-02-06T09:48:14Z",
                "tm_delete": null
            },
            ...
        ],
        "next_page_token": "2021-02-06T08:54:38Z"
    }


Get specific call
-----------------

Getting a given call uuid's call info.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/calls/f457951b-9918-44af-a834-2216b1cc31bc?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "f457951b-9918-44af-a834-2216b1cc31bc",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "flow_id": "246aeabe-fab5-4a1b-8e98-852b50e89dd7",
        "activeflow_id": "00000000-0000-0000-0000-000000000000",
        "type": "flow",
        "master_call_id": "00000000-0000-0000-0000-000000000000",
        "chained_call_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [
            "142e8ef8-392c-4514-abf0-8656da5d2fdf"
        ],
        "groupcall_id": "00000000-0000-0000-0000-000000000000",
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
        "status": "hangup",
        "direction": "outgoing",
        "mute_direction": "",
        "hangup_by": "remote",
        "hangup_reason": "normal",
        "tm_progressing": "2021-01-29T03:18:07Z",
        "tm_ringing": "2021-01-29T03:17:55Z",
        "tm_hangup": "2021-01-29T03:18:22Z",
        "tm_create": "2021-01-29T03:17:54Z",
        "tm_update": "2021-01-29T03:18:22Z",
        "tm_delete": null
    }

Make a groupcall
-----------------

Make a groupcall to the multiple destinations.

.. note::

   Provide either ``flow_id`` or ``actions``, but not both. If ``flow_id`` is set, that flow runs and ``actions`` is ignored. If ``flow_id`` is omitted (or empty), VoIPBIN builds a temporary flow from ``actions``. When both are provided, ``flow_id`` takes precedence and ``actions`` is ignored.

   ``ring_method`` controls how destinations are dialed: ``ring_all`` (dial all at once) or ``linear`` (dial one at a time). ``answer_method`` controls what happens once a destination answers: ``hangup_others`` hangs up the remaining calls. See :ref:`Groupcall struct <call-struct-groupcall>` for details.

   The ``+155****XXXX`` values below are masked placeholders. Replace them with real E.164 numbers (the source must be a number you own; see Prerequisites above).

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/groupcalls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+155****4567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+155****1111"
                },
                {
                    "type": "tel",
                    "target": "+155****2222"
                }
            ],
            "ring_method": "ring_all",
            "answer_method": "hangup_others",
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
        "flow_id": "00000000-0000-0000-0000-000000000000",
        "source": {
            "type": "tel",
            "target": "+155****4567",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "destinations": [
            {
                "type": "tel",
                "target": "+155****1111",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            {
                "type": "tel",
                "target": "+155****2222",
                "target_name": "",
                "name": "",
                "detail": ""
            }
        ],
        "master_call_id": "00000000-0000-0000-0000-000000000000",
        "master_groupcall_id": "00000000-0000-0000-0000-000000000000",
        "ring_method": "ring_all",
        "answer_method": "hangup_others",
        "answer_call_id": "00000000-0000-0000-0000-000000000000",
        "call_ids": [
            "3c77eb43-2098-4890-bb6c-5af0707ba4a6",
            "2bcaff64-e05d-11ed-84a6-133172844032"
        ],
        "answer_groupcall_id": "00000000-0000-0000-0000-000000000000",
        "groupcall_ids": [],
        "tm_create": "2023-04-21T15:33:28Z",
        "tm_update": null,
        "tm_delete": null
    }

Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** The ``source.target`` or ``destinations[].target`` phone number is not in E.164 format (contains dashes, spaces, or missing ``+``).
    * **Fix:** Normalize all phone numbers to E.164 format: ``+`` followed by country code and number, no spaces or dashes (e.g., ``+15551234567``).

* **400 Bad Request (source number):**
    * **Cause:** The ``source.target`` phone number is not owned by your VoIPBIN account.
    * **Fix:** Use a number from ``GET https://api.voipbin.net/v1.0/numbers``. Only **normal** (non-virtual) numbers with **active** status that you own can be used as the source for outgoing PSTN calls.

* **Caller ID shows "Anonymous":**
    * **Cause:** The source number failed validation (not E.164, not owned, virtual type, or not active) and no default outgoing source number is configured on the customer profile.
    * **Fix:** Either use a valid normal number from ``GET https://api.voipbin.net/v1.0/numbers`` as the source, or configure a default outgoing source number on your customer profile via ``PUT https://api.voipbin.net/v1.0/customer``.

* **Caller ID shows a different number than requested:**
    * **Cause:** The requested source number failed validation, but the customer has a default outgoing source number configured. The system used the default number as the caller ID instead.
    * **Fix:** If you want to use a specific source number, ensure it is a normal (non-virtual) number with active status owned by your account. Check ``GET https://api.voipbin.net/v1.0/numbers``.

* **402 Payment Required:**
    * **Cause:** Insufficient account balance to make a call.
    * **Fix:** Check balance via ``GET /billing-accounts``. Top up before retrying.

* **404 Not Found (flow_id):**
    * **Cause:** The ``flow_id`` does not exist or belongs to a different customer.
    * **Fix:** Verify the flow ID was obtained from ``GET /flows`` or ``POST /flows``.

* **Call immediately hangs up:**
    * **Cause:** No ``answer`` action at the beginning of the flow, or the destination is unreachable.
    * **Fix:** Ensure your actions list starts with ``{"type": "answer"}`` for outbound calls. Check the call's ``hangup_reason`` field via ``GET /calls/{id}`` for details.
