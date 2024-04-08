import React from 'react'

// dashboard ----------------------------------------------------------------
const Dashboard = React.lazy(() => import('./views/dashboard/Dashboard'))

// resource ----------------------------------------------------------------
// customers
const CustomersList = React.lazy(() => import('./views/customers/customers_list'))
const CustomersDetail = React.lazy(() => import('./views/customers/customers_detail'))
const CustomersCreate = React.lazy(() => import('./views/customers/customers_create'))

// calls
const CallsList = React.lazy(() => import('./views/calls/calls_list'))
const CallsDetail = React.lazy(() => import('./views/calls/calls_detail'))
const CallsCreate = React.lazy(() => import('./views/calls/calls_create'))

// currentcall
const CurrentcallDetail = React.lazy(() => import('./views/calls/currentcall_detail'))

// groupcalls
const GroupcallsList = React.lazy(() => import('./views/calls/groupcalls_list'))
const GroupcallsDetail = React.lazy(() => import('./views/calls/groupcalls_detail'))
const GroupcallsCreate = React.lazy(() => import('./views/calls/groupcalls_create'))

// queues
const QueuesList = React.lazy(() => import('./views/queues/queues_list'))
const QueuesDetail = React.lazy(() => import('./views/queues/queues_detail'))
const QueuesCreate = React.lazy(() => import('./views/queues/queues_create'))

// queuecalls
const QueuecallsList = React.lazy(() => import('./views/queues/queuecalls_list'))
const QueuecallsDetail = React.lazy(() => import('./views/queues/queuecalls_detail'))

// numbers
const NumbersActiveList = React.lazy(() => import('./views/numbers/active_list'))
const NumbersActiveDetail = React.lazy(() => import('./views/numbers/active_detail'))
const NumbersBuyList = React.lazy(() => import('./views/numbers/buy_list'))
const NumbersBuyCreate = React.lazy(() => import('./views/numbers/buy_create'))

// actiongraph
const Actiongraph = React.lazy(() => import('./views/actiongraph/actiongraph'))

// flows
const FlowsList = React.lazy(() => import('./views/flows/flows_list'))
const FlowsDetail = React.lazy(() => import('./views/flows/flows_detail'))
const FlowsCreate = React.lazy(() => import('./views/flows/flows_create'))
const FlowsActiveflowsList = React.lazy(() => import('./views/flows/activeflows_list'))
const FlowsActiveflowsDetail = React.lazy(() => import('./views/flows/activeflows_detail'))
const FlowsActiveflowsCreate = React.lazy(() => import('./views/flows/activeflows_create'))

// agents
const AgentsList = React.lazy(() => import('./views/agents/agents_list'))
const AgentsDetail = React.lazy(() => import('./views/agents/agents_detail'))
const AgentsCreate = React.lazy(() => import('./views/agents/agents_create'))
const AgentsProfile = React.lazy(() => import('./views/agents/agents_profile'))

// billings
const BillingsList = React.lazy(() => import('./views/billings/billings_list'))
const BillingsDetail = React.lazy(() => import('./views/billings/billings_detail'))

// billing accounts
const BillingAccountsList = React.lazy(() => import('./views/billing_accounts/billing_accounts_list'))
const BillingAccountsDetail = React.lazy(() => import('./views/billing_accounts/billing_accounts_detail'))
const BillingAccountsCreate = React.lazy(() => import('./views/billing_accounts/billing_accounts_create'))

// conferences
const ConferencesList = React.lazy(() => import('./views/conferences/conferences_list'))
const ConferencesDetail = React.lazy(() => import('./views/conferences/conferences_detail'))
const ConferencesCreate = React.lazy(() => import('./views/conferences/conferences_create'))

// conferencecalls
const ConferencecallsList = React.lazy(() => import('./views/conferences/conferencecalls_list'))
const ConferencecallsDetail = React.lazy(() => import('./views/conferences/conferencecalls_detail'))

// chatbots
const ChatbotsList = React.lazy(() => import('./views/chatbots/chatbots_list'))
const ChatbotsDetail = React.lazy(() => import('./views/chatbots/chatbots_detail'))
const ChatbotsCreate = React.lazy(() => import('./views/chatbots/chatbots_create'))

// chatbotcalls
const Chatbotcalls = React.lazy(() => import('./views/chatbots/chatbotcalls'))

// conversations
const ConversationaccountsList = React.lazy(() => import('./views/conversations/accounts_list'))
const ConversationaccountsDetail = React.lazy(() => import('./views/conversations/accounts_detail'))
const ConversationaccountsCreate = React.lazy(() => import('./views/conversations/accounts_create'))
const ConversationsList = React.lazy(() => import('./views/conversations/conversations_list'))
const ConversationsDetail = React.lazy(() => import('./views/conversations/conversations_detail'))
const ConversationsCreate = React.lazy(() => import('./views/conversations/conversations_create'))
const ConversationmessagesList = React.lazy(() => import('./views/conversations/conversationmessages_list'))

// chats
const ChatsList = React.lazy(() => import('./views/chats/chats_list'))
const ChatsDetail = React.lazy(() => import('./views/chats/chats_detail'))
const ChatsCreate = React.lazy(() => import('./views/chats/chats_create'))
const ChatroomsList = React.lazy(() => import('./views/chats/rooms_list'))
const ChatroomsDetail = React.lazy(() => import('./views/chats/rooms_detail'))
const ChatroomsCreate = React.lazy(() => import('./views/chats/rooms_create'))
const ChatroommessagesList = React.lazy(() => import('./views/chats/roommessages_list'))

// messages
const MessagesList = React.lazy(() => import('./views/messages/messages_list'))
const MessagesDetail = React.lazy(() => import('./views/messages/messages_detail'))
const MessagesCreate = React.lazy(() => import('./views/messages/messages_create'))

// trunks
const TrunksList = React.lazy(() => import('./views/trunks/trunks_list'))
const TrunksDetail = React.lazy(() => import('./views/trunks/trunks_detail'))
const TrunksCreate = React.lazy(() => import('./views/trunks/trunks_create'))

// extensions
const ExtensionsList = React.lazy(() => import('./views/extensions/extensions_list'))
const ExtensionsDetail = React.lazy(() => import('./views/extensions/extensions_detail'))
const ExtensionsCreate = React.lazy(() => import('./views/extensions/extensions_create'))

// providers
const ProvidersList = React.lazy(() => import('./views/providers/providers_list'))
const ProvidersDetail = React.lazy(() => import('./views/providers/providers_detail'))
const ProvidersCreate = React.lazy(() => import('./views/providers/providers_create'))

// routes
const RoutesList = React.lazy(() => import('./views/routes/routes_list'))
const RoutesDetail = React.lazy(() => import('./views/routes/routes_detail'))
const RoutesCreate = React.lazy(() => import('./views/routes/routes_create'))

// tags
const TagsList = React.lazy(() => import('./views/tags/tags_list'))
const TagsDetail = React.lazy(() => import('./views/tags/tags_detail'))
const TagsCreate = React.lazy(() => import('./views/tags/tags_create'))

// recordings
const RecordingsList = React.lazy(() => import('./views/recordings/recordings_list'))
const RecordingsDetail = React.lazy(() => import('./views/recordings/recordings_detail'))

// transcripts
const TranscribesList = React.lazy(() => import('./views/transcribes/transcribes_list'))
const TranscribesDetail = React.lazy(() => import('./views/transcribes/transcribes_detail'))
const TranscriptsList = React.lazy(() => import('./views/transcribes/transcripts_list'))



//
// outbound campaign ----------------------------------------------------------------

// campaigncalls
const CampaigncallsList = React.lazy(() => import('./views/campaigns/campaigncalls_list'))
const CampaigncallsDetail = React.lazy(() => import('./views/campaigns/campaigncalls_detail'))


// campaigns
const CampaignsList = React.lazy(() => import('./views/campaigns/campaigns_list'))
const CampaignsDetail = React.lazy(() => import('./views/campaigns/campaigns_detail'))
const CampaignsCreate = React.lazy(() => import('./views/campaigns/campaigns_create'))

// outdials
const OutdialsList = React.lazy(() => import('./views/outdials/outdials_list'))
const OutdialsDetail = React.lazy(() => import('./views/outdials/outdials_detail'))
const OutdialsCreate = React.lazy(() => import('./views/outdials/outdials_create'))

// outdialtargets
const OutdialtargetsList = React.lazy(() => import('./views/outdials/outdialtargets_list'))
const OutdialtargetsDetail = React.lazy(() => import('./views/outdials/outdialtargets_detail'))
const OutdialtargetsCreate = React.lazy(() => import('./views/outdials/outdialtargets_create'))

// outplans
const OutplansList = React.lazy(() => import('./views/outplans/outplans_list'))
const OutplansDetail = React.lazy(() => import('./views/outplans/outplans_detail'))
const OutplansCreate = React.lazy(() => import('./views/outplans/outplans_create'))

const routes = [
  { path: '/', exact: true, name: 'Home', element: Dashboard },
  { path: '/dashboard', name: 'Dashboard', element: Dashboard },

  { path: '/resources/calls/calls_list', name: 'CallsList', element: CallsList },
  { path: '/resources/calls/calls_detail/:id', name: 'CallsDetail', element: CallsDetail },
  { path: '/resources/calls/calls_create', name: 'CallsCreate', element: CallsCreate },
  { path: '/resources/calls/groupcalls_list', name: 'GroupcallsList', element: GroupcallsList },
  { path: '/resources/calls/groupcalls_detail/:id', name: 'GroupcallsDetail', element: GroupcallsDetail },
  { path: '/resources/calls/groupcalls_create', name: 'GroupcallsCreate', element: GroupcallsCreate },
  { path: '/resources/calls/currentcall_detail', name: 'CurrentcallDetail', element: CurrentcallDetail },

  { path: '/resources/customers/customers_list', name: 'CustomersList', element: CustomersList },
  { path: '/resources/customers/customers_create', name: 'CustomersCreate', element: CustomersCreate },
  { path: '/resources/customers/customers_detail/:id', name: 'CustomersDetail', element: CustomersDetail },

  { path: '/resources/queues/queues_list', name: 'Queues', element: QueuesList },
  { path: '/resources/queues/queues_create', name: 'QueuesCreate', element: QueuesCreate },
  { path: '/resources/queues/queues_detail/:id', name: 'QueuesDetail', element: QueuesDetail },

  { path: '/resources/queues/queuecalls_list', name: 'QueuecallsList', element: QueuecallsList },
  { path: '/resources/queues/queuecalls_detail/:id', name: 'QueuecallsDetail', element: QueuecallsDetail },

  { path: '/resources/numbers/active_list', name: 'ActiveList', element: NumbersActiveList },
  { path: '/resources/numbers/active_detail/:id', name: 'ActiveDetail', element: NumbersActiveDetail },
  { path: '/resources/numbers/buy_list', name: 'BuyList', element: NumbersBuyList },
  { path: '/resources/numbers/buy_create/:id', name: 'BuyCreate', element: NumbersBuyCreate },

  { path: '/resources/actiongraphs/actiongraph', name: 'Actiongraph', element: Actiongraph },

  { path: '/resources/flows/flows_list', name: 'FlowsList', element: FlowsList },
  { path: '/resources/flows/flows_create', name: 'FlowsCreate', element: FlowsCreate },
  { path: '/resources/flows/flows_detail/:id', name: 'FlowsDetail', element: FlowsDetail },
  { path: '/resources/flows/activeflows_list', name: 'ActiveflowsList', element: FlowsActiveflowsList },
  { path: '/resources/flows/activeflows_create', name: 'ActiveflowsCreate', element: FlowsActiveflowsCreate },
  { path: '/resources/flows/activeflows_detail/:id', name: 'ActiveflowsDetail', element: FlowsActiveflowsDetail },

  { path: '/resources/agents/agents_list', name: 'AgentsList', element: AgentsList },
  { path: '/resources/agents/agents_create', name: 'AgentsCreate', element: AgentsCreate },
  { path: '/resources/agents/agents_detail/:id', name: 'AgentsDetail', element: AgentsDetail },
  { path: '/resources/agents/agents_profile', name: 'AgentsProfile', element: AgentsProfile },

  { path: '/resources/billings/billings_list', name: 'BillingsList', element: BillingsList },
  { path: '/resources/billings/billings_detail/:id', name: 'BillingsDetail', element: BillingsDetail },

  { path: '/resources/billing_accounts/billing_accounts_list', name: 'BillingAccountsList', element: BillingAccountsList },
  { path: '/resources/billing_accounts/billing_accounts_create', name: 'BillingAccountsCreate', element: BillingAccountsCreate },
  { path: '/resources/billing_accounts/billing_accounts_detail/:id', name: 'BillingAccountsDetail', element: BillingAccountsDetail },

  { path: '/resources/conferences/conferences_list', name: 'ConferencesList', element: ConferencesList },
  { path: '/resources/conferences/conferences_create', name: 'ConferencesCreate', element: ConferencesCreate },
  { path: '/resources/conferences/conferences_detail/:id', name: 'ConferencesDetail', element: ConferencesDetail },

  { path: '/resources/conferences/conferencecalls_list', name: 'ConferencecallsList', element: ConferencecallsList },
  { path: '/resources/conferences/conferencecalls_detail/:id', name: 'ConferencecallsDetail', element: ConferencecallsDetail },

  { path: '/resources/chatbots/chatbots_list', name: 'ChatbotsList', element: ChatbotsList },
  { path: '/resources/chatbots/chatbots_create', name: 'ChatbotsCreate', element: ChatbotsCreate },
  { path: '/resources/chatbots/chatbots_detail/:id', name: 'ChatbotsDetail', element: ChatbotsDetail },

  { path: '/resources/chatbots/chatbotcalls_list', name: 'Chatbotcalls', element: Chatbotcalls },

  { path: '/resources/conversations/accounts_list', name: 'ConversationaccountsList', element: ConversationaccountsList },
  { path: '/resources/conversations/accounts_create', name: 'ConversationaccountsCreate', element: ConversationaccountsCreate },
  { path: '/resources/conversations/accounts_detail/:id', name: 'ConversationaccountsDetail', element: ConversationaccountsDetail },

  { path: '/resources/conversations/conversations_list', name: 'ConversationsList', element: ConversationsList },
  { path: '/resources/conversations/conversations_create', name: 'ConversationsCreate', element: ConversationsCreate },
  { path: '/resources/conversations/conversations_detail/:id', name: 'ConversationsDetail', element: ConversationsDetail },

  { path: '/resources/conversations/:conversation_id/messages_list', name: 'ConversationmessagesList', element: ConversationmessagesList },

  { path: '/resources/chats/chats_list', name: 'ChatsList', element: ChatsList },
  { path: '/resources/chats/chats_create', name: 'ChatsCreate', element: ChatsCreate },
  { path: '/resources/chats/chats_detail/:id', name: 'ChatsDetail', element: ChatsDetail },

  { path: '/resources/chats/rooms_list', name: 'ChatroomsList', element: ChatroomsList },
  { path: '/resources/chats/rooms_create', name: 'ChatroomsCreate', element: ChatroomsCreate },
  { path: '/resources/chats/rooms_detail/:id', name: 'ChatroomsDetail', element: ChatroomsDetail },

  { path: '/resources/chats/:room_id/messages_list', name: 'ChatroommessagesList', element: ChatroommessagesList },

  { path: '/resources/messages/messages_list', name: 'MessagesList', element: MessagesList },
  { path: '/resources/messages/messages_create', name: 'MessagesCreate', element: MessagesCreate },
  { path: '/resources/messages/messages_detail/:id', name: 'MessagesDetail', element: MessagesDetail },

  { path: '/resources/trunks/trunks_list', name: 'TrunksList', element: TrunksList },
  { path: '/resources/trunks/trunks_create', name: 'TrunksCreate', element: TrunksCreate },
  { path: '/resources/trunks/trunks_detail/:id', name: 'TrunksDetail', element: TrunksDetail },

  { path: '/resources/extensions/extensions_list', name: 'ExtensionsList', element: ExtensionsList },
  { path: '/resources/extensions/extensions_create', name: 'ExtensionsCreate', element: ExtensionsCreate },
  { path: '/resources/extensions/extensions_detail/:id', name: 'ExtensionsDetail', element: ExtensionsDetail },

  { path: '/resources/providers/providers_list', name: 'ProvidersList', element: ProvidersList },
  { path: '/resources/providers/providers_create', name: 'ProvidersCreate', element: ProvidersCreate },
  { path: '/resources/providers/providers_detail/:id', name: 'ProvidersDetail', element: ProvidersDetail },

  { path: '/resources/routes/routes_list', name: 'RoutesList', element: RoutesList },
  { path: '/resources/routes/routes_create/:id', name: 'RoutesCreate', element: RoutesCreate },
  { path: '/resources/routes/routes_detail/:id', name: 'RoutesDetail', element: RoutesDetail },

  { path: '/resources/tags/tags_list', name: 'TagsList', element: TagsList },
  { path: '/resources/tags/tags_create', name: 'TagsCreate', element: TagsCreate },
  { path: '/resources/tags/tags_detail/:id', name: 'TagsDetail', element: TagsDetail },

  { path: '/resources/recordings/recordings_list', name: 'RecordingsList', element: RecordingsList },
  { path: '/resources/recordings/recordings_detail/:id', name: 'RecordingsDetail', element: RecordingsDetail },

  { path: '/resources/transcribes/transcribes_list', name: 'TranscribesList', element: TranscribesList },
  { path: '/resources/transcribes/transcribes_detail/:id', name: 'TranscribesDetail', element: TranscribesDetail },

  { path: '/resources/transcribes/:id/transcripts_list', name: 'TranscriptsList', element: TranscriptsList },


  //
  // outbound campaign ------------------------------------------------------------

  { path: '/resources/campaigns/campaigns_list', name: 'CampaignsList', element: CampaignsList },
  { path: '/resources/campaigns/campaigns_create', name: 'CampaignsCreate', element: CampaignsCreate },
  { path: '/resources/campaigns/campaigns_detail/:id', name: 'CampaignsDetail', element: CampaignsDetail },

  { path: '/resources/campaigns/campaigncalls_list', name: 'CampaigncallsList', element: CampaigncallsList },
  { path: '/resources/campaigns/campaigncalls_detail/:id', name: 'CampaigncallsDetail', element: CampaigncallsDetail },

  { path: '/resources/outdials/outdials_list', name: 'OutdialsList', element: OutdialsList },
  { path: '/resources/outdials/outdials_create', name: 'OutdialsCreate', element: OutdialsCreate },
  { path: '/resources/outdials/outdials_detail/:id', name: 'OutdialsDetail', element: OutdialsDetail },

  { path: '/resources/outdials/:outdial_id/outdialtargets_list', name: 'OutdialtargetsList', element: OutdialtargetsList },
  { path: '/resources/outdials/:outdial_id/outdialtargets_create', name: 'OutdialtargetsCreate', element: OutdialtargetsCreate },
  { path: '/resources/outdials/:outdial_id/outdialtargets_detail/:id', name: 'OutdialtargetsDetail', element: OutdialtargetsDetail },

  { path: '/resources/outplans/outplans_list', name: 'OutplansList', element: OutplansList },
  { path: '/resources/outplans/outplans_create', name: 'OutplansCreate', element: OutplansCreate },
  { path: '/resources/outplans/outplans_detail/:id', name: 'OutplansDetail', element: OutplansDetail },

]

export default routes
