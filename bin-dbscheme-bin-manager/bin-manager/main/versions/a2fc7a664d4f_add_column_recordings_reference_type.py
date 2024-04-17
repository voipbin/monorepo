"""add column recordings reference_type

Revision ID: a2fc7a664d4f
Revises: 79f83144d79f
Create Date: 2023-01-05 17:02:04.725003

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a2fc7a664d4f'
down_revision = '79f83144d79f'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table recordings change type reference_type varchar(16);""")
    op.execute("""alter table recordings add column recording_name varchar(255) after format;""")
    op.execute("""alter table recordings add column filenames json after recording_name;""")
    op.execute("""alter table recordings add column channel_ids json after asterisk_id;""")

    op.execute("""create index idx_recordings_reference_id on recordings(reference_id);""")
    op.execute("""create index idx_recordings_recording_name on recordings(recording_name);""")

    op.execute("""alter table recordings drop column filename;""")
    op.execute("""alter table recordings drop column channel_id;""")


def downgrade():
    op.execute("""alter table recordings drop index idx_recordings_reference_id;""")
    op.execute("""alter table recordings drop index idx_recordings_recording_name;""")

    op.execute("""alter table recordings change reference_type type varchar(16);""")
    op.execute("""alter table recordings drop column recording_name;""")
    op.execute("""alter table recordings drop column filenames;""")
    op.execute("""alter table recordings drop column channel_ids;""")

    op.execute("""alter table recordings add column filename varchar(255) after format;""")
    op.execute("""alter table recordings add column channel_id varchar(255) after asterisk_id;""")

    op.execute("""create index idx_recordings_filename on recordings(filename);""")



