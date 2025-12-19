"""drop index all customerid

Revision ID: 431ea0ae1aab
Revises: f12a66c5cc00
Create Date: 2022-01-31 00:55:17.960193

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '431ea0ae1aab'
down_revision = 'f12a66c5cc00'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""create index idx_agents_customerid on agents(customer_id);""")
    op.execute("""create index idx_calls_customerid on calls(customer_id);""")
    op.execute("""create index idx_conferences_customerid on conferences(customer_id);""")
    op.execute("""create index idx_domains_customerid on domains(customer_id);""")
    op.execute("""create index idx_extensions_customerid on extensions(customer_id);""")
    op.execute("""create index idx_flows_customerid on flows(customer_id);""")
    op.execute("""create index idx_numbers_customerid on numbers(customer_id);""")
    op.execute("""create index idx_queuecallreferences_customerid on queuecallreferences(customer_id);""")
    op.execute("""create index idx_queuecalls_customerid on queuecalls(customer_id);""")
    op.execute("""create index idx_queues_customerid on queues(customer_id);""")
    op.execute("""create index idx_recordings_customerid on recordings(customer_id);""")
    op.execute("""create index idx_tags_customerid on tags(customer_id);""")
    op.execute("""create index idx_transcribes_customerid on transcribes(customer_id);""")


    op.execute("""alter table agents drop column user_id;""")
    op.execute("""alter table calls drop column user_id;""")
    op.execute("""alter table conferences drop column user_id;""")
    op.execute("""alter table domains drop column user_id;""")
    op.execute("""alter table extensions drop column user_id;""")
    op.execute("""alter table flows drop column user_id;""")
    op.execute("""alter table numbers drop column user_id;""")
    op.execute("""alter table queuecallreferences drop column user_id;""")
    op.execute("""alter table queuecalls drop column user_id;""")
    op.execute("""alter table queues drop column user_id;""")
    op.execute("""alter table recordings drop column user_id;""")
    op.execute("""alter table tags drop column user_id;""")
    op.execute("""alter table transcribes drop column user_id;""")



def downgrade():
    op.execute("""alter table agents drop index idx_agents_customerid;""")
    op.execute("""alter table calls drop index idx_calls_customerid;""")
    op.execute("""alter table conferences drop index idx_conferences_customerid;""")
    op.execute("""alter table domains drop index idx_domains_customerid;""")
    op.execute("""alter table extensions drop index idx_extensions_customerid;""")
    op.execute("""alter table flows drop index idx_flows_customerid;""")
    op.execute("""alter table numbers drop index idx_numbers_customerid;""")
    op.execute("""alter table queuecallreferences drop index idx_queuecallreferences_customerid;""")
    op.execute("""alter table queuecalls drop index idx_queuecalls_customerid;""")
    op.execute("""alter table queues drop index idx_queues_customerid;""")
    op.execute("""alter table recordings drop index idx_recordings_customerid;""")
    op.execute("""alter table tags drop index idx_tags_customerid;""")
    op.execute("""alter table transcribes drop index idx_transcribes_customerid;""")
