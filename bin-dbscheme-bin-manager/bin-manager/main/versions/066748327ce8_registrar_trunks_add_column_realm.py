"""registrar_trunks add column realm

Revision ID: 066748327ce8
Revises: c0c0b02c9e3d
Create Date: 2024-02-16 12:26:42.883201

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '066748327ce8'
down_revision = 'c0c0b02c9e3d'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table registrar_trunks add column realm varchar(255) after auth_types;""")
    op.execute("""alter table extensions add column realm varchar(255) after domain_name;""")

    op.execute("""create index idx_registrar_trunks_realm on registrar_trunks(realm);""")
    op.execute("""create index idx_extensions_realm on extensions(realm);""")


def downgrade():
    op.execute("""alter table registrar_trunks drop column realm;""")
    op.execute("""alter table extensions drop column realm;""")

    op.execute("""alter table registrar_trunks drop index idx_registrar_trunks_realm;""")
    op.execute("""alter table extensions drop index idx_extensions_realm;""")
