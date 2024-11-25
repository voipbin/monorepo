"""customer_accesskeys_create_table

Revision ID: 133ea3a35091
Revises: 3ffdc562b708
Create Date: 2024-11-26 00:30:42.571257

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '133ea3a35091'
down_revision = '3ffdc562b708'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table customer_accesskeys (
            id          binary(16),
            customer_id binary(16),

            name    varchar(255),
            detail  text,

            token varchar(1023),

            tm_expire datetime(6), -- Expiry timestamp

            tm_create datetime(6), -- Created timestamp
            tm_update datetime(6), -- Updated timestamp
            tm_delete datetime(6), -- Deleted timestamp

            primary key(id)
        );
    """)
    op.execute("""create index idx_customer_accesskeys_customer_id on customer_accesskeys(customer_id);""")
    op.execute("""create index idx_customer_accesskeys_token on customer_accesskeys(token);""")


def downgrade():
    op.execute("""drop table customer_accesskeys;""")
