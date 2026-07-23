"""contact_cases_add_column_reference_id

Revision ID: abfdbef47552
Revises: 80ddd8772905
Create Date: 2026-07-24 03:36:27.676637

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'abfdbef47552'
down_revision = '80ddd8772905'
branch_labels = None
depends_on = None


def upgrade():
    op.execute(
        """alter table contact_cases add column reference_id varchar(255) not null default '' after detail;"""
    )
    op.execute(
        """create index idx_case_customer_reference_id on contact_cases (customer_id, reference_id);"""
    )


def downgrade():
    op.execute("""drop index idx_case_customer_reference_id on contact_cases;""")
    op.execute("""alter table contact_cases drop column reference_id;""")
