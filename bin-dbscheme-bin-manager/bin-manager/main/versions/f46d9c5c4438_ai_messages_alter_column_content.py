"""ai_messages alter column content

Revision ID: f46d9c5c4438
Revises: bad27b40fe8e
Create Date: 2025-11-15 04:56:30.216182

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f46d9c5c4438'
down_revision = 'bad27b40fe8e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE ai_messages 
        MODIFY COLUMN content TEXT 
        CHARACTER SET utf8mb4 
        COLLATE utf8mb4_general_ci;""")


def downgrade():
    op.execute("""
        ALTER TABLE ai_messages 
        MODIFY COLUMN content TEXT 
        CHARACTER SET utf8mb3 
        COLLATE utf8mb3_general_ci;
    """)
