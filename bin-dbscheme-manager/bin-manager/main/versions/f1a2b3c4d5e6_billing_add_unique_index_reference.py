"""billing_add_unique_index_reference_type_reference_id

Revision ID: f1a2b3c4d5e6
Revises: e1f2a3b4c5d6
Create Date: 2026-02-12 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'f1a2b3c4d5e6'
down_revision = 'e1f2a3b4c5d6'
branch_labels = None
depends_on = None


def upgrade():
    # Step 1: Clean up any remaining sentinel tm_delete values.
    # Earlier code used '9999-01-01' as a soft-delete sentinel; normalize to NULL.
    op.execute("""
        UPDATE billing_billings SET tm_delete = NULL
        WHERE tm_delete = '9999-01-01 00:00:00.000000';
    """)

    # Step 2: Resolve any existing duplicates on (reference_type, reference_id).
    # Keep the preferred record per group: prefer active (tm_delete IS NULL) over
    # deleted, then latest by tm_create. Delete the rest.
    op.execute("""
        DELETE b1 FROM billing_billings b1
        INNER JOIN billing_billings b2
            ON b1.reference_type = b2.reference_type
            AND b1.reference_id = b2.reference_id
            AND (
                (b2.tm_delete IS NULL AND b1.tm_delete IS NOT NULL)
                OR (b1.tm_delete IS NULL AND b2.tm_delete IS NULL AND b1.tm_create < b2.tm_create)
                OR (b1.tm_delete IS NULL AND b2.tm_delete IS NULL AND b1.tm_create = b2.tm_create AND b1.id < b2.id)
                OR (b1.tm_delete IS NOT NULL AND b2.tm_delete IS NOT NULL AND b1.tm_create < b2.tm_create)
                OR (b1.tm_delete IS NOT NULL AND b2.tm_delete IS NOT NULL AND b1.tm_create = b2.tm_create AND b1.id < b2.id)
            );
    """)

    # Step 3: Add unique index for billing record deduplication.
    # This enforces that each (reference_type, reference_id) pair is unique,
    # which is used by the free-tier monthly credit top-up to prevent
    # duplicate processing across multiple pods via deterministic UUID v5 reference IDs.
    op.execute("""
        CREATE UNIQUE INDEX idx_billing_billings_ref_type_ref_id
        ON billing_billings(reference_type, reference_id);
    """)


def downgrade():
    op.execute("""
        DROP INDEX idx_billing_billings_ref_type_ref_id ON billing_billings;
    """)
