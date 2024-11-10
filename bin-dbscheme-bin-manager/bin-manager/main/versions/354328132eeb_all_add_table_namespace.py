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
    op.execute("""alter table agent_agents rename index idx_activeflows_customer_id to idx_flow_activeflows_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_activeflows_flow_id to idx_flow_activeflows_flow_id;""")
    op.execute("""alter table agent_agents rename index idx_activeflows_reference_id to idx_flow_activeflows_reference_id;""")
    
    op.execute("""rename table agents to agent_agents;""")
    op.execute("""alter table agent_agents rename index idx_agents_customerid to idx_agents_customerid;""")
    op.execute("""alter table agent_agents rename index idx_agents_username to idx_agent_agents_username;""")
    
    
    op.execute("""rename table bridges to call_bridges;""")
    op.execute("""alter table agent_agents rename index idx_bridges_create to idx_call_bridges_create;""")
    op.execute("""alter table agent_agents rename index idx_bridges_reference_id to idx_call_bridges_reference_id;""")

    op.execute("""rename table calls to call_calls;""")
    op.execute("""alter table agent_agents rename index idx_calls_customer_id to idx_call_calls_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_calls_owner_id to idx_call_calls_owner_id;""")
    op.execute("""alter table agent_agents rename index idx_calls_channelid to idx_call_calls_channelid;""")
    op.execute("""alter table agent_agents rename index idx_calls_flowid to idx_call_calls_flowid;""")
    op.execute("""alter table agent_agents rename index idx_calls_create to idx_call_calls_create;""")
    op.execute("""alter table agent_agents rename index idx_calls_hangup to idx_call_calls_hangup;""")
    op.execute("""alter table agent_agents rename index idx_calls_source_target to idx_call_calls_source_target;""")
    op.execute("""alter table agent_agents rename index idx_calls_destination_target to idx_call_calls_destination_target;""")
    op.execute("""alter table agent_agents rename index idx_calls_external_media_id to idx_call_calls_external_media_id;""")
    op.execute("""alter table agent_agents rename index idx_calls_groupcall_id to idx_call_calls_groupcall_id;""")

    
    
    op.execute("""rename table campaigncalls to campaign_campaigncalls;""")
    op.execute("""alter table agent_agents rename index idx_campaigncalls_customer_id to idx_campaign_campaigncalls_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigncalls_campaign_id to idx_campaign_campaigncalls_campaign_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigncalls_outdial_target_id to idx_campaign_campaigncalls_outdial_target_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigncalls_activeflow_id to idx_campaign_campaigncalls_activeflow_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigncalls_reference_id to idx_campaign_campaigncalls_reference_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigncalls_campaign_id_status to idx_campaign_campaigncalls_campaign_id_status;""")

    op.execute("""rename table campaigns to campaign_campaigns;""")
    op.execute("""alter table agent_agents rename index idx_campaigns_customer_id to idx_campaign_campaigns_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigns_flow_id to idx_campaign_campaigns_flow_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigns_outplan_id to idx_campaign_campaigns_outplan_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigns_outdial_id to idx_campaign_campaigns_outdial_id;""")
    op.execute("""alter table agent_agents rename index idx_campaigns_queue_id to idx_campaign_campaigns_queue_id;""")

    
    op.execute("""rename table channels to call_channels;""")
    op.execute("""alter table agent_agents rename index idx_channels_create to idx_call_channels_create;""")
    op.execute("""alter table agent_agents rename index idx_channels_src_number to idx_call_channels_src_number;""")
    op.execute("""alter table agent_agents rename index idx_channels_dst_number to idx_call_channels_dst_number;""")
    op.execute("""alter table agent_agents rename index idx_channels_sip_call_id to idx_call_channels_sip_call_id;""")

    
    op.execute("""rename table chatbotcalls to chatbot_chatbotcalls;""")
    op.execute("""rename table chatbots to chatbot_chatbots;""")
    
    op.execute("""rename table chatrooms to chat_chatrooms;""")
    op.execute("""alter table agent_agents rename index idx_chatrooms_customer_id to idx_chat_chatrooms_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_chatrooms_chat_id to idx_chat_chatrooms_chat_id;""")
    op.execute("""alter table agent_agents rename index idx_chatrooms_owner_id to idx_chat_chatrooms_owner_id;""")
    op.execute("""alter table agent_agents rename index idx_chatrooms_chat_id_owner_id to idx_chat_chatrooms_chat_id_owner_id;""")
    op.execute("""alter table agent_agents rename index idx_chatrooms_room_owner_id to idx_chat_chatrooms_room_owner_id;""")
    
    
    op.execute("""rename table chats to chat_chats;""")
    op.execute("""alter table agent_agents rename index idx_chats_customer_id to idx_chat_chats_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_chats_owner_id to idx_chat_chats_owner_id;""")
    
    op.execute("""rename table confbridges to call_confbridges;""")
    op.execute("""alter table agent_agents rename index idx_confbridges_create to idx_call_confbridges_create;""")
    op.execute("""alter table agent_agents rename index idx_confbridges_customer_id to idx_call_confbridges_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_confbridges_bridge_id to idx_call_confbridges_bridge_id;""")

    
    op.execute("""rename table conferencecalls to conference_conferencecalls;""")
    op.execute("""rename table conferences to conference_conferences;""")
    op.execute("""rename table customers to customer_customers;""")
    op.execute("""rename table extensions to registrar_extensions;""")
    op.execute("""rename table flows to flow_flows;""")
    
    op.execute("""rename table groupcalls to call_groupcalls;""")
    op.execute("""alter table agent_agents rename index idx_groupcalls_customer_id to idx_call_groupcalls_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_groupcalls_customer_id to idx_call_groupcalls_customer_id;""")


    op.execute("""rename table messagechatrooms to chat_messagechatrooms;""")
    op.execute("""alter table agent_agents rename index idx_messagechatrooms_customer_id to idx_chat_messagechatrooms_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_messagechatrooms_chatroom_id to idx_chat_messagechatrooms_chatroom_id;""")
    op.execute("""alter table agent_agents rename index idx_messagechatrooms_messagechat_id to idx_chat_messagechatrooms_messagechat_id;""")



    op.execute("""rename table messagechats to chat_messagechats;""")
    op.execute("""alter table agent_agents rename index idx_messagechats_customer_id to idx_chat_messagechats_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_messagechats_chat_id to idx_chat_messagechats_chat_id;""")



    op.execute("""rename table messages to message_messages;""")
    op.execute("""rename table numbers to number_numbers;""")
    op.execute("""rename table outdials to outdial_outdials;""")
    op.execute("""rename table outdialtargets to outdial_outdialtargets;""")
    
    op.execute("""rename table outplans to campaign_outplans;""")
    op.execute("""alter table agent_agents rename index idx_outplans_customer_id to idx_campaign_outplans_customer_id;""")

    
    op.execute("""rename table providers to route_providers;""")
    op.execute("""rename table queuecallreferences to queue_queuecallreferences;""")
    op.execute("""rename table queuecalls to queue_queuecalls;""")
    op.execute("""rename table queues to queue_queues;""")
    
    op.execute("""rename table recordings to call_recordings;""")
    op.execute("""alter table agent_agents rename index idx_recordings_tm_start to idx_call_recordings_tm_start;""")
    op.execute("""alter table agent_agents rename index idx_recordings_customer_id to idx_call_recordings_customer_id;""")
    op.execute("""alter table agent_agents rename index idx_recordings_owner_id to idx_call_recordings_owner_id;""")
    op.execute("""alter table agent_agents rename index idx_recordings_reference_id to idx_call_recordings_reference_id;""")
    op.execute("""alter table agent_agents rename index idx_recordings_recording_name to idx_call_recordings_recording_name;""")

    
    op.execute("""rename table routes to route_routes;""")
    op.execute("""rename table tags to tag_tags;""")
    op.execute("""rename table transcribes to transcribe_transcribes;""")
    op.execute("""rename table transcripts to transcribe_transcripts;""")
    op.execute("""rename table outdialtargets to outdial_outdialtargets;""")
    op.execute("""rename table outdialtargets to outdial_outdialtargets;""")
    op.execute("""rename table outdialtargets to outdial_outdialtargets;""")


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
    op.execute("""rename table outdial_outdialtargets to outdialtargets;""")
    op.execute("""rename table outdial_outdialtargets to outdialtargets;""")
    op.execute("""rename table outdial_outdialtargets to outdialtargets;""")

