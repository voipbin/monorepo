.. _quickstart_sandbox:

Sandbox
-------
The VoIPBIN Sandbox lets you run the entire VoIPBIN platform on your local machine with a single command. No sign-up required — it comes with a pre-configured admin account and is ready to use immediately.

.. note:: **AI Implementation Hint**

   The Sandbox is a fully functional local VoIPBIN instance. When working with the Sandbox, replace all API URLs from ``https://api.voipbin.net`` to ``https://api.voipbin.test:8443``. The Sandbox uses a self-signed SSL certificate, so add ``-k`` (insecure) flag to curl commands. The default credentials are ``admin@localhost`` / ``admin@localhost``.

Prerequisites
~~~~~~~~~~~~~
- Docker and Docker Compose v2

Install and Run
~~~~~~~~~~~~~~~
Clone the sandbox repository and launch the interactive CLI:

.. code::

    $ git clone https://github.com/voipbin/sandbox.git
    $ cd sandbox
    $ sudo ./voipbin

Initialize and start the services:

.. code::

    voipbin> init
    voipbin> start

Once started, you can log in with the default admin account:

- **Username**: admin@localhost
- **Password**: admin@localhost

For full documentation, troubleshooting, and advanced usage, see the `Sandbox GitHub repository <https://github.com/voipbin/sandbox>`_.

Using Tutorials with the Sandbox
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
All tutorials in this documentation use ``https://api.voipbin.net`` as the API endpoint. If you are using the sandbox, substitute it with your local sandbox endpoint:

- **Production**: ``https://api.voipbin.net``
- **Sandbox**: ``https://api.voipbin.test:8443``

For example, to generate a token using the sandbox:

.. code::

    $ curl -k --request POST 'https://api.voipbin.test:8443/auth/login' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "username": "admin@localhost",
            "password": "admin@localhost"
        }'

Troubleshooting
~~~~~~~~~~~~~~~

* **"init" or "start" command fails:**
    * **Cause:** Docker daemon is not running or Docker Compose v2 is not installed.
    * **Fix:** Verify Docker is running with ``docker compose version``. Must show v2.x or later.

* **Port 8443 already in use:**
    * **Cause:** Another service is occupying port 8443.
    * **Fix:** Stop the conflicting service, or check with ``ss -tlnp | grep 8443`` (Linux) / ``lsof -i :8443`` (macOS).

* **SSL certificate error (without ``-k`` flag):**
    * **Cause:** The Sandbox uses a self-signed SSL certificate that is not trusted by default.
    * **Fix:** Add the ``-k`` flag to all curl commands when using the Sandbox.

* **401 Unauthorized with default credentials:**
    * **Cause:** The Sandbox has not fully initialized.
    * **Fix:** Run ``voipbin> init`` again and wait for it to complete before running ``voipbin> start``.
