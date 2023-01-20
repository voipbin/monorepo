"""add column calls external media id

Revision ID: d3765d22fa1d
Revises: c66d262cc66c
Create Date: 2023-01-18 15:48:20.626861

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd3765d22fa1d'
down_revision = 'c66d262cc66c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls add external_media_id binary(16) default '' after recording_ids;""")
    op.execute("""alter table confbridges add external_media_id binary(16) default '' after recording_ids;""")


def downgrade():
    op.execute("""alter table calls drop column external_media_id;""")
    op.execute("""alter table confbridges drop column external_media_id;""")
