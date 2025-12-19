"""add_table_transcripts

Revision ID: 044ac45bc2e3
Revises: d9dcf9645e24
Create Date: 2022-12-18 16:24:07.929770

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '044ac45bc2e3'
down_revision = 'd9dcf9645e24'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table transcripts(
            -- identity
            id            binary(16),   -- id
            customer_id   binary(16),   -- customer id
            transcribe_id binary(16),

            direction varchar(16),
            message   text,

            -- timestamps
            tm_create datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_transcripts_customerid on transcripts(customer_id);""")
    op.execute("""create index idx_transcripts_transcribe_id on transcripts(transcribe_id);""")


def downgrade():
    op.execute("""drop table transcripts;""")
