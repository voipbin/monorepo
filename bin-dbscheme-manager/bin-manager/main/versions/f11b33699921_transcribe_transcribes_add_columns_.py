"""transcribe_transcribes add columns activeflow_id

Revision ID: f11b33699921
Revises: 0c6ee949d552
Create Date: 2025-03-25 15:05:14.388748

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f11b33699921'
down_revision = '0c6ee949d552'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE transcribe_transcribes ADD COLUMN activeflow_id BINARY(16) AFTER customer_id;""")
    op.execute("""UPDATE transcribe_transcribes SET activeflow_id = UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''));""")
    op.execute("""CREATE INDEX transcribe_transcribes_activeflow_id ON transcribe_transcribes(activeflow_id);""")

    op.execute("""ALTER TABLE transcribe_transcribes ADD COLUMN on_end_flow_id BINARY(16) AFTER activeflow_id;""")
    op.execute("""UPDATE transcribe_transcribes SET on_end_flow_id = UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''));""")
    op.execute("""CREATE INDEX transcribe_transcribes_on_end_flow_id ON transcribe_transcribes(on_end_flow_id);""")


def downgrade():
    op.execute("""ALTER TABLE transcribe_transcribes DROP COLUMN activeflow_id;""")
    op.execute("""ALTER TABLE transcribe_transcribes DROP COLUMN on_end_flow_id;""")
