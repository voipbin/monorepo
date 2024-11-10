"""all_update_index

Revision ID: 3ffdc562b708
Revises: 354328132eeb
Create Date: 2024-11-10 17:57:41.533504

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy import MetaData


# revision identifiers, used by Alembic.
revision = '3ffdc562b708'
down_revision = '354328132eeb'
branch_labels = None
depends_on = None


def upgrade():
    meta = MetaData(bind=op.get_bind())
    meta.reflect()

    # Iterate over all tables
    for table in meta.tables.values():
        for index in table.indexes:
            # Drop each index
            op.drop_index(index.name, table_name=table.name)

    # agent_agents
    op.execute("""create index idx_agent_agents_customer_id on agent_agents(customer_id);""")
    op.execute("""create index idx_agent_agents_username on agent_agents(username);""")
    
    # billing_accounts
    op.execute("""create index idx_billing_accounts_customer_id on billing_accounts(customer_id);""")
    op.execute("""create index idx_billing_accounts_create on billing_accounts(tm_create);""")
    
    # billing_billings
    op.execute("""create index idx_billing_billings_customer_id on billing_billings(customer_id);""")
    op.execute("""create index idx_billing_billings_account_id on billing_billings(account_id);""")
    op.execute("""create index idx_billing_billings_reference_id on billing_billings(reference_id);""")
    op.execute("""create index idx_billing_billings_create on billing_billings(tm_create);""")
    
    # call_bridges
    op.execute("""create index idx_call_bridges_create on call_bridges(tm_create);""")
    op.execute("""create index idx_call_bridges_reference_id on call_bridges(reference_id);""")
    
    # call_calls
    op.execute("""create index idx_call_calls_customer_id on call_calls(customer_id);""")
    op.execute("""create index idx_call_calls_owner_id on call_calls(owner_id);""")
    op.execute("""create index idx_call_calls_channel_id on call_calls(channel_id);""")
    op.execute("""create index idx_call_calls_flow_id on call_calls(flow_id);""")
    op.execute("""create index idx_call_calls_create on call_calls(tm_create);""")
    op.execute("""create index idx_call_calls_hangup on call_calls(tm_hangup);""")
    op.execute("""create index idx_call_calls_source_target on call_calls(source_target);""")
    op.execute("""create index idx_call_calls_destination_target on call_calls(destination_target);""")
    op.execute("""create index idx_call_calls_external_media_id on call_calls(external_media_id);""")
    op.execute("""create index idx_call_calls_groupcall_id on call_calls(groupcall_id);""")
    
    # call_channels
    op.execute("""create index idx_call_channels_create on call_channels(tm_create);""")
    op.execute("""create index idx_call_channels_src_number on call_channels(src_number);""")
    op.execute("""create index idx_call_channels_dst_number on call_channels(dst_number);""")
    op.execute("""create index idx_call_channels_sip_call_id on call_channels(sip_call_id);""")
    
    # call_confbridges
    op.execute("""create index idx_call_confbridges_create on call_confbridges(tm_create);""")
    op.execute("""create index idx_call_confbridges_customer_id on call_confbridges(customer_id);""")
    op.execute("""create index idx_call_confbridges_bridge_id on call_confbridges(bridge_id);""")
    
    # call_groupcalls
    op.execute("""create index idx_call_groupcalls_customer_id on call_groupcalls(customer_id);""")
    op.execute("""create index idx_call_groupcalls_owner_id on call_groupcalls(owner_id);""")
    
    # call_recordings
    op.execute("""create index idx_call_recordings_tm_start on call_recordings(tm_start);""")
    op.execute("""create index idx_call_recordings_customer_id on call_recordings(customer_id);""")
    op.execute("""create index idx_call_recordings_owner_id on call_recordings(owner_id);""")
    op.execute("""create index idx_call_recordings_reference_id on call_recordings(reference_id);""")
    op.execute("""create index idx_call_recordings_recording_name on call_recordings(recording_name);""")
    
    # campaign_campaigncalls
    op.execute("""create index idx_campaign_campaigncalls_customer_id on campaign_campaigncalls(customer_id);""")
    op.execute("""create index idx_campaign_campaigncalls_campaign_id on campaign_campaigncalls(campaign_id);""")
    op.execute("""create index idx_campaign_campaigncalls_outdial_target_id on campaign_campaigncalls(outdial_target_id);""")
    op.execute("""create index idx_campaign_campaigncalls_activeflow_id on campaign_campaigncalls(activeflow_id);""")
    op.execute("""create index idx_campaign_campaigncalls_reference_id on campaign_campaigncalls(reference_id);""")
    op.execute("""create index idx_campaign_campaigncalls_campaign_id_status on campaign_campaigncalls(campaign_id, status);""")
    
    # campaign_campaigns
    op.execute("""create index idx_campaign_campaigns_customer_id on campaign_campaigns(customer_id);""")
    op.execute("""create index idx_campaign_campaigns_flow_id on campaign_campaigns(flow_id);""")
    op.execute("""create index idx_campaign_campaigns_outplan_id on campaign_campaigns(outplan_id);""")
    op.execute("""create index idx_campaign_campaigns_outdial_id on campaign_campaigns(outdial_id);""")
    op.execute("""create index idx_campaign_campaigns_queue_id on campaign_campaigns(queue_id);""")
    
    # campaign_outplans
    op.execute("""create index idx_campaign_outplans_customer_id on campaign_outplans(customer_id);""")
    
    # chat_chatrooms
    op.execute("""create index idx_chat_chatrooms_customer_id on chat_chatrooms(customer_id);""")
    op.execute("""create index idx_chat_chatrooms_chat_id on chat_chatrooms(chat_id);""")
    op.execute("""create index idx_chat_chatrooms_owner_id on chat_chatrooms(owner_id);""")
    op.execute("""create index idx_chat_chatrooms_chat_id_owner_id on chat_chatrooms(chat_id, owner_id);""")
    op.execute("""create index idx_chat_chatrooms_room_owner_id on chat_chatrooms(room_owner_id);""")
    
    # chat_chats
    op.execute("""create index idx_chat_chats_customer_id on chat_chats(customer_id);""")
    op.execute("""create index idx_chat_chats_room_owner_id on chat_chats(room_owner_id);""")
    
    # chat_messagechatrooms
    op.execute("""create index idx_chat_messagechatrooms_customer_id on chat_messagechatrooms(customer_id);""")
    op.execute("""create index idx_chat_messagechatrooms_chatroom_id on chat_messagechatrooms(chatroom_id);""")
    op.execute("""create index idx_chat_messagechatrooms_messagechat_id on chat_messagechatrooms(messagechat_id);""")
    
    # chat_messagechats
    op.execute("""create index idx_chat_messagechats_customer_id on chat_messagechats(customer_id);""")
    op.execute("""create index idx_chat_messagechats_chat_id on chat_messagechats(chat_id);""")
    
    # chatbot_chatbotcalls
    op.execute("""create index idx_chatbot_chatbotcalls_customer_id on chatbot_chatbotcalls(customer_id);""")
    op.execute("""create index idx_chatbot_chatbotcalls_chatbot_id on chatbot_chatbotcalls(chatbot_id);""")
    op.execute("""create index idx_chatbot_chatbotcalls_reference_type on chatbot_chatbotcalls(reference_type);""")
    op.execute("""create index idx_chatbot_chatbotcalls_reference_id on chatbot_chatbotcalls(reference_id);""")
    op.execute("""create index idx_chatbot_chatbotcalls_transcribe_id on chatbot_chatbotcalls(transcribe_id);""")
    op.execute("""create index idx_chatbot_chatbotcalls_create on chatbot_chatbotcalls(tm_create);""")
    op.execute("""create index idx_chatbot_chatbotcalls_activeflow_id on chatbot_chatbotcalls(activeflow_id);""")
    
    # chatbot_chatbots
    op.execute("""create index idx_chatbot_chatbots_create on chatbot_chatbots(tm_create);""")
    op.execute("""create index idx_chatbot_chatbots_customer_id on chatbot_chatbots(customer_id);""")
    
    # conference_conferencecalls
    op.execute("""create index idx_conference_conferencecalls_customer_id on conference_conferencecalls(customer_id);""")
    op.execute("""create index idx_conference_conferencecalls_conference_id on conference_conferencecalls(conference_id);""")
    op.execute("""create index idx_conference_conferencecalls_reference_id on conference_conferencecalls(reference_id);""")
    op.execute("""create index idx_conference_conferencecalls_create on conference_conferencecalls(tm_create);""")
    
    # conference_conferences
    op.execute("""create index idx_conference_conferences_create on conference_conferences(tm_create);""")
    op.execute("""create index idx_conference_conferences_customer_id on conference_conferences(customer_id);""")
    op.execute("""create index idx_conference_conferences_flow_id on conference_conferences(flow_id);""")
    op.execute("""create index idx_conference_conferences_confbridge_id on conference_conferences(confbridge_id);""")
    
    # conversation_accounts
    op.execute("""create index idx_conversation_accounts_customer_id on conversation_accounts(customer_id);""")

    # conversation_conversations
    op.execute("""create index idx_conversation_conversations_customer_id on conversation_conversations(customer_id);""")
    op.execute("""create index idx_conversation_conversations_reference_type_reference_id on conversation_conversations(reference_type, reference_id);""")
    op.execute("""create index idx_conversation_conversations_owner_id on conversation_conversations(owner_id);""")
    
    # conversation_medias
    op.execute("""create index idx_conversation_medias_customer_id on conversation_medias(customer_id);""")
    
    # conversation_messages
    op.execute("""create index idx_conversation_messages_customer_id on conversation_messages(customer_id);""")
    op.execute("""create index idx_conversation_messages_reference_type_reference_id on conversation_messages(reference_type, reference_id);""")
    op.execute("""create index idx_conversation_messages_transaction_id on conversation_messages(transaction_id);""")
    
    # flow_activeflows
    op.execute("""create index idx_flow_activeflows_customer_id on flow_activeflows(customer_id);""")
    op.execute("""create index idx_flow_activeflows_flow_id on flow_activeflows(flow_id);""")
    op.execute("""create index idx_flow_activeflows_reference_id on flow_activeflows(reference_id);""")
    
    # flow_flows
    op.execute("""create index idx_flow_flows_customer_id on flow_flows(customer_id);""")
    
    # message_messages
    op.execute("""create index idx_message_messages_customer_id on message_messages(customer_id);""")
    op.execute("""create index idx_message_messages_provider_name on message_messages(provider_name);""")
    op.execute("""create index idx_message_messages_provider_reference_id on message_messages(provider_reference_id);""")
    
    # number_numbers
    op.execute("""create index idx_number_numbers_number on number_numbers(number);""")
    op.execute("""create index idx_number_numbers_customer_id on number_numbers(customer_id);""")
    op.execute("""create index idx_number_numbers_call_flow_id on number_numbers(call_flow_id);""")
    op.execute("""create index idx_number_numbers_message_flow_id on number_numbers(message_flow_id);""")
    op.execute("""create index idx_number_numbers_provider_name on number_numbers(provider_name);""")
    op.execute("""create index idx_number_numbers_tm_renew on number_numbers(tm_renew);""")
    
    # outdial_outdials
    op.execute("""create index idx_outdial_outdials_customer_id on outdial_outdials(customer_id);""")
    op.execute("""create index idx_outdial_outdials_campaign_id on outdial_outdials(campaign_id);""")
    
    # outdial_outdialtargetcalls
    op.execute("""create index idx_outdial_outdialtargetcalls_customer_id on outdial_outdialtargetcalls(customer_id);""")
    op.execute("""create index idx_outdial_outdialtargetcalls_campaign_id on outdial_outdialtargetcalls(campaign_id);""")
    op.execute("""create index idx_outdial_outdialtargetcalls_outdial_target_id on outdial_outdialtargetcalls(outdial_target_id);""")
    op.execute("""create index idx_outdial_outdialtargetcalls_activeflow_id on outdial_outdialtargetcalls(activeflow_id);""")
    op.execute("""create index idx_outdial_outdialtargetcalls_reference_id on outdial_outdialtargetcalls(reference_id);""")
    
    # outdial_outdialtargets
    op.execute("""create index idx_outdial_outdialtargets_outdial_id on outdial_outdialtargets(outdial_id);""")
    
    # queue_queuecalls
    op.execute("""create index idx_queue_queuecalls_customer_id on queue_queuecalls(customer_id);""")
    op.execute("""create index idx_queue_queuecalls_queue_id on queue_queuecalls(queue_id);""")
    op.execute("""create index idx_queue_queuecalls_reference_id on queue_queuecalls(reference_id);""")
    op.execute("""create index idx_queue_queuecalls_reference_activeflow_id on queue_queuecalls(reference_activeflow_id);""")
    op.execute("""create index idx_queue_queuecalls_service_agent_id on queue_queuecalls(service_agent_id);""")
    
    # queue_queues
    op.execute("""create index idx_queue_queues_customer_id on queue_queues(customer_id);""")
    
    # registrar_extensions
    op.execute("""create index idx_registrar_extensions_customer_id on registrar_extensions(customer_id);""")
    op.execute("""create index idx_registrar_extensions_extension on registrar_extensions(extension);""")
    op.execute("""create index idx_registrar_extensions_domain_name on registrar_extensions(domain_name);""")
    op.execute("""create index idx_registrar_extensions_username on registrar_extensions(username);""")
    op.execute("""create index idx_registrar_extensions_realm on registrar_extensions(realm);""")
    
    # registrar_sip_auths
    op.execute("""create index idx_registrar_sip_auths_realm on registrar_sip_auths(realm);""")
    
    # registrar_trunks
    op.execute("""create index idx_registrar_trunks_customer_id on registrar_trunks(customer_id);""")
    op.execute("""create index idx_registrar_trunks_domain_name on registrar_trunks(domain_name);""")
    op.execute("""create index idx_registrar_trunks_realm on registrar_trunks(realm);""")
    
    # route_routes
    op.execute("""create index idx_route_routes_customer_id on route_routes(customer_id);""")
    op.execute("""create index idx_route_routes_provider_id on route_routes(provider_id);""")
    
    # storage_accounts
    op.execute("""create index idx_storage_accounts_customer_id on storage_accounts(customer_id);""")
    
    # storage_files
    op.execute("""create index idx_storage_files_customer_id on storage_files(customer_id);""")
    op.execute("""create index idx_storage_files_account_id on storage_files(account_id);""")
    op.execute("""create index idx_storage_files_owner_id on storage_files(owner_id);""")
    op.execute("""create index idx_storage_files_reference_id on storage_files(reference_id);""")
    
    # tag_tags
    op.execute("""create index idx_tag_tags_customer_id on tag_tags(customer_id);""")
    op.execute("""create index idx_tag_tags_name on tag_tags(name);""")
    
    # transcribe_transcribes
    op.execute("""create index idx_transcribe_transcribes_reference_id on transcribe_transcribes(reference_id);""")
    op.execute("""create index idx_transcribe_transcribes_customer_id on transcribe_transcribes(customer_id);""")
    
    # transcribe_transcripts
    op.execute("""create index idx_transcribe_transcripts_customer_id on transcribe_transcripts(customer_id);""")
    op.execute("""create index idx_transcribe_transcripts_transcribe_id on transcribe_transcripts(transcribe_id);""")



def downgrade():
    meta = MetaData(bind=op.get_bind())
    meta.reflect()

    # Iterate over all tables
    for table in meta.tables.values():
        for index in table.indexes:
            # Drop each index
            op.drop_index(index.name, table_name=table.name)
