"""add table agents

Revision ID: 4cb21f78b337
Revises: af6321e8bdef
Create Date: 2021-11-25 13:51:15.067401

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '4cb21f78b337'
down_revision = 'af6321e8bdef'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table agents(
            -- identity
            id            binary(16),  -- id
            user_id       integer,
            username      varchar(255), -- username
            password_hash varchar(255), -- password hash

            -- basic info
            name        varchar(255),
            detail      text,
            ring_method varchar(255), -- ring method

            status      varchar(255),  -- agent's status
            permission  integer,
            tag_ids     json,
            addresses   json,

            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_agents_userid on agents(user_id);
    """)
    op.execute("""
        create index idx_agents_username on agents(username);
    """)


def downgrade():
    op.execute("""
        drop table agents;
    """)
