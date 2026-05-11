.. _self-hosting-main:

.. _self-hosting-overview:
.. _self-hosting-prerequisites:
.. _self-hosting-install:
.. _self-hosting-configuration:
.. _self-hosting-envvars:
.. _self-hosting-troubleshooting:

************
Self-Hosting
************

.. note:: **AI Context**

   This page documents how to deploy a full VoIPBin platform on your own
   Google Cloud Platform project using the official installer at
   https://github.com/voipbin/install. It is intended for operators
   running their own VoIPBin instance, not for customers of the hosted
   service at https://voipbin.net.

VoIPBin is an opensource CPaaS platform. The complete stack, including the
SIP edge (Kamailio, RTPEngine), the Asterisk media layer, all backend
microservices, the database, the message bus, and the admin/talk/meet
frontends, can be installed into a Google Cloud Platform project with a
single CLI.

This section walks through what you need, the three commands that perform
the install, what configuration the installer generates, and which
environment variables you typically need to adjust before the platform is
ready for real traffic.

.. include:: self_hosting_overview.rst
.. include:: self_hosting_prerequisites.rst
.. include:: self_hosting_install.rst
.. include:: self_hosting_configuration.rst
.. include:: self_hosting_envvars.rst
.. include:: self_hosting_troubleshooting.rst
