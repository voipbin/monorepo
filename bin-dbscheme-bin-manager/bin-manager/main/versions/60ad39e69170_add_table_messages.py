"""add table messages

Revision ID: 60ad39e69170
Revises: 975995c9c2f8
Create Date: 2022-03-13 18:07:37.705463

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '60ad39e69170'
down_revision = '975995c9c2f8'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
    create table messages(
        -- identity
        id            binary(16),     -- id
        customer_id   binary(16),     -- customer id
        type          varchar(255),

        source  json,
        targets json,

        provider_name         varchar(255), -- message provider's name
        provider_reference_id varchar(255), -- reference id for searching the message info from the provider

        text          text,
        medias        json,
        direction     varchar(255),

        -- timestamps
        tm_create   datetime(6),  --
        tm_update   datetime(6),  --
        tm_delete   datetime(6),  --

        primary key(id)
        );
    """)

    op.execute("""create index idx_messages_customerid on messages(customer_id);""")
    op.execute("""create index idx_messages_provider_name on messages(provider_name);""")
    op.execute("""create index idx_messages_provider_reference_id on messages(provider_reference_id);""")


def downgrade():
    op.execute("""
        drop table messages;
    """)
