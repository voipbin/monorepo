"""contact_cases_add_column_tag_ids

Revision ID: c2e53c0f6453
Revises: 1d0f4d07ff58
Create Date: 2026-07-14

Adds contact_cases.tag_ids (design VOIP-1254). Replaces the
contact_case_tag_assignments junction table (5c7bc362be27) with a plain
JSON column, mirroring bin-queue-manager's Queue.TagIDs storage exactly
(no junction table, no reverse-lookup index). Case tag usage is
low-frequency and agent-driven (not a routing hot path), strictly
lighter than Queue's own use of the same pattern for real-time
agent-tag intersection matching at call-routing time.

The junction-table rationale from the original design review
(referential integrity, write concurrency) does not actually hold:
bin-contact-manager does not subscribe to bin-tag-manager's
tag_deleted event today, so referential integrity is not enforced
either way; and Case tag writes are low-frequency UI actions, not a
routing hot path.

Confirmed 2026-07-14: contact_case_tag_assignments has 0 rows in
production (read-only SELECT COUNT(*) via a throwaway kubectl
diagnostic pod), so the follow-up drop migration
(<rev>_contact_case_tag_assignments_drop_table.py) needs no backfill
step.

downgrade() drops the column, an unrecoverable data-loss operation for
any tag_ids written after this ships (see the follow-up drop
migration's docstring for the full asymmetric-downgrade-risk
rationale) -- accepted per this repo's standing rule that AI never
runs alembic downgrade against a real target; a human operator running
downgrade already owns this class of risk for any schema rollback.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c2e53c0f6453'
down_revision = '1d0f4d07ff58'
branch_labels = None
depends_on = None


def upgrade():
    op.execute(
        """ALTER TABLE contact_cases ADD COLUMN tag_ids JSON DEFAULT NULL AFTER previous_case_id;"""
    )


def downgrade():
    op.execute("""ALTER TABLE contact_cases DROP COLUMN tag_ids;""")
