.. _call-tutorial: call-tutorial

Tutorial
========

Simple outbound call with TTS
-----------------------------

Making an outbound call with TTS(Text-to-Speech) action.
When the destination answer the call, it will speak the given text message.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDcyNjM5MjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.py7AwXIO0ZNBWSS1PN-05L9oYEREjGgbkkE6CcVyuzw' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+821028286521"
            },
            "destination": {
                "type": "tel",
                "target": "+821021656521"
            },
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "hello. welcome to voipbin. This is test message. This audio file is generated dynamically by the tts module. Please enjoy the voipbin service. Thank you. Bye",
                        "gender": "female",
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

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+821028286521"
        },
        "destination": {
            "type": "tel",
            "target": "+821021656521"
        },
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

    {
        "id": "a023bfa8-1091-4e94-8eaa-7f01fbecc71a",
        "user_id": 1,
        "flow_id": "f089791a-ac78-4ea0-be88-8a8e131f9fc5",
        "conf_id": "00000000-0000-0000-0000-000000000000",
        "type": "flow",
        "master_call_id": "00000000-0000-0000-0000-000000000000",
        "chained_call_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "source": {
            "type": "tel",
            "target": "+821028286521",
            "name": ""
        },
        "destination": {
            "type": "tel",
            "target": "+821021656521",
            "name": ""
        },
        "status": "dialing",
        "direction": "outgoing",
        "hangup_by": "",
        "hangup_reason": "",
        "tm_create": "2021-02-04 04:44:20.904662",
        "tm_update": "",
        "tm_progressing": "",
        "tm_ringing": "",
        "tm_hangup": ""
    }


Simple outbound call with TTS and connect
-------------------------------------------

Making an outbound call with TTS(Text-to-Speech) and connect to other destination.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/calls?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --header 'Content-Type: application/json' \
    --header 'Cookie: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+821021656521"
        },
        "destination": {
            "type": "tel",
            "target": "+821021656521"
        },
        "actions": [
            {
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. This audio file is generated dynamically by the tts module. Please enjoy the voipbin service.",
                    "gender": "female",
                    "language": "en-US"
                }
            },
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
                            "target": "+821043126521"
                        }
                    ]
                }
            }
        ]
    }'

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
            "target": "+821021656521",
            "name": ""
        },
        "destination": {
            "type": "tel",
            "target": "+821021656521",
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


Get call list
-------------

Getting a list of calls.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/calls?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM'

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

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/calls/f457951b-9918-44af-a834-2216b1cc31bc?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM'

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
            "target": "+821028286521",
            "name": ""
        },
        "destination": {
            "type": "tel",
            "target": "+821021656521",
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
