"""Add table billing

Revision ID: d035a214f7da
Revises: aa2bc6442bc7
Create Date: 2023-06-14 01:27:38.053868

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd035a214f7da'
down_revision = 'aa2bc6442bc7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table billing_accounts(
            id            binary(16),
            customer_id   binary(16),

            type varchar(255),

            name    varchar(255),
            detail  text,

            balance float,

            payment_type      varchar(255),
            payment_method    varchar(255),

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_billing_accounts_customer_id on billing_accounts(customer_id);""")
    op.execute("""create index idx_billing_accounts_create on billing_accounts(tm_create);""")

    op.execute("""
        create table billing_billings(
            id            binary(16),
            customer_id   binary(16),
            account_id    binary(16),

            status varchar(32),

            reference_type  varchar(32),
            reference_id    binary(16),

            cost_per_unit float,
            cost_total    float,

            billing_unit_count  float,

            -- timestamps
            tm_billing_start  datetime(6),
            tm_billing_end    datetime(6),

            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_billing_billings_customer_id on billing_billings(customer_id);""")
    op.execute("""create index idx_billing_billings_account_id on billing_billings(account_id);""")
    op.execute("""create index idx_billing_billings_reference_id on billing_billings(reference_id);""")
    op.execute("""create index idx_billing_billings_create on billing_billings(tm_create);""")


def downgrade():
    op.execute("""drop table billing_accounts;""")
    op.execute("""drop table billing_billings;""")
