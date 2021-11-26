"""add table tags

Revision ID: 091439a651db
Revises: 4cb21f78b337
Create Date: 2021-11-27 03:50:55.808869

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '091439a651db'
down_revision = '4cb21f78b337'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table tags(
            -- identity
            id            binary(16),  -- id
            user_id       integer,

            -- basic info
            name        varchar(255),
            detail      text,

            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_tags_userid on tags(user_id);
    """)
    op.execute("""
        create index idx_tags_name on tags(name);
    """)


def downgrade():
    op.execute("""
        drop table tags;
    """)

