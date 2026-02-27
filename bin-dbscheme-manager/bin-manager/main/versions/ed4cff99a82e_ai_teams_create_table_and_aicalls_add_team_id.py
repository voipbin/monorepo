"""ai_teams create table and ai_aicalls add column team_id

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

    op.execute("""
        ALTER TABLE ai_aicalls
        ADD team_id BINARY(16)
        AFTER ai_id;
    """)

    op.execute("""
        CREATE INDEX idx_ai_aicalls_team_id ON ai_aicalls(team_id);
    """)


def downgrade():
    op.execute("""DROP INDEX idx_ai_aicalls_team_id ON ai_aicalls;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN team_id;""")
    op.execute("""DROP TABLE IF EXISTS ai_teams;""")
