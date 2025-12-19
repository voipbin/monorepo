"""email_emails create table

Revision ID: 1c0bc3259406
Revises: 462add34b7f5
Create Date: 2025-03-15 02:45:49.826568

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1c0bc3259406'
down_revision = '462add34b7f5'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table email_emails (
            id          binary(16),
            customer_id binary(16),

            activeflow_id binary(16),

            provider_type varchar(255),
            provider_reference_id varchar(255),

            source        json,
            destinations  json,

            status  varchar(255),
            subject varchar(255),
            content text,

            attachments json,

            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)
    op.execute("""create index idx_email_emails_customer_id on email_emails(customer_id);""")
    op.execute("""create index idx_email_emails_activeflow_id on email_emails(activeflow_id);""")
    op.execute("""create index idx_email_emails_provider_type on email_emails(provider_type);""")
    op.execute("""create index idx_email_emails_provider_reference_id on email_emails(provider_reference_id);""")

def downgrade():
    op.execute("""drop table email_emails;""")
