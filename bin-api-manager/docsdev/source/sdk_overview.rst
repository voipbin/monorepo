.. _sdk-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low -- The SDK wraps the VoIPBIN REST API into language-specific libraries. No separate billing or API calls are specific to the SDK itself.
   * **Cost:** Free. Using the SDK incurs no additional charges beyond the underlying API operations.
   * **Async:** N/A. The SDK is a client library; async behavior depends on the underlying API endpoints called.

The VoIPBIN SDK (Software Development Kit) provides a set of tools and libraries that developers can use to create communication applications using the VoIPBIN platform. The SDK includes client libraries that wrap the VoIPBIN REST API, making it easier to integrate VoIPBIN's capabilities into your applications without dealing with low-level HTTP requests.

SDKs simplify the development process by providing:

* Type-safe API methods with auto-completion support
* Built-in error handling and retry logic
* Automatic request/response serialization
* Authentication management
* Comprehensive documentation and code examples

Accesskey
---------
Most SDK methods require an accesskey (String) to authenticate the user and authorize access to the VoIPBIN platform.
The accesskey is a long-lived API token obtained from ``GET https://api.voipbin.net/v1.0/accesskeys`` or created via ``POST https://api.voipbin.net/v1.0/accesskeys``.
You can also generate an accesskey through the `admin console <https://admin.voipbin.net>`_.

See detail about the accesskey :ref:`here <accesskey-overview>`.

.. note:: **AI Implementation Hint**

   When initializing the SDK, pass the accesskey string directly. Do not wrap it in a Bearer header — the SDK handles authentication internally. Use ``GET https://api.voipbin.net/v1.0/accesskeys`` to list existing keys, or create a new one via ``POST https://api.voipbin.net/v1.0/accesskeys`` with a custom expiration.

Support Languages
-----------------
Currently, the SDK supports the following programming languages:

- Go: https://github.com/voipbin/voipbin-go
