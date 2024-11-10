"""all_add_table_namespace

Revision ID: 354328132eeb
Revises: 05a3b7905842
Create Date: 2024-11-08 21:50:50.107150

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '354328132eeb'
down_revision = '05a3b7905842'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""rename table activeflows to flow_activeflows;""")
    op.execute("""rename table agents to agent_agents;""")
    op.execute("""rename table bridges to call_bridges;""")
    op.execute("""rename table calls to call_calls;""")
    op.execute("""rename table campaigncalls to campaign_campaigncalls;""")
    op.execute("""rename table campaigns to campaign_campaigns;""")
    op.execute("""rename table channels to call_channels;""")
    op.execute("""rename table chatbotcalls to chatbot_chatbotcalls;""")
    op.execute("""rename table chatbots to chatbot_chatbots;""")
    op.execute("""rename table chatrooms to chat_chatrooms;""")    
    op.execute("""rename table chats to chat_chats;""")
    op.execute("""rename table confbridges to call_confbridges;""")
    op.execute("""rename table conferencecalls to conference_conferencecalls;""")
    op.execute("""rename table conferences to conference_conferences;""")
    op.execute("""rename table customers to customer_customers;""")
    op.execute("""rename table extensions to registrar_extensions;""")
    op.execute("""rename table flows to flow_flows;""")
    op.execute("""rename table groupcalls to call_groupcalls;""")
    op.execute("""rename table messagechatrooms to chat_messagechatrooms;""")
    op.execute("""rename table messagechats to chat_messagechats;""")
    op.execute("""rename table messages to message_messages;""")
    op.execute("""rename table numbers to number_numbers;""")
    op.execute("""rename table outdials to outdial_outdials;""")
    op.execute("""rename table outdialtargets to outdial_outdialtargets;""")

    op.execute("""
        create table outdial_outdialtargetcalls(
        id                binary(16),
        customer_id       binary(16),
        campaign_id       binary(16),
        outdial_id        binary(16),
        outdial_target_id binary(16),

        activeflow_id     binary(16),
        reference_type  varchar(255),
        reference_id    binary(16),

        status          varchar(255),

        destination       json,
        destination_index integer,
        try_count         integer,

        tm_create datetime(6),  -- create
        tm_update datetime(6),  -- update
        tm_delete datetime(6),  -- delete

        primary key(id)
        );
    """)
    op.execute("""create index idx_outdial_outdialtargetcalls_customer_id on outdial_outdialtargetcalls(customer_id);""")
    op.execute("""create index idx_outdial_outdialtargetcalls_campaign_id on outdial_outdialtargetcalls(campaign_id);""")
    op.execute("""create index idx_outdial_outdialtargetcalls_outdial_target_id on outdial_outdialtargetcalls(outdial_target_id);""")
    op.execute("""create index idx_outdial_outdialtargetcalls_activeflow_id on outdial_outdialtargetcalls(activeflow_id);""")
    op.execute("""create index idx_outdial_outdialtargetcalls_reference_id on outdial_outdialtargetcalls(reference_id);""")

    op.execute("""rename table outplans to campaign_outplans;""")
    op.execute("""rename table providers to route_providers;""")
    op.execute("""rename table queuecallreferences to queue_queuecallreferences;""")
    op.execute("""rename table queuecalls to queue_queuecalls;""")
    op.execute("""rename table queues to queue_queues;""")
    op.execute("""rename table recordings to call_recordings;""")
    op.execute("""rename table routes to route_routes;""")
    op.execute("""rename table tags to tag_tags;""")
    op.execute("""rename table transcribes to transcribe_transcribes;""")
    op.execute("""rename table transcripts to transcribe_transcripts;""")



def downgrade():
    op.execute("""rename table flow_activeflows to activeflows;""")
    op.execute("""rename table agent_agents to agents;""")
    op.execute("""rename table call_bridges to bridges;""")
    op.execute("""rename table call_calls to calls;""")
    op.execute("""rename table campaign_campaigncalls to campaigncalls;""")
    op.execute("""rename table campaign_campaigns to campaigns;""")
    op.execute("""rename table call_channels to channels;""")
    op.execute("""rename table chatbot_chatbotcalls to chatbotcalls;""")
    op.execute("""rename table chatbot_chatbots to chatbots;""")
    op.execute("""rename table chat_chatrooms to chatrooms;""")
    op.execute("""rename table chat_chats to chats;""")
    op.execute("""rename table call_confbridges to confbridges;""")
    op.execute("""rename table conference_conferencecalls to conferencecalls;""")
    op.execute("""rename table conference_conferences to conferences;""")
    op.execute("""rename table customer_customers to customers;""")    
    op.execute("""rename table registrar_extensions to extensions;""")
    op.execute("""rename table flow_flows to flows;""")
    op.execute("""rename table call_groupcalls to groupcalls;""")
    op.execute("""rename table chat_messagechatrooms to messagechatrooms;""")
    op.execute("""rename table chat_messagechats to messagechats;""")
    op.execute("""rename table message_messages to messages;""")
    op.execute("""rename table number_numbers to numbers;""")
    op.execute("""rename table outdial_outdials to outdials;""")
    op.execute("""rename table outdial_outdialtargets to outdialtargets;""")

    # Drop table created in upgrade
    op.execute("drop table outdial_outdialtargetcalls")

    op.execute("""rename table campaign_outplans to outplans;""")
    op.execute("""rename table route_providers to providers;""")  
    op.execute("""rename table queue_queuecallreferences to queuecallreferences;""")
    op.execute("""rename table queue_queuecalls to queuecalls;""")
    op.execute("""rename table queue_queues to queues;""")
    op.execute("""rename table call_recordings to recordings;""")
    op.execute("""rename table route_routes to routes;""")
    op.execute("""rename table tag_tags to tags;""")
    op.execute("""rename table transcribe_transcribes to transcribes;""")
    op.execute("""rename table transcribe_transcripts to transcripts;""")

