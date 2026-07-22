"""contact_cases_peer_local_address_json

Revision ID: 167bebb7c46f
Revises: a1dca1dc6e67
Create Date: 2026-07-22 09:49:55.518225

Adds `peer`/`local` JSON columns to contact_cases and converts the existing
flat `peer_type`/`peer_target` plain columns into MySQL STORED generated
columns derived from `peer`, and adds new `local_type`/`local_target`
generated columns derived from `local`. See docs/plans/
2026-07-22-case-interaction-peer-local-address-json-design.md §3.1.

`open_peer_uk`/`uq_case_open_peer` must be dropped and recreated around the
peer_type/peer_target drop-and-regenerate step because open_peer_uk is
itself a STORED generated column whose expression directly references
peer_type/peer_target -- MySQL refuses to DROP a column that another
generated column depends on (errno 3108).
"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '167bebb7c46f'
down_revision = 'a1dca1dc6e67'
branch_labels = None
depends_on = None


def upgrade():
    # peer is added nullable first, then tightened to NOT NULL after backfill.
    # Adding a NOT NULL column with no DEFAULT to an already-populated table
    # fails in MySQL strict mode (errno 1364: "Field 'peer' doesn't have a
    # default value"), and even where it didn't fail outright it would leave
    # no row with peer IS NULL for the backfill UPDATE below to match -- the
    # ADD-nullable -> backfill -> MODIFY-NOT-NULL ordering is required, not
    # optional (round-1 design review caught this as a real correctness bug
    # in an earlier draft that added `peer JSON NOT NULL` directly).
    op.execute("""
        ALTER TABLE contact_cases
            ADD COLUMN peer  JSON NULL AFTER customer_id,
            ADD COLUMN local JSON NULL AFTER peer;
    """)

    # local stays nullable permanently (see the note below); peer is only
    # transiently nullable during this migration, tightened to NOT NULL
    # after the backfill step.
    # local is nullable: GetOrCreate's `self` parameter can be a zero
    # commonaddress.Address (design VOIP-1243's proactive-link check already
    # treats `self.Type == ""` as "no local endpoint known", see
    # getorcreate.go:99). A zero Address is still valid JSON ('{}' after
    # omitempty strips every field), but modeling "no local known" as SQL NULL
    # is more honest than a JSON value with every field empty, and lets
    # local_type/local_target's generated expressions below compute NULL
    # (matching the pre-existing convention that empty-string columns, not
    # NULL, meant "the value that was stored," which for a never-before-existing
    # column becomes moot -- NULL is now the correct spelling of "not captured").

    # Backfill: no pre-existing rows carry peer/local JSON (this is a brand new
    # pair of columns; the flat peer_type/peer_target values below are used to
    # backfill `peer` so peer_type/peer_target's generated expressions keep
    # producing identical values for existing rows -- open_peer_uk and
    # uq_case_open_peer depend on this).
    op.execute("""
        UPDATE contact_cases
        SET peer = JSON_OBJECT('type', peer_type, 'target', peer_target)
        WHERE peer IS NULL;
    """)

    # Now that every row has a non-NULL peer, tighten the column to NOT NULL
    # (every Case has a peer by construction -- Create/GetOrCreate both
    # require peerType/peerTarget as non-optional arguments -- so this MODIFY
    # cannot fail against the just-backfilled data).
    op.execute("""
        ALTER TABLE contact_cases
            MODIFY COLUMN peer JSON NOT NULL;
    """)

    # Drop the old plain columns and re-add them as STORED generated columns
    # derived from `peer`. Column order/type/NOT NULL/DEFAULT are unchanged so
    # every existing index and query keeps working byte-for-byte -- EXCEPT
    # for open_peer_uk/uq_case_open_peer, which MUST be dropped and
    # recreated around this step (round-8 design review finding, confirmed
    # by executing this exact DDL against a real MySQL 8.0.46 instance built
    # with the actual open_peer_uk column present): open_peer_uk is itself a
    # STORED generated column (f718e26f2c44) whose expression directly
    # references peer_type/peer_target
    # (`UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target,
    # reference_type), 256))`). MySQL refuses to DROP a column that another
    # generated column depends on (errno 3108: "Column 'peer_type' has a
    # generated column dependency") -- the migration cannot proceed past the
    # next statement without first removing that dependency.
    op.execute("""
        ALTER TABLE contact_cases
            DROP INDEX uq_case_open_peer,
            DROP COLUMN open_peer_uk;
    """)

    op.execute("""
        ALTER TABLE contact_cases
            DROP COLUMN peer_type,
            DROP COLUMN peer_target;
    """)

    op.execute("""
        ALTER TABLE contact_cases
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

    # Recreate open_peer_uk/uq_case_open_peer now that peer_type/peer_target
    # exist again (as generated columns). The expression is copied verbatim
    # from f718e26f2c44 -- unchanged behavior, just re-attached to the new
    # generated peer_type/peer_target instead of the old plain columns.
    op.execute("""
        ALTER TABLE contact_cases
            ADD COLUMN open_peer_uk BINARY(32) GENERATED ALWAYS AS (
                IF(status = 'open',
                   UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target, reference_type), 256)),
                   NULL)
            ) STORED AFTER local_target,
            ADD UNIQUE INDEX uq_case_open_peer (open_peer_uk);
    """)

    # idx_case_customer_reftype, idx_case_unresolved, idx_case_owner are
    # untouched: none of them reference peer_type/peer_target/open_peer_uk,
    # so they were never affected by the drop-and-recreate sequence above.


def downgrade():
    # Recreate peer_type/peer_target as plain VARCHAR(255) NOT NULL DEFAULT
    # '' columns, backfill from the (generated) peer_type/peer_target
    # current values before dropping them, then drop local_type/
    # local_target/peer/local. open_peer_uk/uq_case_open_peer must again be
    # dropped and recreated around the column-type change, mirroring
    # upgrade()'s dependency ordering.
    op.execute("""
        ALTER TABLE contact_cases
            DROP INDEX uq_case_open_peer,
            DROP COLUMN open_peer_uk;
    """)

    op.execute("""
        ALTER TABLE contact_cases
            ADD COLUMN peer_type_plain VARCHAR(255) NOT NULL DEFAULT '' AFTER peer_target,
            ADD COLUMN peer_target_plain VARCHAR(255) NOT NULL DEFAULT '' AFTER peer_type_plain;
    """)

    op.execute("""
        UPDATE contact_cases
        SET peer_type_plain = peer_type,
            peer_target_plain = peer_target;
    """)

    op.execute("""
        ALTER TABLE contact_cases
            DROP COLUMN peer_type,
            DROP COLUMN peer_target,
            DROP COLUMN local_type,
            DROP COLUMN local_target,
            DROP COLUMN peer,
            DROP COLUMN local;
    """)

    op.execute("""
        ALTER TABLE contact_cases
            CHANGE COLUMN peer_type_plain peer_type VARCHAR(255) NOT NULL DEFAULT '',
            CHANGE COLUMN peer_target_plain peer_target VARCHAR(255) NOT NULL DEFAULT '';
    """)

    op.execute("""
        ALTER TABLE contact_cases
            ADD COLUMN open_peer_uk BINARY(32) GENERATED ALWAYS AS (
                IF(status = 'open',
                   UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target, reference_type), 256)),
                   NULL)
            ) STORED,
            ADD UNIQUE INDEX uq_case_open_peer (open_peer_uk);
    """)
