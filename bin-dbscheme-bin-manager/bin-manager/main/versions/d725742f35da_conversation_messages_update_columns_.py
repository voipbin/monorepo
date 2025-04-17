"""conversation_messages update columns reference_id

Revision ID: d725742f35da
Revises: 7a27decc13da
Create Date: 2025-04-18 07:57:03.188182

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd725742f35da'
down_revision = '7a27decc13da'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE conversation_messages ADD tmp_reference_id BINARY(16);
    """)
    op.execute("""
        UPDATE conversation_messages SET tmp_reference_id = UNHEX(REPLACE(reference_id, '-', ''));
    """)
    op.execute("""
        ALTER TABLE conversation_messages DROP COLUMN reference_id;
    """)
    op.execute("""
        ALTER TABLE conversation_messages CHANGE tmp_reference_id reference_id BINARY(16);
    """)



def downgrade():
    op.execute("""
        ALTER TABLE conversation_messages ADD tmp_reference_id VARCHAR(36);
    """)
    op.execute("""
        UPDATE conversation_messages
        SET tmp_reference_id = LOWER(CONCAT(
            SUBSTR(HEX(reference_id), 1, 8), '-', 
            SUBSTR(HEX(reference_id), 9, 4), '-', 
            SUBSTR(HEX(reference_id), 13, 4), '-', 
            SUBSTR(HEX(reference_id), 17, 4), '-', 
            SUBSTR(HEX(reference_id), 21)
        ));
    """)
    op.execute("""
        ALTER TABLE conversation_messages DROP COLUMN reference_id;
    """)
    op.execute("""
        ALTER TABLE conversation_messages CHANGE tmp_reference_id reference_id VARCHAR(36);
    """)
