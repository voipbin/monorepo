# VOIP-1545 PR-C: add upstream Asterisk RTDB migration 2285f2ace275

Date: 2026-09-30. Ticket: VOIP-1545. Branch: `VOIP-1545-Add-asterisk-rtdb-migration-2285f2ace275`.
Issue analysis: covered by the approved VOIP-1545 analysis (RTDB schema delta row, §2 item 3 PR-C, migration-first ordering), closed with 2 consecutive approvals. This document adds the PR-C specifics.
Status: design review. Round 1: A CHANGES_REQUESTED (1: engine basis), B APPROVED; both engines now required (production MariaDB 12.3, install MySQL 8.0), non-blocking notes applied. Round 2: APPROVED (non-blocking: full PIP_PINS, operator-held alembic.ini; applied). Round 3: APPROVED (non-blocking: final-newline wording, run from pulled main; applied). Design review loop finished. PR rounds 1-3 APPROVED (round-3 non-blocking notes applied: INSTANT fallback wording, optional metadata-lock check).

## 1. Background

The Asterisk RealTime DB (`asterisk` database) schema is managed by the `bin-dbscheme-manager/asterisk_config` Alembic stream, kept at parity with upstream `contrib/ast-db-manage/config`. Precedent: monorepo #1001 (NOJIRA-upgrade-asterisk-to-23.4.0, 1821e724a) copied upstream `e89e30cee53f` verbatim (byte-identical to upstream 23.5.0's copy of that file, verified with diff) as the documented exception to the no-hand-written-migration rule, so the upstream revision ID and chain are preserved.

`git diff --stat 23.4.0 23.5.0 -- contrib/ast-db-manage` (upstream asterisk repo) shows exactly one change: new file `config/versions/2285f2ace275_add_external_signaling_hostname.py` (20 lines):

- `revision = '2285f2ace275'`, `down_revision = 'e89e30cee53f'` (our current single head of the stream, 130 files).
- `upgrade()`: `op.add_column('ps_transports', sa.Column('external_signaling_hostname', sa.String(40)))` (nullable, no default).
- `downgrade()`: `op.drop_column('ps_transports', 'external_signaling_hostname')`.
- Upstream ships the file with CRLF line endings (19 CR characters) and no trailing newline. All 130 files in our stream are LF.

Runtime relevance: none of our 3 Asterisk images loads transports from realtime. The registrar's `sorcery.conf`/`extconfig.conf` map only endpoint, auth, aor, domain_alias, contact and identify to realtime tables. call and conference ship no `sorcery.conf`/`extconfig.conf` at all, and the images do not install the Asterisk samples (`make install` only, then `COPY etc/asterisk`), so sorcery uses its default (config file). In all 3 images transports are static `[transport-*]` sections in `pjsip.conf`. So the new column is never read by our Asterisk; the migration is for schema parity with upstream 23.5.0 (so a future realtime transport, or any tooling comparing against upstream, sees the expected schema). No monorepo code references `ps_transports` outside the migration files.

## 2. Goal

Add `2285f2ace275` to `bin-dbscheme-manager/asterisk_config/config/versions/`, keeping upstream's revision ID and chain, so the stream has a single head `2285f2ace275`, and have it applied to production before the Asterisk 23.5.0 images (PR-D) roll out.

## 3. Non-goals

- No change to the `bin-manager` (voipbin DB) stream.
- No Asterisk config change (transports stay static in `pjsip.conf`); no use of `external_signaling_hostname`.
- The `voipbin/voipbin` `install/` dbscheme pin (`versions.lock`) is not bumped here; it is part of the install follow-up tracked for after PR-D.
- No CI change.

## 4. Change

One new file, content identical to upstream except line endings:

- `bin-dbscheme-manager/asterisk_config/config/versions/2285f2ace275_add_external_signaling_hostname.py`
- Obtained with `git show 23.5.0:contrib/ast-db-manage/config/versions/2285f2ace275_add_external_signaling_hostname.py | tr -d '\r'` plus a final newline (LF-normalized like all 130 files, and with a final newline like most of them; 3 existing files, `417c0247fd7e`, `7f85dd44c775`, `9f3692b1654b`, happen to lack one; the revision identifiers, `down_revision` and the upgrade/downgrade bodies are unchanged).
- Not generated with `alembic revision`: the documented exception for upstream RTDB parity (same as #1001), because the upstream revision ID must be preserved for parity with Asterisk's own tooling.

Plus this design document. No docs change is needed in `bin-dbscheme-manager/docs/`: `schema-ownership.md` already states that the Asterisk schema is sourced from `asterisk/contrib/ast-db-manage` and that new version files are copied from the Asterisk repo on upgrade (this PR follows exactly that), and its Asterisk table list names tables, not columns (`ps_transports` is already covered).

## 5. Verification before PR

- Content check: `diff <(git -C <asterisk clone> show 23.5.0:<path> | tr -d '\r') <new file>` shows only the added final newline; `grep -c $'\r'` on the new file is 0.
- Chain check (metadata-only, same as CI `migration-lint`): in `asterisk_config`, `alembic -c alembic.ini heads` prints exactly one head `2285f2ace275 (head)`; `alembic history` resolves the full chain.
- Real apply on local throwaway DBs only (never a shared DB), on both engines that run this stream:
  - `mariadb:12.3`: the production `asterisk` database engine (monorepo-etc `infra-database`, MariaDB 12.3 since the VOIP-1386 cutover).
  - `mysql:8.0`: the `voipbin/voipbin` install path (`install/docker-compose.yml.dist` `db: image: mysql:8.0`).
  Same conditions as the install path: database created with `CHARACTER SET utf8 COLLATE utf8_general_ci`, `python:3.11-slim`, the full `migrate.sh` `PIP_PINS` string as is (10 pins including alembic 1.11.3, SQLAlchemy 1.4.52, PyMySQL 1.1.0 and cryptography, which MySQL 8 auth needs; installed with `--no-deps`), PyMySQL DSN. On each engine: `alembic upgrade head` of the full 131-revision chain, `SHOW COLUMNS FROM ps_transports LIKE 'external_signaling_hostname'` shows `varchar(40)`, nullable, default NULL; `alembic downgrade -1` leaves `alembic_version` at `e89e30cee53f`; `upgrade head` again succeeds; `alembic upgrade e89e30cee53f:2285f2ace275 --sql` shows the single `ALTER TABLE ... ADD COLUMN` plus the version update. Containers removed afterwards.
- No em/en dashes in added lines.

## 6. Rollout (analysis §2 item 3)

1. PR review loop; CEO merge instruction (squash).
2. Main pipeline: `migration-lint` runs automatically; `migration-applied-checkpoint` (main-only approval) is an attestation only and applies nothing.
3. The CEO applies the migration to the production `asterisk` database over VPN, from the base monorepo checkout on `main` after `git pull` (so the merged file is present): in `bin-dbscheme-manager/asterisk_config` with the operator's local production `alembic.ini` (the repo only has `alembic.ini.sample`; the real file with the production `asterisk` DB URL is kept locally by the operator and never committed):
   - `alembic -c alembic.ini current`: must print `e89e30cee53f`. If it prints anything else, stop and report instead of upgrading.
   - Optional sanity check: `SELECT COUNT(*) FROM ps_transports` (expected 0 or very small, since transports are static config).
   - Optional lock check: `SELECT ID, TIME, STATE, INFO FROM information_schema.PROCESSLIST WHERE INFO LIKE '%ps_transports%' AND ID <> CONNECTION_ID()` should return nothing. The ALTER needs a brief exclusive metadata lock on `ps_transports`; nothing in our stack reads that table, but an unexpected long-running session holding it would make the ALTER wait (up to `lock_wait_timeout`).
   - `alembic -c alembic.ini upgrade head`, then `current` again: must print `2285f2ace275 (head)`.
   Hermes does not run this. There is no separate staging `asterisk` database today, so `docs/operations.md`'s "staging first" step has no target here; the throwaway round-trips in §5 on the production engine version stand in for it, as for the 23.4.0 precedent.
4. The CEO approves `migration-applied-checkpoint` on the main pipeline as the attestation.
5. Only after that, PR-D (Asterisk 23.5.0) is released.

Impact of the apply: `ALTER TABLE ps_transports ADD COLUMN external_signaling_hostname VARCHAR(40) NULL` on a table that is expected to be empty or tiny (transports are static config here); MariaDB 12.3 and MySQL 8.0 both add a trailing nullable column with the INSTANT algorithm (verified on a throwaway MariaDB 12.3 copy, also under concurrent ps_endpoints/ps_aors activity); should INSTANT not apply on the production table (for example an old row format), the table is empty or tiny so INPLACE/COPY also finishes immediately; no data change, nothing reads it. Safe to apply while the current 23.4.0 containers run (23.4.0 does not know the column and does not use realtime transports).

## 7. Rollback

`alembic -c alembic.ini downgrade e89e30cee53f` over VPN (drops only the new, unused column). Only needed if the column causes a problem; since nothing reads it, leaving it in place is also safe even if PR-D is rolled back to 23.4.0.

## 8. Risks

- Hand-copied file instead of `alembic revision`: mitigated by the upstream-parity precedent and the single-head check; the revision ID `2285f2ace275` does not collide with any existing file (checked by `alembic heads/history`).
- CRLF normalization changes bytes vs upstream: semantic content identical; verified by the diff in §5.
- Apply ordering: if PR-D shipped before the apply, 23.5.0 would still work (transports are not realtime), but the schema would lag upstream; the rollout keeps migration-first anyway.
