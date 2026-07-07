"""contact_resolutions_case_support

Revision ID: 99e7e955a149
Revises: 8d5a344905e7
Create Date: 2026-07-07 09:24:03.001122

Adds case-level attribution support to contact_resolutions (VOIP-1228).
See docs/plans/2026-07-07-contact-case-management-design.md §3.3.

This is additive-safe at the DB level (relaxing interaction_id NOT NULL
to nullable cannot reject previously-valid data), but requires a real
Go-side breaking-change refactor to dbhandler/contacthandler call sites
that assumed non-nullability as a scoping parameter -- see Phase 2 of
the implementation plan, not assumed free here.

A Resolution row now has two independent modes:
  - case_id set (primary path): "this whole case belongs to this contact."
  - interaction_id set (exception path, existing behavior): fine-grained
    override of a single Interaction.

case_positive_uk (generated column + UNIQUE) guarantees at most one
active case-level positive Resolution per case, mirroring the
open_peer_uk technique in contact_cases (same MySQL partial-unique-index
workaround).
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '99e7e955a149'
down_revision = '8d5a344905e7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE contact_resolutions ADD COLUMN case_id BINARY(16) DEFAULT NULL;""")
    op.execute("""ALTER TABLE contact_resolutions MODIFY COLUMN interaction_id BINARY(16) DEFAULT NULL;""")
    op.execute("""
        ALTER TABLE contact_resolutions
        ADD CONSTRAINT chk_resolution_case_or_interaction
        CHECK (interaction_id IS NOT NULL OR case_id IS NOT NULL);
    """)
    op.execute("""
        ALTER TABLE contact_resolutions
        ADD COLUMN case_positive_uk BINARY(16) GENERATED ALWAYS AS (
            IF(resolution_type = 'positive' AND interaction_id IS NULL AND tm_delete IS NULL,
               case_id, NULL)
        ) STORED;
    """)
    op.execute("""ALTER TABLE contact_resolutions ADD UNIQUE INDEX uq_resolution_case_positive (case_positive_uk);""")
    op.execute("""CREATE INDEX idx_contact_resolutions_case_id ON contact_resolutions(case_id);""")


def downgrade():
    op.execute("""DROP INDEX idx_contact_resolutions_case_id ON contact_resolutions;""")
    op.execute("""ALTER TABLE contact_resolutions DROP INDEX uq_resolution_case_positive;""")
    op.execute("""ALTER TABLE contact_resolutions DROP COLUMN case_positive_uk;""")
    op.execute("""ALTER TABLE contact_resolutions DROP CONSTRAINT chk_resolution_case_or_interaction;""")
    op.execute("""ALTER TABLE contact_resolutions MODIFY COLUMN interaction_id BINARY(16) NOT NULL;""")
    op.execute("""ALTER TABLE contact_resolutions DROP COLUMN case_id;""")
