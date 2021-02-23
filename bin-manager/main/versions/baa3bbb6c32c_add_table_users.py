"""add table users

Revision ID: baa3bbb6c32c
Revises: 4c348a96c3b8
Create Date: 2021-02-23 14:09:10.752237

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = 'baa3bbb6c32c'
down_revision = '4c348a96c3b8'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'users' in tables:
        return

    op.execute("""
        create table users(
            -- identity
            id            integer primary key auto_increment,  -- id

            username      varchar(255), -- username
            password_hash varchar(255), -- password hash

            permission integer,

            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6)
        );
    """)

    op.execute("""
        create index idx_users_username on users(username);
    """)



def downgrade():
    op.execute("""
        drop table users;
    """)
