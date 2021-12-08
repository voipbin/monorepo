"""add channels playbackid

Revision ID: ae08184032e6
Revises: 091439a651db
Create Date: 2021-12-08 17:54:06.699200

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ae08184032e6'
down_revision = '091439a651db'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table channels add column playback_id varchar(255) after bridge_id;""")


def downgrade():
    op.execute("""alter table users drop column playback_id;""")
