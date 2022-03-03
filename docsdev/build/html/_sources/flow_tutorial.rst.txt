.. _flow-tutorial: flow-tutorial

Tutorial
========

Get list of flows
-----------------

Gets the list of registered flows.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/flows?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM'

    {
        "result": [
            {
                "id": "decc2634-0b2a-11eb-b38d-87a8f1051188",
                "name": "default flow",
                "detail": "default flow for voipbin incoming calls",
                "actions": [
                    {
                        "id": "b34aa8a4-0b30-11eb-8016-1f5bc75b1c04",
                        "type": "play",
                        "option": {
                            "stream_url": [
                                "https://github.com/pchero/asterisk-medias/raw/master/voipbin/welcome.wav"
                            ]
                        }
                    },
                    {
                        "id": "57a3dcd2-0b2b-11eb-94a6-a7129b64693c",
                        "type": "play",
                        "option": {
                            "stream_url": [
                                "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
                            ]
                        }
                    }
                ],
                "tm_create": "2020-10-11 01:00:00.000001",
                "tm_update": "",
                "tm_delete": ""
            },
            {
                "id": "af9dae94-ef07-11ea-a101-8f52e568f39b",
                "name": "test flow",
                "detail": "manual flow test",
                "actions": [
                    {
                        "id": "00000000-0000-0000-0000-000000000000",
                        "type": "echo"
                    }
                ],
                "tm_create": "2020-09-04 23:53:14.496918",
                "tm_update": "",
                "tm_delete": ""
            }
        ],
        "next_page_token": "2020-09-04 23:53:14.496918"
    }

Simple voicemail scenario
-------------------------

Making a outgoing call for forwarding. If call not answered, leave a voicemail.

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
     Sleep(30 sec)
       |
       |
     Hangup

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/flows?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --header 'Content-Type: application/json' \
    --header 'Cookie: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --data-raw '{
        "name": "simple voicemail scenario",
        "detail": "simple flow for voicemail scenario",
        "actions": [
            {
                "type": "connect",
                "option": {
                    "source": {
                        "type": "tel",
                        "target": "+821021656521"
                    },
                    "destinations": [
                        {
                            "type": "tel",
                            "target": "+821021546521"
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
                    "gender": "female",
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
    }'


Get detail of specified flow
----------------------------

Gets the detail of registered flows.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/flows/decc2634-0b2a-11eb-b38d-87a8f1051188?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM'

    {
        "id": "decc2634-0b2a-11eb-b38d-87a8f1051188",
        "name": "default flow",
        "detail": "default flow for voipbin incoming calls",
        "actions": [
            {
                "id": "b34aa8a4-0b30-11eb-8016-1f5bc75b1c04",
                "type": "play",
                "option": {
                    "stream_url": [
                        "https://github.com/pchero/asterisk-medias/raw/master/voipbin/welcome.wav"
                    ]
                }
            },
            {
                "id": "57a3dcd2-0b2b-11eb-94a6-a7129b64693c",
                "type": "play",
                "option": {
                    "stream_url": [
                        "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
                    ]
                }
            }
        ],
        "tm_create": "2020-10-11 01:00:00.000001",
        "tm_update": "",
        "tm_delete": ""
    }

Create a flow
-------------

Create a new flow for incoming call requests.
When the call is comming, this flow will answer the call first, then will speech the welcome text.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/flows?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test flow",
        "detail": "test voipbin flow example",
        "actions": [
            {
                "type": "answer"
            },
            {
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin'\''s service. thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            }
        ]
    }'

    {
        "id": "24013a0e-d15b-4b5e-9a96-04221a8c6a15",
        "name": "test flow",
        "detail": "test voipbin flow example",
        "actions": [
            {
                "id": "9461bda1-54fd-4e27-ab04-4186c6f72830",
                "type": "answer"
            },
            {
                "id": "69af787e-f5fa-4a1b-9d12-f0b43b86dae6",
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin's service. thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            }
        ],
        "tm_create": "2021-02-04 06:47:01.139361",
        "tm_update": "",
        "tm_delete": ""
    }

Update the flow
---------------

Update the existed flow with given info.
The doesn't affect to the existed call. The flow changes will be affected only a new calls.

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/flows/decc2634-0b2a-11eb-b38d-87a8f1051188?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test flow update",
        "detail": "test voipbin flow example update",
        "actions": [
            {
                "type": "answer"
            },
            {
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin'\''s service. thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            },
            {
                "type": "play",
                "option": {
                    "stream_url": [
                        "https://github.com/pchero/asterisk-medias/raw/master/voipbin/welcome.wav"
                    ]
                }
            },
            {
                "type": "play",
                "option": {
                    "stream_url": [
                        "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
                    ]
                }
            }
        ]
    }'

    {
        "id": "decc2634-0b2a-11eb-b38d-87a8f1051188",
        "name": "test flow update",
        "detail": "test voipbin flow example update",
        "actions": [
            {
                "id": "be682498-e57e-41e9-b210-a578f9c044c5",
                "type": "answer"
            },
            {
                "id": "6669bfdd-a7b0-45e6-9a8d-db6bb898159f",
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin's service. thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            },
            {
                "id": "099b60c1-7b95-4d69-8cac-df11a992ee11",
                "type": "play",
                "option": {
                    "stream_url": [
                        "https://github.com/pchero/asterisk-medias/raw/master/voipbin/welcome.wav"
                    ]
                }
            },
            {
                "id": "89fa5091-a192-4758-8a29-316776ead8fe",
                "type": "play",
                "option": {
                    "stream_url": [
                        "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
                    ]
                }
            }
        ],
        "tm_create": "2020-10-11 01:00:00.000001",
        "tm_update": "2021-02-05 13:08:56.113036",
        "tm_delete": ""
    }

Delete the flow
---------------

Delete the existed flow of given flow id.
The doesn't affect to the existed call.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/flows/af9dae94-ef07-11ea-a101-8f52e568f39b?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \


