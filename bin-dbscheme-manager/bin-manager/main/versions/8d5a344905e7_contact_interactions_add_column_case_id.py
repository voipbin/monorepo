"""contact_interactions_add_column_case_id

Revision ID: 8d5a344905e7
Revises: f718e26f2c44
Create Date: 2026-07-07 09:23:10.455404

Adds nullable case_id FK to contact_interactions (VOIP-1228). See
docs/plans/2026-07-07-contact-case-management-design.md §3.2.

Interaction stays the event log; Case is the new grouping layer sitting
alongside it. Nullable: nil for pre-Case historical rows (no backfill,
§8), always set going forward.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '8d5a344905e7'
down_revision = 'f718e26f2c44'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE contact_interactions ADD COLUMN case_id BINARY(16) DEFAULT NULL;""")
    op.execute("""CREATE INDEX idx_contact_interactions_case_id ON contact_interactions(case_id);""")


def downgrade():
    op.execute("""DROP INDEX idx_contact_interactions_case_id ON contact_interactions;""")
    op.execute("""ALTER TABLE contact_interactions DROP COLUMN case_id;""")
