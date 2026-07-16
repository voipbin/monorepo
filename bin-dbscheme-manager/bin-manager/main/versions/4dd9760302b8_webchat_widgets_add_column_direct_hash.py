"""webchat_widgets_add_column_direct_hash

Revision ID: 4dd9760302b8
Revises: c9602a744cb3
Create Date: 2026-07-17 05:46:14.243572

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '4dd9760302b8'
down_revision = 'c9602a744cb3'
branch_labels = None
depends_on = None


def upgrade():
    # direct_hash was missing from the original webchat_widgets_sessions_messages_
    # migration (c9602a744cb3) -- the table only stored direct_id (the internal
    # linkage to bin-direct-manager), never the hash string itself that the
    # embed script needs (data-hash="<hash>"). Every other direct-hash-issuing
    # resource (ai_ais, ai_ai_teams, etc.) stores both direct_id and direct_hash;
    # this backfills webchat_widgets to match that convention.
    op.execute("""
        alter table webchat_widgets
        add column direct_hash varchar(255) after direct_id;
    """)


def downgrade():
    op.execute("""
        alter table webchat_widgets
        drop column direct_hash;
    """)
