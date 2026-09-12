.. _variable-main:

**************
Variable
**************
Define and manage flow variables that store dynamic data for use within automation workflows.

.. note::

   Variables are not a standalone REST resource and have no dedicated ``/variables`` endpoints or API tag. They are seeded via the optional ``variables`` field on ``POST /activeflows`` (or ``POST /calls``), and from then on are read and written only from within the flow itself (``${key}`` substitution, the ``variable_set`` action, and actions that store their own results). See :ref:`variable-overview` for details.

.. toctree::
   :maxdepth: 2

   variable_overview
   variable_variable
