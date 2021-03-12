"""add table flows

Revision ID: 890ee70f6ffd
Revises: baa3bbb6c32c
Create Date: 2021-02-23 14:10:36.731316

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = '890ee70f6ffd'
down_revision = 'baa3bbb6c32c'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'flows' in tables:
        return

    op.execute("""
        create table flows(
            -- identity
            id binary(16),
            user_id int(10),

            name varchar(255),
            detail text,

            actions json,

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),

            primary key(id)
        )
    """)

def downgrade():
    op.execute("""
        drop table flows;
    """)
