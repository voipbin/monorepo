"""calls add column owner_type owner_id

Revision ID: c3fb144b1c95
Revises: 0886f56ccfbe
Create Date: 2024-06-17 01:04:25.634294

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c3fb144b1c95'
down_revision = '0886f56ccfbe'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls add column owner_type varchar(255) after customer_id;""")
    op.execute("""alter table calls add column owner_id binary(16) after owner_type;""")
    op.execute("""update calls set owner_type = "";""")
    op.execute("""update calls set owner_id = "";""")
    op.execute("""create index idx_calls_owner_id on calls(owner_id);""")



    op.execute("""alter table groupcalls add column owner_type varchar(255) after customer_id;""")
    op.execute("""alter table groupcalls add column owner_id binary(16) after owner_type;""")
    op.execute("""update groupcalls set owner_type = "";""")
    op.execute("""update groupcalls set owner_id = "";""")
    op.execute("""create index idx_groupcalls_owner_id on groupcalls(owner_id);""")



    op.execute("""alter table recordings add column owner_type varchar(255) after customer_id;""")
    op.execute("""alter table recordings add column owner_id binary(16) after owner_type;""")
    op.execute("""update recordings set owner_type = "";""")
    op.execute("""update recordings set owner_id = "";""")
    op.execute("""create index idx_recordings_owner_id on recordings(owner_id);""")


def downgrade():
    op.execute("""alter table calls drop column owner_type;""")
    op.execute("""alter table calls drop column owner_id;""")
    
    op.execute("""alter table groupcalls drop column owner_type;""")
    op.execute("""alter table groupcalls drop column owner_id;""")
    
    op.execute("""alter table recordings drop column owner_type;""")
    op.execute("""alter table recordings drop column owner_id;""")
