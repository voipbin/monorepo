"""add table numbers

Revision ID: 520eee292397
Revises: 3743a43a215d
Create Date: 2021-02-23 13:56:46.097870

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = '520eee292397'
down_revision = '3743a43a215d'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'numbers' in tables:
        return

    op.execute("""
        create table numbers(
            -- identity
            id        binary(16),     -- id
            number    varchar(255),   -- number
            flow_id   binary(16),     -- flow id
            user_id   integer,        -- user id

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_numbers_number on numbers(number);
    """)


def downgrade():
    op.execute("""
        drop table numbers;
    """)
