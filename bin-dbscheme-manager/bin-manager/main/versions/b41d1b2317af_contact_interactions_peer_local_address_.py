"""contact_interactions_peer_local_address_json

Revision ID: b41d1b2317af
Revises: 167bebb7c46f
Create Date: 2026-07-22 09:49:58.561857

Adds `peer`/`local` JSON columns to contact_interactions and converts the
existing flat `peer_type`/`peer_target`/`local_type`/`local_target` plain
columns into MySQL STORED generated columns derived from `peer`/`local`.
See docs/plans/2026-07-22-case-interaction-peer-local-address-json-design.md
§3.2.

Unlike message.source's precedent (no backfill, nullable JSON), backfill
here is MANDATORY: contact_interactions.peer_type/peer_target are NOT NULL
and carry a live UNIQUE index (idx_contact_interactions_idem) and a live
composite INDEX (idx_contact_interactions_peer). A generated column
computed from a NULL `peer` JSON value evaluates to NULL, which would flip
peer_type/peer_target from NOT NULL to effectively-NULL for every
pre-migration row and break the idempotency unique's invariant.

idx_contact_interactions_idem/idx_contact_interactions_peer must be
dropped BEFORE dropping peer_type/peer_target/local_type/local_target
(round-9 design review finding): dropping a column referenced by a
composite index while the index is still live implicitly shrinks the
index rather than refusing the drop outright, which can silently narrow
the index or fail outright with errno 1062 against pre-existing
duplicate-reference rows using the fan-out capability the 3-column index
shape was built for. Both indexes are restored at their FULL original
column lists afterward, attached to the newly-recreated generated
columns.

Behavioral note: pre-migration rows where local_type/local_target were
both '' (the pre-adb8daac2bb0 historical default) backfill to
`local IS NULL` -> generated local_type/local_target become SQL NULL
instead of ''. This is a documented, intentional behavior change (no
`WHERE local_type = ''` predicate exists in the repo today per the
design's grep audit).
"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'b41d1b2317af'
down_revision = '167bebb7c46f'
branch_labels = None
depends_on = None


def upgrade():
    # peer is added nullable first, then tightened to NOT NULL after backfill --
    # same ADD-nullable -> backfill -> MODIFY-NOT-NULL ordering fix as §3.1
    # (round-1 design review caught the same NOT-NULL-with-no-DEFAULT bug here).
    op.execute("""
        ALTER TABLE contact_interactions
            ADD COLUMN peer  JSON NULL AFTER direction,
            ADD COLUMN local JSON NULL AFTER peer;
    """)

    # MANDATORY backfill (unlike message.source's precedent of "no backfill,
    # nullable JSON, historical rows show NULL"): contact_interactions.peer_type/
    # peer_target are NOT NULL and carry a live UNIQUE index
    # (idx_contact_interactions_idem) and a live composite INDEX
    # (idx_contact_interactions_peer) today. A generated column computed from a
    # NULL `peer` JSON value evaluates to NULL, which would flip
    # peer_type/peer_target from NOT NULL to effectively-NULL for every
    # pre-migration row and break the idempotency unique's invariant (MySQL
    # treats NULL as "distinct" under UNIQUE, so historical rows would silently
    # stop deduplicating against re-delivered events). Backfill peer/local JSON
    # from the existing plain columns in the SAME migration transaction, before
    # the plain columns are dropped and regenerated:
    op.execute("""
        UPDATE contact_interactions
        SET peer  = JSON_OBJECT('type', peer_type, 'target', peer_target),
            local = IF(local_type = '' AND local_target = '', NULL,
                       JSON_OBJECT('type', local_type, 'target', local_target))
        WHERE peer IS NULL;
    """)

    # Now that every row has a non-NULL peer, tighten the column to NOT NULL
    # (mirrors §3.1's contact_cases fix -- this MODIFY cannot fail against the
    # just-backfilled data since every row now has a non-NULL peer).
    op.execute("""
        ALTER TABLE contact_interactions
            MODIFY COLUMN peer JSON NOT NULL;
    """)

    # Drop idx_contact_interactions_idem/idx_contact_interactions_peer BEFORE
    # dropping peer_type/peer_target/local_type/local_target (round-9 design
    # review finding, confirmed by executing this exact sequence against a
    # real MySQL 8.0.46 instance seeded with pre-existing SMS-fan-out-shaped
    # data): dropping peer_target while idx_contact_interactions_idem is
    # still live implicitly shrinks the index to (reference_type,
    # reference_id) as MySQL processes the ALTER, and if any pre-existing
    # rows already share that narrower key -- guaranteed in any real
    # deployment using the fan-out capability this index's 3-column shape
    # was built for (per ac5d4e18060c's own comment: "the bare triple
    # distinguishes SMS fan-out") -- the implicit shrink itself violates the
    # not-yet-fully-dropped unique constraint and the DROP COLUMN statement
    # fails outright with errno 1062 (duplicate entry), not silently. This
    # mirrors §3.1's already-correct index-before-column ordering for
    # open_peer_uk/uq_case_open_peer; §3.2's original ordering had the drop
    # sequence reversed relative to §3.1's pattern, which round 8's test
    # (post-migration inserts only, no pre-existing duplicate-reference rows
    # seeded before running the migration) did not exercise.
    op.execute("""
        ALTER TABLE contact_interactions
            DROP INDEX idx_contact_interactions_idem,
            DROP INDEX idx_contact_interactions_peer;
    """)

    op.execute("""
        ALTER TABLE contact_interactions
            DROP COLUMN peer_type,
            DROP COLUMN peer_target,
            DROP COLUMN local_type,
            DROP COLUMN local_target;
    """)

    op.execute("""
        ALTER TABLE contact_interactions
            ADD COLUMN peer_type VARCHAR(255)
                GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.type'))) STORED NOT NULL
                AFTER peer,
            ADD COLUMN peer_target VARCHAR(255)
                GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.target'))) STORED NOT NULL
                AFTER peer_type,
            ADD COLUMN local_type VARCHAR(255)
                GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(local, '$.type'))) STORED
                AFTER local,
            ADD COLUMN local_target VARCHAR(255)
                GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(local, '$.target'))) STORED
                AFTER local_type;
    """)

    # Restore idx_contact_interactions_idem/idx_contact_interactions_peer to
    # their FULL original column lists (round-8 design review finding):
    # unlike §3.1's contact_cases case, dropping peer_type/peer_target does
    # NOT error on its own -- MySQL silently SHRINKS a composite index when a
    # trailing column it references is dropped, rather than refusing the
    # drop outright. idx_contact_interactions_idem
    # (ac5d4e18060c: UNIQUE INDEX (reference_type, reference_id, peer_target))
    # would silently narrow to (reference_type, reference_id) alone, and
    # idx_contact_interactions_peer
    # (INDEX (customer_id, peer_type, peer_target)) would silently narrow to
    # (customer_id) alone, if the columns were dropped while these indexes
    # were still attached to them -- which is exactly why both indexes are
    # explicitly DROPped above, BEFORE the column drop (round-9 design
    # review finding: dropping the columns first, while the indexes were
    # still live, made the DROP COLUMN statement itself fail outright with
    # errno 1062 against any table already containing legitimate
    # same-reference/different-peer rows, since the implicit index-shrink
    # MySQL performs mid-ALTER collides with pre-existing data using the
    # fan-out capability this index's 3-column shape was built for). Neither
    # problem (silent shrink, or hard failure against real fan-out data) is
    # possible once the indexes are gone before the columns are ever
    # touched. Re-add both indexes now, at their full original column lists,
    # attached to the newly-recreated generated columns:
    op.execute("""
        ALTER TABLE contact_interactions
            ADD UNIQUE INDEX idx_contact_interactions_idem (reference_type, reference_id, peer_target),
            ADD INDEX        idx_contact_interactions_peer (customer_id, peer_type, peer_target);
    """)

    # idx_contact_interactions_cursor (customer_id, tm_create) is untouched:
    # it references neither peer_type/peer_target/local_type/local_target
    # nor any dropped column, so it was never subject to the same-column
    # index-shrinkage risk the two indexes above were.


def downgrade():
    # Recreate peer_type/peer_target/local_type/local_target as plain
    # VARCHAR(255) NOT NULL DEFAULT '' columns, backfill from the
    # (generated) current values before dropping them, then drop peer/local.
    # idx_contact_interactions_idem/idx_contact_interactions_peer must again
    # be dropped and recreated around the column-type change, mirroring
    # upgrade()'s dependency ordering.
    op.execute("""
        ALTER TABLE contact_interactions
            DROP INDEX idx_contact_interactions_idem,
            DROP INDEX idx_contact_interactions_peer;
    """)

    op.execute("""
        ALTER TABLE contact_interactions
            ADD COLUMN peer_type_plain VARCHAR(255) NOT NULL DEFAULT '' AFTER local_target,
            ADD COLUMN peer_target_plain VARCHAR(255) NOT NULL DEFAULT '' AFTER peer_type_plain,
            ADD COLUMN local_type_plain VARCHAR(255) NOT NULL DEFAULT '' AFTER peer_target_plain,
            ADD COLUMN local_target_plain VARCHAR(255) NOT NULL DEFAULT '' AFTER local_type_plain;
    """)

    op.execute("""
        UPDATE contact_interactions
        SET peer_type_plain = peer_type,
            peer_target_plain = peer_target,
            local_type_plain = IFNULL(local_type, ''),
            local_target_plain = IFNULL(local_target, '');
    """)

    op.execute("""
        ALTER TABLE contact_interactions
            DROP COLUMN peer_type,
            DROP COLUMN peer_target,
            DROP COLUMN local_type,
            DROP COLUMN local_target,
            DROP COLUMN peer,
            DROP COLUMN local;
    """)

    op.execute("""
        ALTER TABLE contact_interactions
            CHANGE COLUMN peer_type_plain peer_type VARCHAR(255) NOT NULL DEFAULT '',
            CHANGE COLUMN peer_target_plain peer_target VARCHAR(255) NOT NULL DEFAULT '',
            CHANGE COLUMN local_type_plain local_type VARCHAR(255) NOT NULL DEFAULT '',
            CHANGE COLUMN local_target_plain local_target VARCHAR(255) NOT NULL DEFAULT '';
    """)

    op.execute("""
        ALTER TABLE contact_interactions
            ADD UNIQUE INDEX idx_contact_interactions_idem (reference_type, reference_id, peer_target),
            ADD INDEX        idx_contact_interactions_peer (customer_id, peer_type, peer_target);
    """)
