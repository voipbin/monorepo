"""contact create tables

Revision ID: a1b2c3d4e5f6
Revises: 1ef85e7392ce
Create Date: 2026-02-03 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f6'
down_revision = '1ef85e7392ce'
branch_labels = None
depends_on = None


def upgrade():
    # contact_contacts - Main contacts table
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_contacts (
            id BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,
            first_name VARCHAR(100) DEFAULT '',
            last_name VARCHAR(100) DEFAULT '',
            display_name VARCHAR(200) DEFAULT '',
            company VARCHAR(200) DEFAULT '',
            job_title VARCHAR(100) DEFAULT '',
            source VARCHAR(50) DEFAULT 'manual',
            external_id VARCHAR(255) DEFAULT '',
            notes TEXT,
            total_calls BIGINT DEFAULT 0,
            total_messages BIGINT DEFAULT 0,
            last_contact_at DATETIME(6) DEFAULT '9999-01-01 00:00:00.000000',
            tm_create DATETIME(6) NOT NULL,
            tm_update DATETIME(6) NOT NULL DEFAULT '9999-01-01 00:00:00.000000',
            tm_delete DATETIME(6) NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

            PRIMARY KEY (id),
            INDEX idx_contact_contacts_customer (customer_id),
            INDEX idx_contact_contacts_customer_name (customer_id, display_name),
            INDEX idx_contact_contacts_customer_external (customer_id, external_id),
            INDEX idx_contact_contacts_tm_create (tm_create),
            FULLTEXT idx_contact_contacts_search (first_name, last_name, display_name, company)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)

    # contact_phone_numbers - Phone numbers for contacts
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_phone_numbers (
            id BINARY(16) NOT NULL,
            contact_id BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,
            number VARCHAR(30) NOT NULL,
            number_e164 VARCHAR(20) NOT NULL,
            type VARCHAR(20) DEFAULT 'mobile',
            is_primary TINYINT(1) DEFAULT 0,
            label VARCHAR(50) DEFAULT '',
            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (id),
            INDEX idx_contact_phone_numbers_contact (contact_id),
            INDEX idx_contact_phone_numbers_customer_number (customer_id, number_e164),
            UNIQUE INDEX idx_contact_phone_numbers_unique (customer_id, number_e164)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)

    # contact_emails - Email addresses for contacts
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_emails (
            id BINARY(16) NOT NULL,
            contact_id BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,
            address VARCHAR(255) NOT NULL,
            type VARCHAR(20) DEFAULT 'work',
            is_primary TINYINT(1) DEFAULT 0,
            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (id),
            INDEX idx_contact_emails_contact (contact_id),
            INDEX idx_contact_emails_customer_address (customer_id, address)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)

    # contact_lists - Contact groups/segments
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_lists (
            id BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,
            name VARCHAR(100) NOT NULL,
            detail TEXT,
            type VARCHAR(20) DEFAULT 'static',
            query TEXT,
            contact_count BIGINT DEFAULT 0,
            tm_create DATETIME(6) NOT NULL,
            tm_update DATETIME(6) NOT NULL DEFAULT '9999-01-01 00:00:00.000000',
            tm_delete DATETIME(6) NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

            PRIMARY KEY (id),
            INDEX idx_contact_lists_customer (customer_id),
            INDEX idx_contact_lists_tm_create (tm_create)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)

    # contact_list_members - Many-to-many relationship
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_list_members (
            list_id BINARY(16) NOT NULL,
            contact_id BINARY(16) NOT NULL,
            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (list_id, contact_id),
            INDEX idx_contact_list_members_contact (contact_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)

    # contact_activities - Communication history
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_activities (
            id BINARY(16) NOT NULL,
            contact_id BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,
            type VARCHAR(20) NOT NULL,
            direction VARCHAR(20) NOT NULL,
            reference_id BINARY(16),
            summary VARCHAR(500) DEFAULT '',
            duration BIGINT DEFAULT 0,
            tm_activity DATETIME(6) NOT NULL,
            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (id),
            INDEX idx_contact_activities_contact_time (contact_id, tm_activity DESC),
            INDEX idx_contact_activities_customer_time (customer_id, tm_activity DESC),
            INDEX idx_contact_activities_reference (reference_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)

    # contact_tag_assignments - Many-to-many with existing tags table
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_tag_assignments (
            contact_id BINARY(16) NOT NULL,
            tag_id BINARY(16) NOT NULL,
            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (contact_id, tag_id),
            INDEX idx_contact_tag_assignments_tag (tag_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS contact_tag_assignments;""")
    op.execute("""DROP TABLE IF EXISTS contact_activities;""")
    op.execute("""DROP TABLE IF EXISTS contact_list_members;""")
    op.execute("""DROP TABLE IF EXISTS contact_lists;""")
    op.execute("""DROP TABLE IF EXISTS contact_emails;""")
    op.execute("""DROP TABLE IF EXISTS contact_phone_numbers;""")
    op.execute("""DROP TABLE IF EXISTS contact_contacts;""")
