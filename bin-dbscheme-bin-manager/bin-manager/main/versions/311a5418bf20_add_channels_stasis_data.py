"""add_channels_stasis_data

Revision ID: 311a5418bf20
Revises: 20da0907370d
Create Date: 2021-10-28 13:47:37.226516

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '311a5418bf20'
down_revision = '20da0907370d'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table channels change stasis stasis_name varchar(255);""")
    op.execute("""alter table channels add stasis_data json after stasis_name;""")


def downgrade():
    op.execute("""alter table channels change stasis_name stasis varchar(255);""")
    op.execute("""alter table channels drop column stasis_data;""")
