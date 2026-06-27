"""contact_crm_m1_migrate_addresses

Revision ID: bbcf80d332eb
Revises: ac5d4e18060c
Create Date: 2026-06-27 22:27:18.763159

VOIP-1207 (CRM M1). Migrates the legacy normalized child tables
contact_phone_numbers + contact_emails into the unified contact_addresses
table created in VOIP-1206 (ac5d4e18060c). Design:
docs/plans/2026-06-27-voip-1207-crm-m1-address-migration.md

Mapping (active parent contacts only, tm_delete IS NULL):
  phone -> type='tel',   target=number_e164
  email -> type='email', target=LOWER(address)
  contact_id / customer_id / tm_create preserved; target_name=''.

This is ONE INSERT from a unified CTE (NOT two independent INSERT...SELECT):
a contact with both a primary phone AND a primary email maps both rows to
primary_contact_uk = contact_id, and MySQL/MariaDB check UNIQUE immediately,
so two independent inserts would fail with ERROR 1062. The CTE unions both
legacy tables, then resolves in TWO STAGES, ordering by lowest legacy id
throughout (fully deterministic, repeatable):

  STAGE 1 (dedup): one row per (customer_id, type, target); lowest id wins.
    Satisfies UNIQUE(customer_id, type, target). Folds email case/sub-type
    duplicates (same-contact) and collapses cross-contact shared emails to a
    single owning contact (lowest id). tel cannot collide at source
    (legacy UNIQUE(customer_id, number_e164)).
  STAGE 2 (primary): over the STAGE 1 survivors, keep is_primary=1 only for
    the lowest-id row among a contact's primaries; demote the rest. Satisfies
    UNIQUE(customer_id, primary_contact_uk). Running on survivors means dedup
    can never delete the chosen primary.

DO NOT remove is_primary from the STAGE 2 partition key: the is_primary=0
group also gets primary_rank=1, but the final CASE's is_primary=1 guard
discards it. Partitioning by (contact_id, is_primary) keeps the primary
ranking from being polluted by non-primary rows.

DOWNGRADE SAFETY: downgrade() is TRUNCATE contact_addresses. This is safe
ONLY before the M1 code is deployed, while contact_addresses holds exactly
this migration's output and has no other writer. AFTER the contact-manager
M1 code is deployed, contact_addresses is the live source of truth and this
downgrade would destroy production data. Roll back via binary redeploy +
forward-fix migration, NEVER an alembic downgrade in production.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'bbcf80d332eb'
down_revision = 'ac5d4e18060c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        INSERT INTO contact_addresses
          (id, customer_id, contact_id, type, target, target_name, is_primary, tm_create)
        WITH unioned AS (
          SELECT p.customer_id, p.contact_id, 'tel' AS type, p.number_e164 AS target,
                 p.is_primary, p.tm_create, p.id AS id_src
          FROM contact_phone_numbers p
          JOIN contact_contacts c ON c.id = p.contact_id AND c.tm_delete IS NULL
          UNION ALL
          SELECT e.customer_id, e.contact_id, 'email', LOWER(e.address),
                 e.is_primary, e.tm_create, e.id
          FROM contact_emails e
          JOIN contact_contacts c ON c.id = e.contact_id AND c.tm_delete IS NULL
        ),
        deduped AS (
          SELECT u.*,
            ROW_NUMBER() OVER (PARTITION BY u.customer_id, u.type, u.target
                               ORDER BY u.id_src ASC) AS dup_rank
          FROM unioned u
        ),
        survivors AS (
          SELECT * FROM deduped WHERE dup_rank = 1
        ),
        ranked AS (
          SELECT s.*,
            ROW_NUMBER() OVER (PARTITION BY s.contact_id, s.is_primary
                               ORDER BY s.id_src ASC) AS primary_rank
          FROM survivors s
        )
        SELECT
          UNHEX(REPLACE(UUID(), '-', '')),
          customer_id, contact_id, type, target, '',
          CASE WHEN is_primary = 1 AND primary_rank = 1 THEN 1 ELSE 0 END,
          tm_create
        FROM ranked;
    """)


def downgrade():
    # SAFE ONLY pre-cutover. See module docstring: after the M1 code is
    # deployed, contact_addresses is the live source of truth and this would
    # destroy production data. Roll back via binary redeploy, not downgrade.
    op.execute("""TRUNCATE TABLE contact_addresses;""")
