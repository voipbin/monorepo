"""add_column_transcribes_status

Revision ID: fa7ab340d705
Revises: 0095a6990ebe
Create Date: 2022-12-19 14:46:08.196646

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'fa7ab340d705'
down_revision = '0095a6990ebe'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table transcribes add column status varchar(16) after reference_id;""")
    op.execute("""alter table transcribes drop column transcripts;""")
    op.execute("""alter table transcribes change type reference_type varchar(16);""")


def downgrade():
    op.execute("""alter table transcribes drop column status;""")
    op.execute("""alter table transcribes add column transcripts json after message;""")
    op.execute("""alter table transcribes change reference_type type varchar(16);""")
