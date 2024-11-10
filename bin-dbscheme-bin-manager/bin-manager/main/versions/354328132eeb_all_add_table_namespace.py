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
    op.execute("""alter table flow_activeflows rename index idx_activeflows_customer_id to idx_flow_activeflows_customer_id;""")
    op.execute("""alter table flow_activeflows rename index idx_activeflows_flow_id to idx_flow_activeflows_flow_id;""")
    op.execute("""alter table flow_activeflows rename index idx_activeflows_reference_id to idx_flow_activeflows_reference_id;""")

    op.execute("""rename table agents to agent_agents;""")
    op.execute("""alter table agent_agents rename index idx_agents_customerid to idx_agents_customerid;""")
    op.execute("""alter table agent_agents rename index idx_agents_username to idx_agent_agents_username;""")

    op.execute("""rename table bridges to call_bridges;""")
    op.execute("""alter table call_bridges rename index idx_bridges_create to idx_call_bridges_create;""")
    op.execute("""alter table call_bridges rename index idx_bridges_reference_id to idx_call_bridges_reference_id;""")

    op.execute("""rename table calls to call_calls;""")
    op.execute("""alter table call_calls rename index idx_calls_customer_id to idx_call_calls_customer_id;""")
    op.execute("""alter table call_calls rename index idx_calls_owner_id to idx_call_calls_owner_id;""")
    op.execute("""alter table call_calls rename index idx_calls_channelid to idx_call_calls_channelid;""")
    op.execute("""alter table call_calls rename index idx_calls_flowid to idx_call_calls_flowid;""")
    op.execute("""alter table call_calls rename index idx_calls_create to idx_call_calls_create;""")
    op.execute("""alter table call_calls rename index idx_calls_hangup to idx_call_calls_hangup;""")
    op.execute("""alter table call_calls rename index idx_calls_source_target to idx_call_calls_source_target;""")
    op.execute("""alter table call_calls rename index idx_calls_destination_target to idx_call_calls_destination_target;""")
    op.execute("""alter table call_calls rename index idx_calls_external_media_id to idx_call_calls_external_media_id;""")
    op.execute("""alter table call_calls rename index idx_calls_groupcall_id to idx_call_calls_groupcall_id;""")

    op.execute("""rename table campaigncalls to campaign_campaigncalls;""")
    op.execute("""alter table campaign_campaigncalls rename index idx_campaigncalls_customer_id to idx_campaign_campaigncalls_customer_id;""")
    op.execute("""alter table campaign_campaigncalls rename index idx_campaigncalls_campaign_id to idx_campaign_campaigncalls_campaign_id;""")
    op.execute("""alter table campaign_campaigncalls rename index idx_campaigncalls_outdial_target_id to idx_campaign_campaigncalls_outdial_target_id;""")
    op.execute("""alter table campaign_campaigncalls rename index idx_campaigncalls_activeflow_id to idx_campaign_campaigncalls_activeflow_id;""")
    op.execute("""alter table campaign_campaigncalls rename index idx_campaigncalls_reference_id to idx_campaign_campaigncalls_reference_id;""")
    op.execute("""alter table campaign_campaigncalls rename index idx_campaigncalls_campaign_id_status to idx_campaign_campaigncalls_campaign_id_status;""")

    op.execute("""rename table campaigns to campaign_campaigns;""")
    op.execute("""alter table campaign_campaigns rename index idx_campaigns_customer_id to idx_campaign_campaigns_customer_id;""")
    op.execute("""alter table campaign_campaigns rename index idx_campaigns_flow_id to idx_campaign_campaigns_flow_id;""")
    op.execute("""alter table campaign_campaigns rename index idx_campaigns_outplan_id to idx_campaign_campaigns_outplan_id;""")
    op.execute("""alter table campaign_campaigns rename index idx_campaigns_outdial_id to idx_campaign_campaigns_outdial_id;""")
    op.execute("""alter table campaign_campaigns rename index idx_campaigns_queue_id to idx_campaign_campaigns_queue_id;""")

    op.execute("""rename table channels to call_channels;""")
    op.execute("""alter table call_channels rename index idx_channels_create to idx_call_channels_create;""")
    op.execute("""alter table call_channels rename index idx_channels_src_number to idx_call_channels_src_number;""")
    op.execute("""alter table call_channels rename index idx_channels_dst_number to idx_call_channels_dst_number;""")
    op.execute("""alter table call_channels rename index idx_channels_sip_call_id to idx_call_channels_sip_call_id;""")

    op.execute("""rename table chatbotcalls to chatbot_chatbotcalls;""")
    op.execute("""alter table chatbot_chatbotcalls rename index idx_chatbotcalls_customer_id to idx_chatbot_chatbotcalls_customer_id;""")
    op.execute("""alter table chatbot_chatbotcalls rename index idx_chatbotcalls_chatbot_id to idx_chatbot_chatbotcalls_chatbot_id;""")
    op.execute("""alter table chatbot_chatbotcalls rename index idx_chatbotcalls_reference_type to idx_chatbot_chatbotcalls_reference_type;""")
    op.execute("""alter table chatbot_chatbotcalls rename index idx_chatbotcalls_reference_id to idx_chatbot_chatbotcalls_reference_id;""")
    op.execute("""alter table chatbot_chatbotcalls rename index idx_chatbotcalls_transcribe_id to idx_chatbot_chatbotcalls_transcribe_id;""")
    op.execute("""alter table chatbot_chatbotcalls rename index idx_chatbotcalls_create to idx_chatbot_chatbotcalls_create;""")
    op.execute("""alter table chatbot_chatbotcalls rename index idx_chatbotcalls_activeflow_id to idx_chatbot_chatbotcalls_activeflow_id;""")

    op.execute("""rename table chatbots to chatbot_chatbots;""")
    op.execute("""alter table chatbot_chatbots rename index idx_chatbots_create to idx_chatbot_chatbots_create;""")
    op.execute("""alter table chatbot_chatbots rename index idx_chatbots_customer_id to idx_chatbot_chatbots_customer_id;""")

    op.execute("""rename table chatrooms to chat_chatrooms;""")
    op.execute("""alter table chat_chatrooms rename index idx_chatrooms_customer_id to idx_chat_chatrooms_customer_id;""")
    op.execute("""alter table chat_chatrooms rename index idx_chatrooms_chat_id to idx_chat_chatrooms_chat_id;""")
    op.execute("""alter table chat_chatrooms rename index idx_chatrooms_owner_id to idx_chat_chatrooms_owner_id;""")
    op.execute("""alter table chat_chatrooms rename index idx_chatrooms_chat_id_owner_id to idx_chat_chatrooms_chat_id_owner_id;""")
    op.execute("""alter table chat_chatrooms rename index idx_chatrooms_room_owner_id to idx_chat_chatrooms_room_owner_id;""")
    
    op.execute("""rename table chats to chat_chats;""")
    op.execute("""alter table chat_chats rename index idx_chats_customer_id to idx_chat_chats_customer_id;""")
    op.execute("""alter table chat_chats rename index idx_chats_owner_id to idx_chat_chats_owner_id;""")
    
    op.execute("""rename table confbridges to call_confbridges;""")
    op.execute("""alter table call_confbridges rename index idx_confbridges_create to idx_call_confbridges_create;""")
    op.execute("""alter table call_confbridges rename index idx_confbridges_customer_id to idx_call_confbridges_customer_id;""")
    op.execute("""alter table call_confbridges rename index idx_confbridges_bridge_id to idx_call_confbridges_bridge_id;""")
    
    op.execute("""rename table conferencecalls to conference_conferencecalls;""")
    op.execute("""alter table conference_conferencecalls rename index idx_conferencecalls_customer_id to idx_conference_conferencecalls_customer_id;""")
    op.execute("""alter table conference_conferencecalls rename index idx_conferencecalls_conference_id to idx_conference_conferencecalls_conference_id;""")
    op.execute("""alter table conference_conferencecalls rename index idx_conferencecalls_reference_id to idx_conference_conferencecalls_reference_id;""")
    op.execute("""alter table conference_conferencecalls rename index idx_conferencecalls_create to idx_conference_conferencecalls_create;""")

    op.execute("""rename table conferences to conference_conferences;""")
    op.execute("""alter table conference_conferences rename index idx_conferences_create to idx_conference_conferences_create;""")
    op.execute("""alter table conference_conferences rename index idx_conferences_customer_id to idx_conference_conferences_customer_id;""")
    op.execute("""alter table conference_conferences rename index idx_conferences_flow_id to idx_conference_conferences_flow_id;""")
    op.execute("""alter table conference_conferences rename index idx_conferences_confbridge_id to idx_conference_conferences_confbridge_id;""")

    op.execute("""rename table customers to customer_customers;""")
    
    op.execute("""rename table extensions to registrar_extensions;""")
    op.execute("""alter table registrar_extensions rename index idx_extensions_customerid to idx_registrar_extensions_customerid;""")
    op.execute("""alter table registrar_extensions rename index idx_extensions_extension to idx_registrar_extensions_extension;""")
    op.execute("""alter table registrar_extensions rename index idx_extensions_domain_name to idx_registrar_extensions_domain_name;""")
    op.execute("""alter table registrar_extensions rename index idx_extensions_username to idx_registrar_extensions_username;""")
    op.execute("""alter table registrar_extensions rename index idx_extensions_realm to idx_registrar_extensions_realm;""")

    op.execute("""rename table flows to flow_flows;""")
    op.execute("""alter table flow_flows rename index idx_customer_id to idx_flows_customer_id;""")

    op.execute("""rename table groupcalls to call_groupcalls;""")
    op.execute("""alter table call_groupcalls rename index idx_groupcalls_customer_id to idx_call_groupcalls_customer_id;""")
    op.execute("""alter table call_groupcalls rename index idx_groupcalls_owner_id to idx_call_groupcalls_owner_id;""")

    op.execute("""rename table messagechatrooms to chat_messagechatrooms;""")
    op.execute("""alter table chat_messagechatrooms rename index idx_messagechatrooms_customer_id to idx_chat_messagechatrooms_customer_id;""")
    op.execute("""alter table chat_messagechatrooms rename index idx_messagechatrooms_chatroom_id to idx_chat_messagechatrooms_chatroom_id;""")
    op.execute("""alter table chat_messagechatrooms rename index idx_messagechatrooms_messagechat_id to idx_chat_messagechatrooms_messagechat_id;""")

    op.execute("""rename table messagechats to chat_messagechats;""")
    op.execute("""alter table chat_messagechats rename index idx_messagechats_customer_id to idx_chat_messagechats_customer_id;""")
    op.execute("""alter table chat_messagechats rename index idx_messagechats_chat_id to idx_chat_messagechats_chat_id;""")

    op.execute("""rename table messages to message_messages;""")
    op.execute("""alter table message_messages rename index idx_messages_customerid to idx_message_messages_customerid;""")
    op.execute("""alter table message_messages rename index idx_messages_provider_name to idx_message_messages_provider_name;""")
    op.execute("""alter table message_messages rename index idx_messages_provider_reference_id to idx_message_messages_provider_reference_id;""")

    op.execute("""rename table numbers to number_numbers;""")
    op.execute("""alter table number_numbers rename index idx_numbers_number to idx_number_numbers_number;""")
    op.execute("""alter table number_numbers rename index idx_numbers_customerid to idx_number_numbers_customerid;""")
    op.execute("""alter table number_numbers rename index idx_numbers_call_flow_id to idx_number_numbers_call_flow_id;""")
    op.execute("""alter table number_numbers rename index idx_numbers_message_flow_id to idx_number_numbers_message_flow_id;""")
    op.execute("""alter table number_numbers rename index idx_numbers_provider_name to idx_number_numbers_provider_name;""")
    op.execute("""alter table number_numbers rename index idx_numbers_tm_renew to idx_number_numbers_tm_renew;""")

    op.execute("""rename table outdials to outdial_outdials;""")
    op.execute("""alter table outdial_outdials rename index idx_outdials_customer_id to idx_outdial_outdials_customer_id;""")
    op.execute("""alter table outdial_outdials rename index idx_outdials_campaign_id to idx_outdial_outdials_campaign_id;""")

    op.execute("""rename table outdialtargets to outdial_outdialtargets;""")
    op.execute("""alter table outdial_outdialtargets rename index idx_outdialtargets_outdial_id to idx_outdial_outdialtargets_outdial_id;""")

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
    op.execute("""alter table campaign_outplans rename index idx_outplans_customer_id to idx_campaign_outplans_customer_id;""")

    op.execute("""rename table providers to route_providers;""")
    
    op.execute("""rename table queuecallreferences to queue_queuecallreferences;""")

    op.execute("""rename table queuecalls to queue_queuecalls;""")
    op.execute("""alter table queue_queuecalls rename index idx_queuecalls_customerid to idx_queue_queuecalls_customerid;""")
    op.execute("""alter table queue_queuecalls rename index idx_queuecalls_queueid to idx_queue_queuecalls_queueid;""")
    op.execute("""alter table queue_queuecalls rename index idx_queuecalls_referenceid to idx_queue_queuecalls_referenceid;""")
    op.execute("""alter table queue_queuecalls rename index idx_queuecalls_reference_activeflow_id to idx_queue_queuecalls_reference_activeflow_id;""")
    op.execute("""alter table queue_queuecalls rename index idx_queuecalls_serviceagentid to idx_queue_queuecalls_serviceagentid;""")

    op.execute("""rename table queues to queue_queues;""")
    op.execute("""alter table queue_queues rename index idx_queues_customerid to idx_queue_queues_customerid;""")
    op.execute("""alter table queue_queues rename index idx_queues_flowid to idx_queue_queues_flowid;""")
    op.execute("""alter table queue_queues rename index idx_queues_confbridgeid to idx_queue_queues_confbridgeid;""")

    op.execute("""rename table recordings to call_recordings;""")
    op.execute("""alter table call_recordings rename index idx_recordings_tm_start to idx_call_recordings_tm_start;""")
    op.execute("""alter table call_recordings rename index idx_recordings_customer_id to idx_call_recordings_customer_id;""")
    op.execute("""alter table call_recordings rename index idx_recordings_owner_id to idx_call_recordings_owner_id;""")
    op.execute("""alter table call_recordings rename index idx_recordings_reference_id to idx_call_recordings_reference_id;""")
    op.execute("""alter table call_recordings rename index idx_recordings_recording_name to idx_call_recordings_recording_name;""")

    op.execute("""rename table routes to route_routes;""")
    op.execute("""alter table route_routes rename index idx_routes_customer_id to idx_route_routes_customer_id;""")
    op.execute("""alter table route_routes rename index idx_routes_provider_id to idx_route_routes_provider_id;""")

    op.execute("""rename table tags to tag_tags;""")
    op.execute("""alter table tag_tags rename index idx_tags_customerid to idx_tag_tags_customerid;""")
    op.execute("""alter table tag_tags rename index idx_tags_name to idx_tag_tags_name;""")

    op.execute("""rename table transcribes to transcribe_transcribes;""")
    op.execute("""alter table transcribe_transcribes rename index idx_transcribes_reference_id to idx_transcribe_transcribes_reference_id;""")
    op.execute("""alter table transcribe_transcribes rename index idx_transcribes_customerid to idx_transcribe_transcribes_customerid;""")

    op.execute("""rename table transcripts to transcribe_transcripts;""")
    op.execute("""alter table transcribe_transcripts rename index idx_transcripts_customerid to idx_transcribe_transcripts_customerid;""")
    op.execute("""alter table transcribe_transcripts rename index idx_transcripts_transcribe_id to idx_transcribe_transcripts_transcribe_id;""")



def downgrade():
    op.execute("""rename table flow_activeflows to activeflows;""")
    op.execute("""alter table activeflows rename index idx_flow_activeflows_customer_id to idx_activeflows_customer_id;""")
    op.execute("""alter table activeflows rename index idx_flow_activeflows_flow_id to idx_activeflows_flow_id;""")
    op.execute("""alter table activeflows rename index idx_flow_activeflows_reference_id to idx_activeflows_reference_id;""")

    op.execute("""rename table agent_agents to agents;""")
    op.execute("""alter table agents rename index idx_agent_agents_customerid to idx_agents_customerid;""")
    op.execute("""alter table agents rename index idx_agent_agents_username to idx_agents_username;""")

    op.execute("""rename table call_bridges to bridges;""")
    op.execute("""alter table bridges rename index idx_call_bridges_create to idx_bridges_create;""")
    op.execute("""alter table bridges rename index idx_call_bridges_reference_id to idx_bridges_reference_id;""")

    op.execute("""rename table call_calls to calls;""")
    op.execute("""alter table calls rename index idx_call_calls_customer_id to idx_calls_customer_id;""")
    op.execute("""alter table calls rename index idx_call_calls_owner_id to idx_calls_owner_id;""")
    op.execute("""alter table calls rename index idx_call_calls_channelid to idx_calls_channelid;""")
    op.execute("""alter table calls rename index idx_call_calls_flowid to idx_calls_flowid;""")
    op.execute("""alter table calls rename index idx_call_calls_create to idx_calls_create;""")
    op.execute("""alter table calls rename index idx_call_calls_hangup to idx_calls_hangup;""")
    op.execute("""alter table calls rename index idx_call_calls_source_target to idx_calls_source_target;""")
    op.execute("""alter table calls rename index idx_call_calls_destination_target to idx_calls_destination_target;""")
    op.execute("""alter table calls rename index idx_call_calls_external_media_id to idx_calls_external_media_id;""")
    op.execute("""alter table calls rename index idx_call_calls_groupcall_id to idx_calls_groupcall_id;""")

    op.execute("""rename table campaign_campaigncalls to campaigncalls;""")
    op.execute("""alter table campaigncalls rename index idx_campaign_campaigncalls_customer_id to idx_campaigncalls_customer_id;""")
    op.execute("""alter table campaigncalls rename index idx_campaign_campaigncalls_campaign_id to idx_campaigncalls_campaign_id;""")
    op.execute("""alter table campaigncalls rename index idx_campaign_campaigncalls_outdial_target_id to idx_campaigncalls_outdial_target_id;""")
    op.execute("""alter table campaigncalls rename index idx_campaign_campaigncalls_activeflow_id to idx_campaigncalls_activeflow_id;""")
    op.execute("""alter table campaigncalls rename index idx_campaign_campaigncalls_reference_id to idx_campaigncalls_reference_id;""")
    op.execute("""alter table campaigncalls rename index idx_campaign_campaigncalls_campaign_id_status to idx_campaigncalls_campaign_id_status;""")

    op.execute("""rename table campaign_campaigns to campaigns;""")
    op.execute("""alter table campaigns rename index idx_campaign_campaigns_customer_id to idx_campaigns_customer_id;""")
    op.execute("""alter table campaigns rename index idx_campaign_campaigns_flow_id to idx_campaigns_flow_id;""")
    op.execute("""alter table campaigns rename index idx_campaign_campaigns_outplan_id to idx_campaigns_outplan_id;""")
    op.execute("""alter table campaigns rename index idx_campaign_campaigns_outdial_id to idx_campaigns_outdial_id;""")
    op.execute("""alter table campaigns rename index idx_campaign_campaigns_queue_id to idx_campaigns_queue_id;""")

    op.execute("""rename table call_channels to channels;""")
    op.execute("""alter table channels rename index idx_call_channels_create to idx_channels_create;""")
    op.execute("""alter table channels rename index idx_call_channels_src_number to idx_channels_src_number;""")
    op.execute("""alter table channels rename index idx_call_channels_dst_number to idx_channels_dst_number;""")
    op.execute("""alter table channels rename index idx_call_channels_sip_call_id to idx_channels_sip_call_id;""")

    op.execute("""rename table chatbot_chatbotcalls to chatbotcalls;""")
    op.execute("""alter table chatbotcalls rename index idx_chatbot_chatbotcalls_customer_id to idx_chatbotcalls_customer_id;""")
    op.execute("""alter table chatbotcalls rename index idx_chatbot_chatbotcalls_chatbot_id to idx_chatbotcalls_chatbot_id;""")
    op.execute("""alter table chatbotcalls rename index idx_chatbot_chatbotcalls_reference_type to idx_chatbotcalls_reference_type;""")
    op.execute("""alter table chatbotcalls rename index idx_chatbot_chatbotcalls_reference_id to idx_chatbotcalls_reference_id;""")
    op.execute("""alter table chatbotcalls rename index idx_chatbot_chatbotcalls_transcribe_id to idx_chatbotcalls_transcribe_id;""")
    op.execute("""alter table chatbotcalls rename index idx_chatbot_chatbotcalls_create to idx_chatbotcalls_create;""")
    op.execute("""alter table chatbotcalls rename index idx_chatbot_chatbotcalls_activeflow_id to idx_chatbotcalls_activeflow_id;""")

    op.execute("""rename table chatbot_chatbots to chatbots;""")
    op.execute("""alter table chatbots rename index idx_chatbot_chatbots_create to idx_chatbots_create;""")
    op.execute("""alter table chatbots rename index idx_chatbot_chatbots_customer_id to idx_chatbots_customer_id;""")

    op.execute("""rename table chat_chatrooms to chatrooms;""")
    op.execute("""alter table chatrooms rename index idx_chat_chatrooms_customer_id to idx_chatrooms_customer_id;""")
    op.execute("""alter table chatrooms rename index idx_chat_chatrooms_chat_id to idx_chatrooms_chat_id;""")
    op.execute("""alter table chatrooms rename index idx_chat_chatrooms_owner_id to idx_chatrooms_owner_id;""")
    op.execute("""alter table chatrooms rename index idx_chat_chatrooms_chat_id_owner_id to idx_chatrooms_chat_id_owner_id;""")
    op.execute("""alter table chatrooms rename index idx_chat_chatrooms_room_owner_id to idx_chatrooms_room_owner_id;""")
    
    op.execute("""rename table chat_chats to chats;""")
    op.execute("""alter table chats rename index idx_chat_chats_customer_id to idx_chats_customer_id;""")
    op.execute("""alter table chats rename index idx_chat_chats_owner_id to idx_chats_owner_id;""")
    
    op.execute("""rename table call_confbridges to confbridges;""")
    op.execute("""alter table confbridges rename index idx_call_confbridges_create to idx_confbridges_create;""")
    op.execute("""alter table confbridges rename index idx_call_confbridges_customer_id to idx_confbridges_customer_id;""")
    op.execute("""alter table confbridges rename index idx_call_confbridges_bridge_id to idx_confbridges_bridge_id;""")
    
    op.execute("""rename table conference_conferencecalls to conferencecalls;""")
    op.execute("""alter table conferencecalls rename index idx_conference_conferencecalls_customer_id to idx_conferencecalls_customer_id;""")
    op.execute("""alter table conferencecalls rename index idx_conference_conferencecalls_conference_id to idx_conferencecalls_conference_id;""")
    op.execute("""alter table conferencecalls rename index idx_conference_conferencecalls_reference_id to idx_conferencecalls_reference_id;""")
    op.execute("""alter table conferencecalls rename index idx_conference_conferencecalls_create to idx_conferencecalls_create;""")

    op.execute("""rename table conference_conferences to conferences;""")
    op.execute("""alter table conferences rename index idx_conference_conferences_create to idx_conferences_create;""")
    op.execute("""alter table conferences rename index idx_conference_conferences_customer_id to idx_conferences_customer_id;""")
    op.execute("""alter table conferences rename index idx_conference_conferences_flow_id to idx_conferences_flow_id;""")
    op.execute("""alter table conferences rename index idx_conference_conferences_confbridge_id to idx_conferences_confbridge_id;""")

    op.execute("""rename table customer_customers to customers;""")
    
    op.execute("""rename table registrar_extensions to extensions;""")
    op.execute("""alter table extensions rename index idx_registrar_extensions_customerid to idx_extensions_customerid;""")
    op.execute("""alter table extensions rename index idx_registrar_extensions_extension to idx_extensions_extension;""")
    op.execute("""alter table extensions rename index idx_registrar_extensions_domain_name to idx_extensions_domain_name;""")
    op.execute("""alter table extensions rename index idx_registrar_extensions_username to idx_extensions_username;""")
    op.execute("""alter table extensions rename index idx_registrar_extensions_realm to idx_extensions_realm;""")

    op.execute("""rename table flow_flows to flows;""")
    op.execute("""alter table flows rename index idx_flows_customer_id to idx_customer_id;""")



    op.execute("""rename table call_groupcalls to groupcalls;""")
    op.execute("""alter table groupcalls rename index idx_call_groupcalls_customer_id to idx_groupcalls_customer_id;""")
    op.execute("""alter table groupcalls rename index idx_call_groupcalls_owner_id to idx_groupcalls_owner_id;""")

    op.execute("""rename table chat_messagechatrooms to messagechatrooms;""")
    op.execute("""alter table messagechatrooms rename index idx_chat_messagechatrooms_customer_id to idx_messagechatrooms_customer_id;""")
    op.execute("""alter table messagechatrooms rename index idx_chat_messagechatrooms_chatroom_id to idx_messagechatrooms_chatroom_id;""")
    op.execute("""alter table messagechatrooms rename index idx_chat_messagechatrooms_messagechat_id to idx_messagechatrooms_messagechat_id;""")

    op.execute("""rename table chat_messagechats to messagechats;""")
    op.execute("""alter table messagechats rename index idx_chat_messagechats_customer_id to idx_messagechats_customer_id;""")
    op.execute("""alter table messagechats rename index idx_chat_messagechats_chat_id to idx_messagechats_chat_id;""")

    op.execute("""rename table message_messages to messages;""")
    op.execute("""alter table messages rename index idx_message_messages_customerid to idx_messages_customerid;""")
    op.execute("""alter table messages rename index idx_message_messages_provider_name to idx_messages_provider_name;""")
    op.execute("""alter table messages rename index idx_message_messages_provider_reference_id to idx_messages_provider_reference_id;""")

    op.execute("""rename table number_numbers to numbers;""")
    op.execute("""alter table numbers rename index idx_number_numbers_number to idx_numbers_number;""")
    op.execute("""alter table numbers rename index idx_number_numbers_customerid to idx_numbers_customerid;""")
    op.execute("""alter table numbers rename index idx_number_numbers_call_flow_id to idx_numbers_call_flow_id;""")
    op.execute("""alter table numbers rename index idx_number_numbers_message_flow_id to idx_numbers_message_flow_id;""")
    op.execute("""alter table numbers rename index idx_number_numbers_provider_name to idx_numbers_provider_name;""")
    op.execute("""alter table numbers rename index idx_number_numbers_tm_renew to idx_numbers_tm_renew;""")

    op.execute("""rename table outdial_outdials to outdials;""")
    op.execute("""alter table outdials rename index idx_outdial_outdials_customer_id to idx_outdials_customer_id;""")
    op.execute("""alter table outdials rename index idx_outdial_outdials_campaign_id to idx_outdials_campaign_id;""")

    op.execute("""rename table outdial_outdialtargets to outdialtargets;""")
    op.execute("""alter table outdialtargets rename index idx_outdial_outdialtargets_outdial_id to idx_outdialtargets_outdial_id;""")

    # Drop table created in upgrade
    op.execute("drop table outdial_outdialtargetcalls")

    op.execute("""rename table campaign_outplans to outplans;""")
    op.execute("""alter table outplans rename index idx_campaign_outplans_customer_id to idx_outplans_customer_id;""")

    op.execute("""rename table route_providers to providers;""")
    
    op.execute("""rename table queue_queuecallreferences to queuecallreferences;""")

    op.execute("""rename table queue_queuecalls to queuecalls;""")
    op.execute("""alter table queuecalls rename index idx_queue_queuecalls_customerid to idx_queuecalls_customerid;""")
    op.execute("""alter table queuecalls rename index idx_queue_queuecalls_queueid to idx_queuecalls_queueid;""")
    op.execute("""alter table queuecalls rename index idx_queue_queuecalls_referenceid to idx_queuecalls_referenceid;""")
    op.execute("""alter table queuecalls rename index idx_queue_queuecalls_reference_activeflow_id to idx_queuecalls_reference_activeflow_id;""")
    op.execute("""alter table queuecalls rename index idx_queue_queuecalls_serviceagentid to idx_queuecalls_serviceagentid;""")

    op.execute("""rename table queue_queues to queues;""")
    op.execute("""alter table queues rename index idx_queue_queues_customerid to idx_queues_customerid;""")
    op.execute("""alter table queues rename index idx_queue_queues_flowid to idx_queues_flowid;""")
    op.execute("""alter table queues rename index idx_queue_queues_confbridgeid to idx_queues_confbridgeid;""")

    op.execute("""rename table call_recordings to recordings;""")
    op.execute("""alter table recordings rename index idx_call_recordings_tm_start to idx_recordings_tm_start;""")
    op.execute("""alter table recordings rename index idx_call_recordings_customer_id to idx_recordings_customer_id;""")
    op.execute("""alter table recordings rename index idx_call_recordings_owner_id to idx_recordings_owner_id;""")
    op.execute("""alter table recordings rename index idx_call_recordings_reference_id to idx_recordings_reference_id;""")
    op.execute("""alter table recordings rename index idx_call_recordings_recording_name to idx_recordings_recording_name;""")

    op.execute("""rename table route_routes to routes;""")
    op.execute("""alter table routes rename index idx_route_routes_customer_id to idx_routes_customer_id;""")
    op.execute("""alter table routes rename index idx_route_routes_provider_id to idx_routes_provider_id;""")

    op.execute("""rename table tag_tags to tags;""")
    op.execute("""alter table tags rename index idx_tag_tags_customerid to idx_tags_customerid;""")
    op.execute("""alter table tags rename index idx_tag_tags_name to idx_tags_name;""")

    op.execute("""rename table transcribe_transcribes to transcribes;""")
    op.execute("""alter table transcribes rename index idx_transcribe_transcribes_reference_id to idx_transcribes_reference_id;""")
    op.execute("""alter table transcribes rename index idx_transcribe_transcribes_customerid to idx_transcribes_customerid;""")

    op.execute("""rename table transcribe_transcripts to transcripts;""")
    op.execute("""alter table transcripts rename index idx_transcribe_transcripts_customerid to idx_transcripts_customerid;""")
    op.execute("""alter table transcripts rename index idx_transcribe_transcripts_transcribe_id to idx_transcripts_transcribe_id;""")

