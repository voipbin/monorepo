"""Remove customers column username password permission

Revision ID: c0c0b02c9e3d
Revises: dbbf8225587a
Create Date: 2024-02-14 16:52:27.942738

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c0c0b02c9e3d'
down_revision = 'dbbf8225587a'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table customers drop column username;""")
    op.execute("""alter table customers drop column password_hash;""")
    op.execute("""alter table customers drop column permission_ids;""")


def downgrade():
    op.execute("""alter table customers add column username varchar(255) after id;""")
    op.execute("""alter table customers add column password_hash varchar(255) after username;""")
    op.execute("""alter table customers add column permission_ids json after webhook_uri;""")
