"""conversation_messages_add_column_subject

Revision ID: 4579f211877c
Revises: a63b82d73655
Create Date: 2026-06-27 00:42:00.770618

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '4579f211877c'
down_revision = 'a63b82d73655'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conversation_messages add subject varchar(255) not null default '' after text;""")


def downgrade():
    op.execute("""alter table conversation_messages drop column subject;""")
