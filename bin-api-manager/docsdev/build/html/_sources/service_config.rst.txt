.. _service-config-main:

**************
Service Config
**************
A service config is a stored setting of a platform service, such as the credentials that the email service uses to talk to its providers or the LLM keys of the AI service. The Service Config API lets a platform administrator see where the value of each key comes from, store or delete a secret, and read who changed it. Only secret keys can be changed through this API for now.

**API Reference:** `Service config endpoints <https://api.voipbin.net/redoc/#tag/ServiceConfig>`_

.. toctree::
   :maxdepth: 2

   service_config_overview
   service_config_struct_service_config
