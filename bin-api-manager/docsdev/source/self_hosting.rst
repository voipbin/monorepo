.. _self-hosting-main:

.. _installation-main:

************
Installation
************

.. note:: **AI Context**

   This page documents how to self-host the full VoIPBin platform on your
   own server using Docker Compose, via the official installer at
   https://github.com/voipbin/voipbin (the ``install/`` directory). It is
   intended for operators running their own VoIPBin instance, not for
   customers of the hosted service at https://voipbin.net.

VoIPBin is an opensource CPaaS platform. The complete stack, including the
SIP edge (Kamailio, RTPEngine), the Asterisk media layer, all backend
microservices, the database, the message bus, and the admin/talk/meet
frontends, runs as a single Docker Compose project on one server.

This section walks through what you need, the four commands that perform
the install, the two install modes (a local/LAN evaluation domain or your
own real domain), what configuration the installer generates, and the
day-to-day CLI used for backup, restore, and upgrades.

.. include:: self_hosting_overview.rst
.. include:: self_hosting_prerequisites.rst
.. include:: self_hosting_install.rst
.. include:: self_hosting_first_login.rst
.. include:: self_hosting_providers.rst
.. include:: self_hosting_configuration.rst
.. include:: self_hosting_envvars.rst
.. include:: self_hosting_troubleshooting.rst
