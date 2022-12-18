"""add_column_transcripts_tm_transcript

Revision ID: 0095a6990ebe
Revises: 044ac45bc2e3
Create Date: 2022-12-19 03:33:01.007065

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0095a6990ebe'
down_revision = '044ac45bc2e3'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table transcripts add column tm_transcript datetime(6) after message;""")
    op.execute("""alter table transcripts add column tm_delete datetime(6) after tm_create;""")


def downgrade():
    op.execute("""alter table transcripts drop column tm_transcript;""")
    op.execute("""alter table transcripts drop column tm_delete;""")

