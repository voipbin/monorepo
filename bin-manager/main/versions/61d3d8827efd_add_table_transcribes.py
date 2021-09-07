"""add table transcribes

Revision ID: 61d3d8827efd
Revises: c63a7f966a05
Create Date: 2021-09-06 21:22:28.457516

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = '61d3d8827efd'
down_revision = 'c63a7f966a05'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'transcribes' in tables:
        return

    op.execute("""
        create table transcribes(
            -- identity
            id            binary(16),   -- id
            user_id       integer,      -- user id
            type          varchar(16),  -- type of transcribe
            reference_id  binary(16),   -- call/conference/recording's id
            host_id       binary(16),   -- host id

            language          varchar(16),    -- BCP47 type's language code. en-US
            webhook_uri       varchar(1023),  -- webhook uri
            webhook_method    varchar(16),    -- webhook method

            transcripts json, -- transcripts

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_transcribes_reference_id on transcribes(reference_id);
    """)

    op.execute("""
        create index idx_transcribes_user_id on transcribes(user_id);
    """)

def downgrade():
    op.execute("""
        drop table transcribes;
    """)
