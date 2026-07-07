"""contact_case_notes_create_table

Revision ID: 437ca5f2e4ee
Revises: 99e7e955a149
Create Date: 2026-07-07 09:23:43.480542

Creates the CaseNote table (VOIP-1228). See
docs/plans/2026-07-07-contact-case-management-design.md §3.5.

Physically separate from contact_interactions -- CaseNote is never
surfaced in any customer-facing webhook or API response. Soft-delete
via tm_delete (NULL = active), matching contact_resolutions'
retraction pattern rather than contact_interactions' append-only
(no-delete) pattern, since a note can be removed by the authoring agent.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '437ca5f2e4ee'
down_revision = '99e7e955a149'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE contact_case_notes (
            id          BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,
            case_id     BINARY(16) NOT NULL,

            author_type VARCHAR(32) NOT NULL DEFAULT '',   -- agent | system
            author_id   BINARY(16)  DEFAULT NULL,

            text TEXT NOT NULL,

            tm_create DATETIME(6),
            tm_update DATETIME(6),
            tm_delete DATETIME(6) DEFAULT NULL,

            PRIMARY KEY (id),
            INDEX idx_contact_case_notes_case_id (case_id, tm_delete)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS contact_case_notes;""")
