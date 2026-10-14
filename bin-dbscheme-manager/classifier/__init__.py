"""Migration classifier: decides whether a pending Alembic range is provably additive."""

import sys

# Importing a revision module must never write bytecode next to it: the source
# gate treats anything but regular revision files in versions/ as a violation.
sys.dont_write_bytecode = True
