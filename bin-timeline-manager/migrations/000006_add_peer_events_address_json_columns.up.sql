-- peer/local now carry the full commonaddress.Address (Type/Target/TargetName/
-- Name/Detail) as JSON text, matching contact_interactions' pattern
-- (JSON + generated-column split, see
-- bin-contact-manager/scripts/database_scripts_test/contacts.sql). ClickHouse
-- has no STORED GENERATED COLUMN equivalent, so peer_type/peer_target/
-- local_type/local_target remain physical columns populated by the
-- application at insert time (buildPeerEventRows), NOT derived by
-- ClickHouse -- they stay INTERNAL to this table (ORDER BY index +
-- WHERE search only) and are never exposed by the read API response.
ALTER TABLE peer_events
    ADD COLUMN peer  String AFTER direction,
    ADD COLUMN local String AFTER peer;
