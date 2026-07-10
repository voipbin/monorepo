"""contact_cases_add_column_name_detail

Revision ID: a10299e7932a
Revises: 29ba6e093d30
Create Date: 2026-07-10 09:42:42.699470

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a10299e7932a'
down_revision = '29ba6e093d30'
branch_labels = None
depends_on = None


def upgrade():
    op.execute(
        """alter table contact_cases add column name varchar(255) not null default '' after reference_type;"""
    )
    op.execute("""alter table contact_cases add column detail text after name;""")


def downgrade():
    op.execute("""alter table contact_cases drop column detail;""")
    op.execute("""alter table contact_cases drop column name;""")
