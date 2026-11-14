.. _service-config-main:

**************
Service Config
**************
A service config is a stored setting of a platform service, such as the credentials that the email service uses to talk to its providers, the LLM keys of the AI service or the SMS and MMS provider tokens of the message service, the provider keys of the text to speech service, the AWS credential of the transcribe service, the Paddle keys of the billing service or the platform keys of the voice AI vendors of the pipecat service. The Service Config API lets a platform administrator see where the value of each key comes from, store or delete a secret, and read who changed it. Only secret keys can be changed through this API for now. The services read their platform secrets from environment variables, so a value stored through this API is not used now (see the note in the overview).

**API Reference:** `Service config endpoints <https://api.voipbin.net/redoc/#tag/ServiceConfig>`_

.. toctree::
   :maxdepth: 2

   service_config_overview
   service_config_struct_service_config
