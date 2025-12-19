"""conference_conferences change columns pre_flow_id and post_flow_id

Revision ID: 268e0c57e1ec
Revises: d725742f35da
Create Date: 2025-04-21 04:45:33.441198

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '268e0c57e1ec'
down_revision = 'd725742f35da'
branch_labels = None
depends_on = None


def upgrade():
    # Add the columns without the default value
    op.execute("""
        ALTER TABLE conference_conferences 
        ADD COLUMN pre_flow_id BINARY(16) 
        AFTER timeout;
    """)
    op.execute("""
        ALTER TABLE conference_conferences 
        ADD COLUMN post_flow_id BINARY(16) 
        AFTER pre_flow_id;
    """)

    # Update the columns to set UUID.Nil (0x00000000000000000000000000000000)
    op.execute("""
        UPDATE conference_conferences 
        SET pre_flow_id = UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))
    """)
    op.execute("""
        UPDATE conference_conferences 
        SET post_flow_id = UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))
    """)

    # Drop the old columns
    op.execute("""ALTER TABLE conference_conferences DROP COLUMN pre_actions;""")
    op.execute("""ALTER TABLE conference_conferences DROP COLUMN post_actions;""")

def downgrade():
    op.execute("""ALTER TABLE conference_conferences ADD COLUMN pre_actions JSON;""")
    op.execute("""ALTER TABLE conference_conferences ADD COLUMN post_actions JSON;""")
    op.execute("""ALTER TABLE conference_conferences DROP COLUMN pre_flow_id;""")
    op.execute("""ALTER TABLE conference_conferences DROP COLUMN post_flow_id;""")
