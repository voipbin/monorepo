"""add all customer id

Revision ID: f12a66c5cc00
Revises: 9142c979f5df
Create Date: 2022-01-29 00:37:02.906757

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f12a66c5cc00'
down_revision = '9142c979f5df'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table agents add column customer_id binary(16) after id;""")
    op.execute("""alter table calls add column customer_id binary(16) after id;""")
    op.execute("""alter table conferences add column customer_id binary(16) after id;""")
    op.execute("""alter table domains add column customer_id binary(16) after id;""")
    op.execute("""alter table extensions add column customer_id binary(16) after id;""")
    op.execute("""alter table flows add column customer_id binary(16) after id;""")
    op.execute("""alter table numbers add column customer_id binary(16) after id;""")
    op.execute("""alter table queuecallreferences add column customer_id binary(16) after id;""")
    op.execute("""alter table queuecalls add column customer_id binary(16) after id;""")
    op.execute("""alter table queues add column customer_id binary(16) after id;""")
    op.execute("""alter table recordings add column customer_id binary(16) after id;""")
    op.execute("""alter table tags add column customer_id binary(16) after id;""")
    op.execute("""alter table transcribes add column customer_id binary(16) after id;""")


def downgrade():
    op.execute("""alter table agents drop column customer_id;""")
    op.execute("""alter table calls drop column customer_id;""")
    op.execute("""alter table conferences drop column customer_id;""")
    op.execute("""alter table domains drop column customer_id;""")
    op.execute("""alter table extensions drop column customer_id;""")
    op.execute("""alter table flows drop column customer_id;""")
    op.execute("""alter table numbers drop column customer_id;""")
    op.execute("""alter table queuecallreferences drop column customer_id;""")
    op.execute("""alter table queuecalls drop column customer_id;""")
    op.execute("""alter table queues drop column customer_id;""")
    op.execute("""alter table recordings drop column customer_id;""")
    op.execute("""alter table tags drop column customer_id;""")
    op.execute("""alter table transcribes drop column customer_id;""")
