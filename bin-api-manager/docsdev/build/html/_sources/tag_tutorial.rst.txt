.. _tag-tutorial:

Tutorial
========

Create a new tag
----------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/tags?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "name": "test tag",
            "detail": "test tag example"
        }'
