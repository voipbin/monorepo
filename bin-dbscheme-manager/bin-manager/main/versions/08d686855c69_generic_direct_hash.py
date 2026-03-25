"""generic_direct_hash

Revision ID: 08d686855c69
Revises: 3783e1841f4b
Create Date: 2026-03-25 08:33:14.224324

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '08d686855c69'
down_revision = '3783e1841f4b'
branch_labels = None
depends_on = None


def upgrade():
    # Step 1: Create direct_directs table
    op.execute("""
        CREATE TABLE direct_directs (
            id            binary(16) NOT NULL,
            customer_id   binary(16) NOT NULL,
            resource_type varchar(32) NOT NULL,
            resource_id   binary(16) NOT NULL,
            hash          varchar(255) NOT NULL,
            tm_create     datetime(6),
            tm_update     datetime(6),

            PRIMARY KEY(id),
            UNIQUE INDEX idx_direct_directs_hash (hash),
            UNIQUE INDEX idx_direct_directs_resource (resource_type, resource_id),
            INDEX idx_direct_directs_customer_id (customer_id)
        )
    """)

    # Step 2: Alter resource tables — add direct_id and direct_hash
    for table in ['registrar_extensions', 'conference_conferences', 'ai_ais', 'ai_teams', 'agent_agents']:
        op.execute(f"ALTER TABLE {table} ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255)")

    # Step 3: Migrate existing registrar_directs data
    # Existing hash values are raw hex — store as-is (no prefix)
    op.execute("""
        INSERT INTO direct_directs (id, customer_id, resource_type, resource_id, hash, tm_create, tm_update)
        SELECT id, customer_id, 'extension', extension_id, hash, tm_create, tm_update
        FROM registrar_directs
        WHERE tm_delete IS NULL
    """)

    # Step 4: Populate direct_id and direct_hash on registrar_extensions
    op.execute("""
        UPDATE registrar_extensions e
        INNER JOIN direct_directs d ON d.resource_id = e.id AND d.resource_type = 'extension'
        SET e.direct_id = d.id, e.direct_hash = d.hash
    """)

    # Step 5: Drop old table
    op.execute("DROP TABLE registrar_directs")


def downgrade():
    # Recreate registrar_directs
    op.execute("""
        CREATE TABLE registrar_directs (
            id            binary(16),
            customer_id   binary(16),
            extension_id  binary(16),
            hash          varchar(255),
            tm_create     datetime(6),
            tm_update     datetime(6),
            tm_delete     datetime(6),
            PRIMARY KEY(id),
            UNIQUE INDEX idx_registrar_directs_extension_id (extension_id),
            UNIQUE INDEX idx_registrar_directs_hash (hash),
            INDEX idx_registrar_directs_customer_id (customer_id)
        )
    """)

    # Migrate data back
    op.execute("""
        INSERT INTO registrar_directs (id, customer_id, extension_id, hash, tm_create, tm_update)
        SELECT id, customer_id, resource_id, hash, tm_create, tm_update
        FROM direct_directs
        WHERE resource_type = 'extension'
    """)

    # Remove direct columns from resource tables
    for table in ['registrar_extensions', 'conference_conferences', 'ai_ais', 'ai_teams', 'agent_agents']:
        op.execute(f"ALTER TABLE {table} DROP COLUMN direct_id, DROP COLUMN direct_hash")

    # Drop direct_directs
    op.execute("DROP TABLE direct_directs")
