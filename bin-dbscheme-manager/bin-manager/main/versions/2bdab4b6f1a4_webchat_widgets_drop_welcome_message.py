"""webchat_widgets_drop_welcome_message

Revision ID: 2bdab4b6f1a4
Revises: 1a1f28d6842c
Create Date: 2026-07-18 08:45:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '2bdab4b6f1a4'
down_revision = '1a1f28d6842c'
branch_labels = None
depends_on = None


def upgrade():
    # Removes the static welcome_message text field in favor of relying
    # solely on Widget.SessionFlowID (an existing Flow-trigger mechanism)
    # for any "welcome the visitor" behavior. See design doc
    # 2026-07-18-webchat-welcome-message-flow-consolidation-design.md.
    #
    # This is a data-loss migration by design (locked decision, §3.2 of
    # the design doc): no backfill into a Flow, existing customers who
    # had welcome_message set lose it silently and must reconfigure via
    # a SessionFlowID Flow. `DROP COLUMN` on MySQL is not covered by
    # 8.0's instant-DDL algorithm (it always requires a table rebuild),
    # so this can briefly hold a metadata/write lock -- acceptable given
    # webchat-manager's current low row count (service shipped
    # 2026-07-16, no external consumers beyond square-admin yet).
    op.execute("""
        alter table webchat_widgets
        drop column welcome_message;
    """)


def downgrade():
    # Restores the column (empty) but CANNOT restore lost data -- this
    # downgrade is lossy by design, matching the locked "no backfill"
    # decision. Do not roll back below this revision expecting prior
    # welcome_message values to reappear.
    op.execute("""
        alter table webchat_widgets
        add column welcome_message text after direct_hash;
    """)
