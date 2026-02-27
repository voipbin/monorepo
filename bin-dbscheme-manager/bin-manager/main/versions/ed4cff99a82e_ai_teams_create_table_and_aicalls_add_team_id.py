"""ai_teams create table and ai_aicalls add assistance_type/assistance_id

Revision ID: ed4cff99a82e
Revises: 1486701fb1f1
Create Date: 2026-02-27 01:30:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ed4cff99a82e'
down_revision = '1486701fb1f1'
branch_labels = None
depends_on = None


def upgrade():
    # Create ai_teams table
    op.execute("""
        CREATE TABLE IF NOT EXISTS ai_teams (
            id              BINARY(16) PRIMARY KEY,
            customer_id     BINARY(16) NOT NULL,
            name            VARCHAR(255) NOT NULL DEFAULT '',
            detail          TEXT,
            start_member_id BINARY(16),
            members         JSON,
            tm_create       DATETIME(6),
            tm_update       DATETIME(6),
            tm_delete       DATETIME(6),
            INDEX idx_ai_teams_tm_create (tm_create),
            INDEX idx_ai_teams_customer_id (customer_id)
        );
    """)

    # Add assistance_type and assistance_id columns
    op.execute("""ALTER TABLE ai_aicalls ADD assistance_type VARCHAR(255) DEFAULT '' AFTER ai_id;""")
    op.execute("""ALTER TABLE ai_aicalls ADD assistance_id BINARY(16) AFTER assistance_type;""")

    # Migrate existing data: ai_id -> assistance_type='ai', assistance_id=ai_id
    op.execute("""UPDATE ai_aicalls SET assistance_type='ai', assistance_id=ai_id WHERE ai_id IS NOT NULL;""")

    # Drop old ai_id column and its index
    op.execute("""DROP INDEX idx_ai_aicalls_ai_id ON ai_aicalls;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_id;""")

    # Create indexes on new columns
    op.execute("""CREATE INDEX idx_ai_aicalls_assistance_type ON ai_aicalls(assistance_type);""")
    op.execute("""CREATE INDEX idx_ai_aicalls_assistance_id ON ai_aicalls(assistance_id);""")


def downgrade():
    # Reverse: add ai_id back, migrate data, drop new columns
    op.execute("""ALTER TABLE ai_aicalls ADD ai_id BINARY(16) AFTER customer_id;""")
    op.execute("""UPDATE ai_aicalls SET ai_id=assistance_id WHERE assistance_type='ai';""")
    op.execute("""CREATE INDEX idx_ai_aicalls_ai_id ON ai_aicalls(ai_id);""")
    op.execute("""DROP INDEX idx_ai_aicalls_assistance_type ON ai_aicalls;""")
    op.execute("""DROP INDEX idx_ai_aicalls_assistance_id ON ai_aicalls;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN assistance_type;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN assistance_id;""")
    op.execute("""DROP TABLE IF EXISTS ai_teams;""")
