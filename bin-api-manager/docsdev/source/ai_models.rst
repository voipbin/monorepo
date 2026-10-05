.. _ai-models:

AI Models
=========

The AI Models endpoint returns the catalog of LLM models that can be used as the ``engine_model`` of an AI configuration. Use it to build a model picker or to validate a model ``id`` before calling ``POST /ais`` or ``PUT /ais/{id}``.

.. note:: **AI Implementation Hint**

   Always read the model ``id`` from ``GET /ai_models`` response rather than hard-coding it. The catalog can change over time. An ``id`` that is not in the list (and does not use the ``openai.``, ``gemini.``, or ``grok.`` prefix) is rejected with ``HTTP 400 INVALID_ENGINE_MODEL`` when you create or update an AI.

.. _ai-models-get:

Get the model list
------------------

``GET https://api.voipbin.net/v1.0/ai_models``

Returns every available model in a single response. Pagination is not used: ``page_size`` and ``page_token`` are ignored and ``next_page_token`` is always an empty string. Logged-in users and accesskeys can call this endpoint. Direct (public token) access is not supported.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/ai_models?token=<YOUR_AUTH_TOKEN>'

Response:

.. code::

    {
        "result": [
            {
                "id": "gemini.gemini-2.5-flash",
                "label": "Gemini 2.5 Flash",
                "vendor": "Google",
                "recommended": true,
                "tags": ["low-cost"],
                "description": "Fast, balanced model for most voice conversations.",
                "platform_managed": false
            },
            {
                "id": "anthropic.claude-sonnet-4.5",
                "label": "Claude Sonnet 4.5",
                "vendor": "Anthropic",
                "recommended": false,
                "tags": [],
                "description": "Balanced Claude model with strong instruction following.",
                "platform_managed": true
            }
        ],
        "next_page_token": ""
    }

.. _ai-models-struct:

Model
-----

* ``id`` (String): The model identifier. Use it as ``engine_model`` in ``POST /ais`` and ``PUT /ais/{id}``. Format: ``<provider>.<model>``.
* ``label`` (String): Human-readable model name for display.
* ``vendor`` (String): The model vendor (e.g., ``Google``, ``OpenAI``, ``Anthropic``). Use it to group models in a picker.
* ``recommended`` (Boolean): ``true`` when the platform recommends this model as a starting point.
* ``tags`` (Array of String): Model tags. Currently ``low-cost`` is the only tag.
* ``description`` (String): A short description of the model.
* ``platform_managed`` (Boolean): ``true`` when the platform supplies the credentials for this model. The ``engine_key`` of the AI is not needed: send an empty string and any value you send is ignored. ``false`` means you must provide your own provider key in ``engine_key``.

See :ref:`Engine Model <ai-struct-ai-engine_model>` for the provider prefixes and validation rules.
