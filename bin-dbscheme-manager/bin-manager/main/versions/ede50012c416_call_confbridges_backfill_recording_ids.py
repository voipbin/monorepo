"""call_confbridges_backfill_recording_ids

Revision ID: ede50012c416
Revises: 902325885953
Create Date: 2026-08-10 20:00:03.914006

VOIP-1305 data backfill. Design document,
docs/plans/2026-08-10-voip-1305-confbridge-recording-ids-fix-and-backfill-design.md

Before VOIP-1305, bin-call-manager ConfbridgeAddRecordingIDs bound
recordingID.Bytes() (16 raw bytes) instead of recordingID.String() (36 char
UUID text) into json_array_append, so call_confbridges.recording_ids collected
corrupted elements that the Go read path ([]uuid.UUID unmarshal) cannot parse.
Depending on the go-sql-driver protocol path there are two corruption forms:

  Form A (prepared statement path, driver default): the raw 16 bytes stored as
    a JSON STRING element. Detected per element with
    JSON_TYPE(elem) = 'STRING' AND LENGTH(JSON_UNQUOTE(elem)) = 16
    (byte length; a healthy UUID string is 36 bytes).
  Form B (interpolateParams text protocol or a _binary literal): an opaque JSON
    BLOB element that MySQL renders as base64 with a type tag. Detected with
    JSON_TYPE(elem) = 'BLOB'.

This migration restores both forms to the lowercase 36 char UUID text form,
element by element with JSON_REPLACE (array order preserved, healthy elements
untouched). All detection and conversion happens in server side SQL. Only
BINARY(16) row ids and integer counts ever reach the Python client, so the
corrupted bytes never cross the driver boundary (avoids UTF-8 decode issues).

Idempotent and re-run safe. Repaired elements no longer match any detection
condition, so a re-run collects nothing new and updates nothing. Elements that
match detection but cannot be converted (for example a BLOB whose payload does
not decode to exactly 16 bytes) are preserved as is and reported in
remaining_anomaly_rows for manual investigation.

Constraint. The always executed Phase 0 path must stay MariaDB compatible,
because the dbscheme Docker image build runs the whole migration chain against
a temporary MariaDB with empty tables. Phase 0 uses only COALESCE, MAX and
JSON_LENGTH, and exits early on the empty table, so later MySQL specific
behavior is never reached in that environment.

SQL text in this file deliberately contains no colon characters. op.execute
and text() treat a colon prefixed identifier as a bind parameter, which has
bitten this repository before. The Form B separator is produced with CHAR(58)
and the id lists are inlined as UNHEX hex literals (hex digits only).
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ede50012c416'
down_revision = '902325885953'
branch_labels = None
depends_on = None


TABLE = 'call_confbridges'
CHUNK_SIZE = 500


def _element_corrupted_predicate(i):
    """SQL predicate matching a corrupted element at array index i.

    Form B first (opaque BLOB), then Form A (16 byte STRING). LENGTH counts
    bytes, so a healthy 36 char UUID string never matches.
    """
    elem = "JSON_EXTRACT(recording_ids, '$[{i}]')".format(i=i)
    return (
        "( JSON_TYPE({elem}) = 'BLOB'"
        " OR (JSON_TYPE({elem}) = 'STRING'"
        " AND LENGTH(JSON_UNQUOTE({elem})) = 16) )"
    ).format(elem=elem)


def _in_list(hex_ids):
    """Inline IN list of UNHEX literals for a chunk of id hex strings.

    The hex strings come from bytes.hex() and contain only 0-9a-f, so literal
    inlining is safe and avoids text() bind parameters entirely.
    """
    return ', '.join("UNHEX('" + h + "')" for h in hex_ids)


def _chunks(items, size):
    for pos in range(0, len(items), size):
        yield items[pos:pos + size]


def _collect_corrupted_ids(conn, max_len, restrict_hex_ids=None):
    """Collect the hex ids of rows holding at least one corrupted element.

    Plain SELECTs only. Under REPEATABLE READ a plain SELECT is a lock free
    consistent read, unlike INSERT ... SELECT which takes shared next key locks
    on every scanned source row (measured; that approach was rejected in design
    review). When restrict_hex_ids is given (Phase 3 re-check), the scan is
    limited to those rows in CHUNK_SIZE batches.
    """
    found = set()
    for i in range(max_len):
        base = (
            'SELECT id FROM ' + TABLE +
            ' WHERE recording_ids IS NOT NULL'
            ' AND JSON_LENGTH(recording_ids) > {i}'
            ' AND {pred}'
        ).format(i=i, pred=_element_corrupted_predicate(i))

        if restrict_hex_ids is None:
            rows = conn.execute(sa.text(base)).fetchall()
            found.update(bytes(row[0]).hex() for row in rows)
        else:
            for chunk in _chunks(sorted(restrict_hex_ids), CHUNK_SIZE):
                sql = base + ' AND id IN (' + _in_list(chunk) + ')'
                rows = conn.execute(sa.text(sql)).fetchall()
                found.update(bytes(row[0]).hex() for row in rows)
    return found


def upgrade():
    conn = op.get_bind()

    # Phase 0. Scalar upper bound for the per index loops. Returns a number
    # only. Must stay MariaDB compatible (see module docstring). Caveat, a row
    # whose recording_ids is the JSON null scalar has JSON_LENGTH = 1, so
    # max_len can be >= 1 with zero corrupted rows. That is harmless, the
    # loops below simply match nothing for such rows.
    max_len = conn.execute(sa.text(
        'SELECT COALESCE(MAX(JSON_LENGTH(recording_ids)), 0) FROM ' + TABLE
    )).scalar()
    max_len = int(max_len or 0)

    if max_len == 0:
        print('VOIP-1305 backfill: candidate_rows=0, repaired_rows=0, '
              'remaining_anomaly_rows=0 (no non empty recording_ids)')
        return

    # Phase 1. Collect the ids of corrupted rows (bytes to hex set).
    corrupted_ids = _collect_corrupted_ids(conn, max_len)
    if not corrupted_ids:
        print('VOIP-1305 backfill: candidate_rows=0, repaired_rows=0, '
              'remaining_anomaly_rows=0 (no corrupted elements found)')
        return

    # Phase 2. Repair with primary key targeted UPDATEs so exclusive locks
    # touch only the corrupted rows. UUID text restore expression, HEX of the
    # 16 raw bytes reformatted to 8-4-4-4-12 lowercase. Form A feeds HEX the
    # unquoted string bytes. Form B feeds it the FROM_BASE64 decoded payload
    # after the last CHAR(58) separator of the opaque value rendering, and
    # only when that payload decodes to exactly 16 bytes.
    for chunk in _chunks(sorted(corrupted_ids), CHUNK_SIZE):
        in_list = _in_list(chunk)
        for i in range(max_len):
            elem = "JSON_EXTRACT(recording_ids, '$[{i}]')".format(i=i)
            unq = 'JSON_UNQUOTE(' + elem + ')'
            b64 = 'FROM_BASE64(SUBSTRING_INDEX(' + unq + ', CHAR(58), -1))'

            form_a = (
                'UPDATE ' + TABLE + ' SET recording_ids = '
                "JSON_REPLACE(recording_ids, '$[{i}]', "
                'LOWER(INSERT(INSERT(INSERT(INSERT('
                'HEX({unq})'
                ", 9, 0, '-'), 14, 0, '-'), 19, 0, '-'), 24, 0, '-')))"
                ' WHERE id IN ({ids})'
                " AND JSON_TYPE({elem}) = 'STRING'"
                ' AND LENGTH({unq}) = 16'
            ).format(i=i, unq=unq, elem=elem, ids=in_list)
            conn.execute(sa.text(form_a))

            form_b = (
                'UPDATE ' + TABLE + ' SET recording_ids = '
                "JSON_REPLACE(recording_ids, '$[{i}]', "
                'LOWER(INSERT(INSERT(INSERT(INSERT('
                'HEX({b64})'
                ", 9, 0, '-'), 14, 0, '-'), 19, 0, '-'), 24, 0, '-')))"
                ' WHERE id IN ({ids})'
                " AND JSON_TYPE({elem}) = 'BLOB'"
                ' AND LENGTH({b64}) = 16'
            ).format(i=i, b64=b64, elem=elem, ids=in_list)
            conn.execute(sa.text(form_b))

    # Phase 3. Verify and report, id set based so a row with several corrupted
    # elements counts once. Only the Phase 1 candidates are re-checked.
    remaining_ids = _collect_corrupted_ids(
        conn, max_len, restrict_hex_ids=corrupted_ids)

    candidates = len(corrupted_ids)
    remaining = len(remaining_ids)
    repaired = candidates - remaining
    print('VOIP-1305 backfill: candidate_rows={c}, repaired_rows={r}, '
          'remaining_anomaly_rows={a}'.format(c=candidates, r=repaired,
                                              a=remaining))
    if remaining > 0:
        print('VOIP-1305 backfill WARNING: {a} row(s) still hold elements '
              'that match corruption detection but could not be converted. '
              'They were preserved unchanged, manual investigation '
              'required.'.format(a=remaining))


def downgrade():
    # No downgrade. Reversing a data repair would mean re-corrupting the
    # restored UUID strings, which is never desirable. Precedent for a no-op
    # downgrade, 9443d2d65ad8_add_join_bridge.py.
    pass
