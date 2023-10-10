import React from 'react'

// activity
const MessagesList = React.lazy(() => import('./views/messages/messages_list'))
const MessagesDetail = React.lazy(() => import('./views/messages/messages_detail'))

// resources
// customers
const CustomersList = React.lazy(() => import('./views/customers/customers_list'))
const CustomersDetail = React.lazy(() => import('./views/customers/customers_detail'))
const CustomersCreate = React.lazy(() => import('./views/customers/customers_create'))


// calls
const CallsList = React.lazy(() => import('./views/calls/calls_list'))
const CallsDetail = React.lazy(() => import('./views/calls/calls_detail'))
const CallsCreate = React.lazy(() => import('./views/calls/calls_create'))

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

// flowgraph
const Flowgraph = React.lazy(() => import('./views/flowgraph/flowgraph'))

// flows
const FlowsList = React.lazy(() => import('./views/flows/flows_list'))
const FlowsDetail = React.lazy(() => import('./views/flows/flows_detail'))
const FlowsCreate = React.lazy(() => import('./views/flows/flows_create'))
const FlowsActiveflowsList = React.lazy(() => import('./views/flows/activeflows_list'))
const FlowsActiveflowsDetail = React.lazy(() => import('./views/flows/activeflows_detail'))



// agents
const AgentsList = React.lazy(() => import('./views/agents/agents_list'))
const AgentsDetail = React.lazy(() => import('./views/agents/agents_detail'))
const AgentsCreate = React.lazy(() => import('./views/agents/agents_create'))

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


// campaigncalls
const Campaigncalls = React.lazy(() => import('./views/campaigns/campaigncalls'))

// chatbots
const ChatbotsList = React.lazy(() => import('./views/chatbots/chatbots_list'))
const ChatbotsDetail = React.lazy(() => import('./views/chatbots/chatbots_detail'))
const ChatbotsCreate = React.lazy(() => import('./views/chatbots/chatbots_create'))

// chatbotcalls
const Chatbotcalls = React.lazy(() => import('./views/chatbots/chatbotcalls'))

// chats
const ChatsList = React.lazy(() => import('./views/chats/chats_list'))
const ChatsDetail = React.lazy(() => import('./views/chats/chats_detail'))
const ChatsCreate = React.lazy(() => import('./views/chats/chats_create'))

// trunks
const TrunksList = React.lazy(() => import('./views/trunks/trunks_list'))
const TrunksDetail = React.lazy(() => import('./views/trunks/trunks_detail'))
const TrunksCreate = React.lazy(() => import('./views/trunks/trunks_create'))

// extensions
const ExtensionsList = React.lazy(() => import('./views/extensions/extensions_list'))
const ExtensionsDetail = React.lazy(() => import('./views/extensions/extensions_detail'))
const ExtensionsCreate = React.lazy(() => import('./views/extensions/extensions_create'))




const Dashboard = React.lazy(() => import('./views/dashboard/Dashboard'))
const Colors = React.lazy(() => import('./views/theme/colors/Colors'))
const Typography = React.lazy(() => import('./views/theme/typography/Typography'))

// Base
const Accordion = React.lazy(() => import('./views/base/accordion/Accordion'))
const Breadcrumbs = React.lazy(() => import('./views/base/breadcrumbs/Breadcrumbs'))
const Cards = React.lazy(() => import('./views/base/cards/Cards'))
const Carousels = React.lazy(() => import('./views/base/carousels/Carousels'))
const Collapses = React.lazy(() => import('./views/base/collapses/Collapses'))
const ListGroups = React.lazy(() => import('./views/base/list-groups/ListGroups'))
const Navs = React.lazy(() => import('./views/base/navs/Navs'))
const Paginations = React.lazy(() => import('./views/base/paginations/Paginations'))
const Placeholders = React.lazy(() => import('./views/base/placeholders/Placeholders'))
const Popovers = React.lazy(() => import('./views/base/popovers/Popovers'))
const Progress = React.lazy(() => import('./views/base/progress/Progress'))
const Spinners = React.lazy(() => import('./views/base/spinners/Spinners'))
const Tables = React.lazy(() => import('./views/base/tables/Tables'))
const Tooltips = React.lazy(() => import('./views/base/tooltips/Tooltips'))

// Buttons
const Buttons = React.lazy(() => import('./views/buttons/buttons/Buttons'))
const ButtonGroups = React.lazy(() => import('./views/buttons/button-groups/ButtonGroups'))
const Dropdowns = React.lazy(() => import('./views/buttons/dropdowns/Dropdowns'))

//Forms
const ChecksRadios = React.lazy(() => import('./views/forms/checks-radios/ChecksRadios'))
const FloatingLabels = React.lazy(() => import('./views/forms/floating-labels/FloatingLabels'))
const FormControl = React.lazy(() => import('./views/forms/form-control/FormControl'))
const InputGroup = React.lazy(() => import('./views/forms/input-group/InputGroup'))
const Layout = React.lazy(() => import('./views/forms/layout/Layout'))
const Range = React.lazy(() => import('./views/forms/range/Range'))
const Select = React.lazy(() => import('./views/forms/select/Select'))
const Validation = React.lazy(() => import('./views/forms/validation/Validation'))

const Charts = React.lazy(() => import('./views/charts/Charts'))

// Icons
const CoreUIIcons = React.lazy(() => import('./views/icons/coreui-icons/CoreUIIcons'))
const Flags = React.lazy(() => import('./views/icons/flags/Flags'))
const Brands = React.lazy(() => import('./views/icons/brands/Brands'))

// Notifications
const Alerts = React.lazy(() => import('./views/notifications/alerts/Alerts'))
const Badges = React.lazy(() => import('./views/notifications/badges/Badges'))
const Modals = React.lazy(() => import('./views/notifications/modals/Modals'))
const Toasts = React.lazy(() => import('./views/notifications/toasts/Toasts'))

const Widgets = React.lazy(() => import('./views/widgets/Widgets'))

const routes = [
  { path: '/', exact: true, name: 'Home' },
  { path: '/dashboard', name: 'Dashboard', element: Dashboard },

  { path: '/activity', name: 'Activity', element: Colors, exact: true },
  { path: '/activity/calls', name: 'Calls', element: CallsList },
  { path: '/activity/calls/:id', name: 'CallsDetail', element: CallsDetail },

  { path: '/activity/activeflows', name: 'Activeflows', element: FlowsActiveflowsList },
  { path: '/activity/activeflows/:id', name: 'ActiveflowsDetail', element: FlowsActiveflowsDetail },

  { path: '/activity/messages', name: 'MessagesList', element: MessagesList },
  { path: '/activity/messages/:id', name: 'MessagesDetail', element: MessagesDetail },


  { path: '/resources', name: 'Resources', element: Colors, exact: true },

  { path: '/resources/calls/calls_list', name: 'CallsList', element: CallsList },
  { path: '/resources/calls/calls_detail/:id', name: 'CallsDetail', element: CallsDetail },
  { path: '/resources/calls/calls_create', name: 'CallsCreate', element: CallsCreate },
  { path: '/resources/calls/groupcalls_list', name: 'GroupcallsList', element: GroupcallsList },
  { path: '/resources/calls/groupcalls_detail/:id', name: 'GroupcallsDetail', element: GroupcallsDetail },
  { path: '/resources/calls/groupcalls_create', name: 'GroupcallsCreate', element: GroupcallsCreate },


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

  { path: '/resources/flowgraph/flowgraph', name: 'Flowgraph', element: Flowgraph },

  { path: '/resources/flows/flows_list', name: 'FlowsList', element: FlowsList },
  { path: '/resources/flows/flows_create', name: 'FlowsCreate', element: FlowsCreate },
  { path: '/resources/flows/flows_detail/:id', name: 'FlowsDetail', element: FlowsDetail },
  { path: '/resources/flows/activeflows_list', name: 'Activeflows', element: FlowsActiveflowsList },
  { path: '/resources/flows/activeflows_detail/:id', name: 'ActiveflowsDetail', element: FlowsActiveflowsDetail },



  { path: '/resources/agents/agents_list', name: 'AgentsList', element: AgentsList },
  { path: '/resources/agents/agents_create', name: 'AgentsCreate', element: AgentsCreate },
  { path: '/resources/agents/agents_detail/:id', name: 'AgentsDetail', element: AgentsDetail },

  { path: '/resources/billing_accounts/billing_accounts_list', name: 'BillingAccountsList', element: BillingAccountsList },
  { path: '/resources/billing_accounts/billing_accounts_create', name: 'BillingAccountsCreate', element: BillingAccountsCreate },
  { path: '/resources/billing_accounts/billing_accounts_detail/:id', name: 'BillingAccountsDetail', element: BillingAccountsDetail },

  { path: '/resources/conferences/conferences_list', name: 'ConferencesList', element: ConferencesList },
  { path: '/resources/conferences/conferences_create', name: 'ConferencesCreate', element: ConferencesCreate },
  { path: '/resources/conferences/conferences_detail/:id', name: 'ConferencesDetail', element: ConferencesDetail },

  { path: '/resources/conferences/conferencecalls_list', name: 'ConferencecallsList', element: ConferencecallsList },
  { path: '/resources/conferences/conferencecalls_detail/:id', name: 'ConferencecallsDetail', element: ConferencecallsDetail },



  { path: '/resources/campaigns/campaigns_list', name: 'CampaignsList', element: CampaignsList },
  { path: '/resources/campaigns/campaigns_create', name: 'CampaignsCreate', element: CampaignsCreate },
  { path: '/resources/campaigns/campaigns_detail/:id', name: 'CampaignsDetail', element: CampaignsDetail },

  { path: '/resources/campaigns/campaigncalls_list', name: 'Campaigncalls', element: Campaigncalls },

  { path: '/resources/outdials/outdials_list', name: 'OutdialsList', element: OutdialsList },
  { path: '/resources/outdials/outdials_create', name: 'OutdialsCreate', element: OutdialsCreate },
  { path: '/resources/outdials/outdials_detail/:id', name: 'OutdialsDetail', element: OutdialsDetail },

  { path: '/resources/outdials/:outdial_id/outdialtargets_list', name: 'OutdialtargetsList', element: OutdialtargetsList },
  { path: '/resources/outdials/:outdial_id/outdialtargets_create', name: 'OutdialtargetsCreate', element: OutdialtargetsCreate },
  { path: '/resources/outdials/:outdial_id/outdialtargets_detail/:id', name: 'OutdialtargetsDetail', element: OutdialtargetsDetail },

  { path: '/resources/outplans/outplans_list', name: 'OutplansList', element: OutplansList },
  { path: '/resources/outplans/outplans_create', name: 'OutplansCreate', element: OutplansCreate },
  { path: '/resources/outplans/outplans_detail/:id', name: 'OutplansDetail', element: OutplansDetail },



  { path: '/resources/chatbots/chatbots_list', name: 'ChatbotsList', element: ChatbotsList },
  { path: '/resources/chatbots/chatbots_create', name: 'ChatbotsCreate', element: ChatbotsCreate },
  { path: '/resources/chatbots/chatbots_detail/:id', name: 'ChatbotsDetail', element: ChatbotsDetail },



  { path: '/resources/chatbots/chatbotcalls_list', name: 'Chatbotcalls', element: Chatbotcalls },

  { path: '/resources/chats/chats_list', name: 'ChatsList', element: ChatsList },
  { path: '/resources/chats/chats_create', name: 'ChatsCreate', element: ChatsCreate },
  { path: '/resources/chats/chats_detail/:id', name: 'ChatsDetail', element: ChatsDetail },

  { path: '/resources/trunks/trunks_list', name: 'TrunksList', element: TrunksList },
  { path: '/resources/trunks/trunks_create', name: 'TrunksCreate', element: TrunksCreate },
  { path: '/resources/trunks/trunks_detail/:id', name: 'TrunksDetail', element: TrunksDetail },

  { path: '/resources/extensions/extensions_list', name: 'ExtensionsList', element: ExtensionsList },
  { path: '/resources/extensions/extensions_create', name: 'ExtensionsCreate', element: ExtensionsCreate },
  { path: '/resources/extensions/extensions_detail/:id', name: 'ExtensionsDetail', element: ExtensionsDetail },




  { path: '/theme', name: 'Theme', element: Colors, exact: true },
  { path: '/theme/colors', name: 'Colors', element: Colors },
  { path: '/theme/typography', name: 'Typography', element: Typography },
  { path: '/base', name: 'Base', element: Cards, exact: true },
  { path: '/base/accordion', name: 'Accordion', element: Accordion },
  { path: '/base/breadcrumbs', name: 'Breadcrumbs', element: Breadcrumbs },
  { path: '/base/cards', name: 'Cards', element: Cards },
  { path: '/base/carousels', name: 'Carousel', element: Carousels },
  { path: '/base/collapses', name: 'Collapse', element: Collapses },
  { path: '/base/list-groups', name: 'List Groups', element: ListGroups },
  { path: '/base/navs', name: 'Navs', element: Navs },
  { path: '/base/paginations', name: 'Paginations', element: Paginations },
  { path: '/base/placeholders', name: 'Placeholders', element: Placeholders },
  { path: '/base/popovers', name: 'Popovers', element: Popovers },
  { path: '/base/progress', name: 'Progress', element: Progress },
  { path: '/base/spinners', name: 'Spinners', element: Spinners },
  { path: '/base/tables', name: 'Tables', element: Tables },
  { path: '/base/tooltips', name: 'Tooltips', element: Tooltips },
  { path: '/buttons', name: 'Buttons', element: Buttons, exact: true },
  { path: '/buttons/buttons', name: 'Buttons', element: Buttons },
  { path: '/buttons/dropdowns', name: 'Dropdowns', element: Dropdowns },
  { path: '/buttons/button-groups', name: 'Button Groups', element: ButtonGroups },
  { path: '/charts', name: 'Charts', element: Charts },
  { path: '/forms', name: 'Forms', element: FormControl, exact: true },
  { path: '/forms/form-control', name: 'Form Control', element: FormControl },
  { path: '/forms/select', name: 'Select', element: Select },
  { path: '/forms/checks-radios', name: 'Checks & Radios', element: ChecksRadios },
  { path: '/forms/range', name: 'Range', element: Range },
  { path: '/forms/input-group', name: 'Input Group', element: InputGroup },
  { path: '/forms/floating-labels', name: 'Floating Labels', element: FloatingLabels },
  { path: '/forms/layout', name: 'Layout', element: Layout },
  { path: '/forms/validation', name: 'Validation', element: Validation },
  { path: '/icons', exact: true, name: 'Icons', element: CoreUIIcons },
  { path: '/icons/coreui-icons', name: 'CoreUI Icons', element: CoreUIIcons },
  { path: '/icons/flags', name: 'Flags', element: Flags },
  { path: '/icons/brands', name: 'Brands', element: Brands },
  { path: '/notifications', name: 'Notifications', element: Alerts, exact: true },
  { path: '/notifications/alerts', name: 'Alerts', element: Alerts },
  { path: '/notifications/badges', name: 'Badges', element: Badges },
  { path: '/notifications/modals', name: 'Modals', element: Modals },
  { path: '/notifications/toasts', name: 'Toasts', element: Toasts },
  { path: '/widgets', name: 'Widgets', element: Widgets },
]

export default routes
