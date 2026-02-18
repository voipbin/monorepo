"""call external_media_id to external_media_ids array

Revision ID: a3b4c5d6e7f8
Revises: 27b482dafc0c
Create Date: 2026-02-19 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'a3b4c5d6e7f8'
down_revision = '27b482dafc0c'
branch_labels = None
depends_on = None


def upgrade():
    # call_calls: drop old column and index, add new JSON array column
    op.execute("""drop index idx_call_calls_external_media_id on call_calls;""")
    op.execute("""alter table call_calls drop column external_media_id;""")
    op.execute("""alter table call_calls add column external_media_ids json default (json_array()) after recording_ids;""")

    # call_confbridges: drop old column, add new JSON array column
    op.execute("""alter table call_confbridges drop column external_media_id;""")
    op.execute("""alter table call_confbridges add column external_media_ids json default (json_array()) after recording_ids;""")


def downgrade():
    # call_calls: drop JSON array column, restore binary column and index
    op.execute("""alter table call_calls drop column external_media_ids;""")
    op.execute("""alter table call_calls add column external_media_id binary(16) default '' after recording_ids;""")
    op.execute("""create index idx_call_calls_external_media_id on call_calls(external_media_id);""")

    # call_confbridges: drop JSON array column, restore binary column
    op.execute("""alter table call_confbridges drop column external_media_ids;""")
    op.execute("""alter table call_confbridges add column external_media_id binary(16) default '' after recording_ids;""")
