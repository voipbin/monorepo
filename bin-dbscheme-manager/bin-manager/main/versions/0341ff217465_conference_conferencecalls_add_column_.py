"""conference_conferencecalls add column activeflow_id

Revision ID: 0341ff217465
Revises: 5f23751b14a9
Create Date: 2025-04-01 01:00:05.444619

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0341ff217465'
down_revision = '5f23751b14a9'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE conference_conferencecalls ADD COLUMN activeflow_id BINARY(16) AFTER customer_id;""")
    op.execute("""UPDATE conference_conferencecalls SET activeflow_id = UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''));""")
    op.execute("""CREATE INDEX conference_conferencecalls_activeflow_id ON conference_conferencecalls(activeflow_id);""")


def downgrade():
    op.execute("""ALTER TABLE conference_conferencecalls DROP COLUMN activeflow_id;""")
