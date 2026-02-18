.. _support:

=======
Support
=======

This section provides information on how to get help with VoIPBIN services.

.. _support-contact:

Contact Us
==========

For technical assistance, questions, or feedback:

- **Discord:** https://discord.gg/waztvb63Yx
- **Email:** sungtae@voipbin.net

When contacting us, please include:

- Detailed description of the issue or question
- Your deployment environment (GCP, on-premise, etc.)
- VoIPBIN version or commit hash you're running
- Relevant log outputs or error messages
- Steps to reproduce the issue
- Expected vs actual behavior
- Any configuration files or settings (sanitized of sensitive data)

.. _support-resources:

Additional Resources
====================

**Documentation:**

- **API Reference (ReDoc):** https://api.voipbin.net/redoc/index.html
- **Developer Docs (Sphinx):** https://api.voipbin.net/docs/

**Web Interfaces:**

- **Admin Console:** https://admin.voipbin.net/
- **Agent Interface:** https://talk.voipbin.net/

.. note:: **AI Implementation Hint**

   For programmatic debugging, check the resource status via ``GET /<resource>/{id}`` and review recent events via WebSocket or webhook logs. Most issues can be diagnosed by examining the call/flow/transcribe status fields.

.. _support-troubleshooting-tips:

Before Contacting Us
====================

To help us resolve your issue faster:

1. **Review the documentation** - Many common issues are covered in our guides
2. **Gather relevant information** - Collect IDs, timestamps, and error messages
3. **Test with minimal configuration** - Isolate the issue if possible
4. **Check your API credentials** - Ensure tokens and access keys are valid
5. **Search Discord** - Your question may have already been answered
