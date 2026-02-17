.. _sdk_overview:

Overview
========
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
The accesskey is a long-lived API token obtained from ``GET /accesskeys`` or created via ``POST /accesskeys``.
You can also generate an accesskey through the `admin console <https://admin.voipbin.net>`_.

See detail about the accesskey :ref:`here <accesskey-overview>`.

.. note:: **AI Implementation Hint**

   When initializing the SDK, pass the accesskey string directly. Do not wrap it in a Bearer header — the SDK handles authentication internally. Use ``GET /accesskeys`` to list existing keys, or create a new one via ``POST /accesskeys`` with a custom expiration.

Support Languages
-----------------
Currently, the SDK supports the following programming languages:

- Go: https://github.com/voipbin/voipbin-go
