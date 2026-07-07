"""contact_case_tag_assignments_create_table

Revision ID: 5c7bc362be27
Revises: 437ca5f2e4ee
Create Date: 2026-07-07 09:23:43.683186

Creates the Case-to-tag junction table, owned by bin-contact-manager
(VOIP-1228). See
docs/plans/2026-07-07-contact-case-management-design.md §7 (round-22
correction).

bin-tag-manager has no generic "taggable resource" registration
mechanism -- it manages only the Tag label itself. The actual
tag-assignment mechanism lives in bin-contact-manager, hardcoded per
resource: this table mirrors contact_tag_assignments
(a1b2c3d4e5f6_contact_create_tables.py) exactly, referencing the same
Tag rows (by tag_id) that Contacts already do. bin-tag-manager itself
needs zero changes for this feature.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '5c7bc362be27'
down_revision = '437ca5f2e4ee'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_case_tag_assignments (
            case_id   BINARY(16) NOT NULL,
            tag_id    BINARY(16) NOT NULL,
            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (case_id, tag_id),
            INDEX idx_contact_case_tag_assignments_tag (tag_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS contact_case_tag_assignments;""")
