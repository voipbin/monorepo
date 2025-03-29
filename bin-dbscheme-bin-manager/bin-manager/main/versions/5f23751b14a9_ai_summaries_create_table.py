"""ai_summaries create table

Revision ID: 5f23751b14a9
Revises: f11b33699921
Create Date: 2025-03-30 03:55:08.501715

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '5f23751b14a9'
down_revision = 'f11b33699921'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""CREATE TABLE ai_summaries (
        id            BINARY(16) NOT NULL,
        customer_id   BINARY(16) NOT NULL,

        activeflow_id     BINARY(16) NOT NULL,
        on_end_flow_id    BINARY(16) NOT NULL,

        reference_type    VARCHAR(255) NOT NULL,
        reference_id      BINARY(16) NOT NULL,

        status    VARCHAR(16) NOT NULL,
        language  VARCHAR(16) NOT NULL,
        content   TEXT NOT NULL,

        tm_create DATETIME(6),
        tm_update DATETIME(6),
        tm_delete DATETIME(6),

        PRIMARY KEY(id)
    );""")
    op.execute("""CREATE INDEX idx_ai_summaries_customer_id ON ai_summaries(customer_id);""")
    op.execute("""CREATE INDEX idx_ai_summaries_activeflow ON ai_summaries(activeflow_id);""")
    op.execute("""CREATE INDEX idx_ai_summaries_on_end_flow_id ON ai_summaries(on_end_flow_id);""")
    op.execute("""CREATE INDEX idx_ai_summaries_reference_id_language ON ai_summaries(reference_id, language);""")
    op.execute("""CREATE INDEX idx_ai_summaries_reference_type ON ai_summaries(reference_type);""")
    op.execute("""CREATE INDEX idx_ai_summaries_reference_id ON ai_summaries(reference_id);""")
    

def downgrade():
    op.execute("""drop table ai_summaries;""")

