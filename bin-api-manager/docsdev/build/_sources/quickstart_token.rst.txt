.. _quickstart_token:

Token
=====
In this Quickstart, you'll learn how to sign up and generate to your token.

Sign up to VoIPBin
-------------------
Send the sign up request to `pchero21@gmail.com`

Generate token
---------------
To generate jwt token, you need to send the request with your username and password like the below.

.. code::

    $ curl --request POST 'https://api.voipbin.net/auth/login' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "username": "your-voipbin-username",
            "password": "your-voipbin-password"
        }'

    {
        "username": "voipbin",
        "token": "eyJhbsdiOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VubG1ieXVqamowbWcueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIzLTAyLTIxIDA4OjAxOjEyLjI2MDM4OFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAsMFwifSIsamV4cCI6MTY4MDcxNTQ5M30.mEYsKYdRnoVGVOl7Zk21YBbxMvD-0lD1dlQVnav4kRw"
    }

The token is valid for 7 days.
