"""contact_case_tag_assignments_drop_table

Revision ID: 43947d3f7312
Revises: c2e53c0f6453
Create Date: 2026-07-14

Drops contact_case_tag_assignments (design VOIP-1254), the junction
table created in 5c7bc362be27 for Case-to-tag assignment. Superseded by
contact_cases.tag_ids (added in c2e53c0f6453, the immediately preceding
migration), which mirrors bin-queue-manager's Queue.TagIDs storage
exactly. Confirmed 2026-07-14 via read-only SELECT COUNT(*) against
production bin_manager: contact_case_tag_assignments has 0 rows (this
feature was never exposed via any client-facing REST surface --
VOIP-1242's REST layer was never built), so this DROP is a pure
schema-only change with zero data loss on upgrade().

Asymmetric downgrade data-loss risk (accepted, see also
c2e53c0f6453's docstring): this migration's downgrade() only recreates
the junction table's empty STRUCTURE -- it cannot repopulate it with
any tag_ids data written to the new JSON column after this PR ships,
because the junction-table concept (one row per case-tag pair) is
retired by this migration's own upgrade(). A real `alembic downgrade`
run against a live target would, in strict LIFO order, recreate this
empty junction table BEFORE the earlier migration's downgrade() drops
the tag_ids column -- so there is no path to recover any tag_ids data
written in between via a downgrade sequence. Accepted per this repo's
standing rule that AI never runs alembic downgrade against a real
target in the first place; a human operator running downgrade against
a real database already owns this class of risk for any schema
rollback.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '43947d3f7312'
down_revision = 'c2e53c0f6453'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""DROP TABLE IF EXISTS contact_case_tag_assignments;""")


def downgrade():
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_case_tag_assignments (
            case_id   BINARY(16) NOT NULL,
            tag_id    BINARY(16) NOT NULL,
            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (case_id, tag_id),
            INDEX idx_contact_case_tag_assignments_tag (tag_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)
