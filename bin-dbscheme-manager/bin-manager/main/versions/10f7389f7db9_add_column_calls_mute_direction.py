"""add column calls mute_direction

Revision ID: 10f7389f7db9
Revises: 453add0eb376
Create Date: 2023-04-04 03:18:44.355497

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '10f7389f7db9'
down_revision = '453add0eb376'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table channels add column mute_direction varchar(16) after direction;""")
    op.execute("""alter table calls add column mute_direction varchar(16) after direction;""")

    op.execute("""update channels set mute_direction = "";""")
    op.execute("""update calls set mute_direction = "";""")


def downgrade():
    op.execute("""alter table channels drop column mute_direction;""")
    op.execute("""alter table calls drop column mute_direction;""")
