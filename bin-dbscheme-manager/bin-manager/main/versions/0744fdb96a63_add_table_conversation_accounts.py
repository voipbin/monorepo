"""add table conversation_accounts

Revision ID: 0744fdb96a63
Revises: f11eeaae9441
Create Date: 2023-05-31 02:25:32.203038

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0744fdb96a63'
down_revision = 'f11eeaae9441'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table conversation_accounts(
            -- identity
            id            binary(16),     -- id
            customer_id   binary(16),     -- customer id

            type varchar(32),

            name    varchar(255),
            detail  text,

            secret  text,
            token   text,

            -- timestamps
            tm_create   datetime(6),  --
            tm_update   datetime(6),  --
            tm_delete   datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_conversation_accounts_customerid on conversation_accounts(customer_id);""")

    op.execute("""alter table conversation_conversations add account_id binary(16) after customer_id;""")
    op.execute("""update conversation_conversations set account_id = "";""")


def downgrade():
    op.execute("""drop table conversation_accounts;""")

    op.execute("""alter table conversation_conversations drop column account_id;""")

