import React from 'react'
import CIcon from '@coreui/icons-react'
import {
  cilBell,
  cilCalculator,
  cilChartPie,
  cilCursor,
  cilDescription,
  cilDrop,
  cilNotes,
  cilPencil,
  cilPuzzle,
  cilSpeedometer,
  cilStar,
  cilPhone,
  cilSpeech,
  cilSmile,
  cilLan,
  cilDialpad,
  cilFork,
  cilEqualizer,
  cilGroup,
  cilDollar,
  cilBook,
  cilVideogame,
  cilSpreadsheet,
  cilListRich,
  cilHandPointRight,
  cilRouter,
  cilMap,
  cilChatBubble,
  cilLocomotive,
  cilLoopCircular,
} from '@coreui/icons'
import { CNavGroup, CNavItem, CNavTitle } from '@coreui/react'

const _nav = [
  {
    component: CNavItem,
    name: 'Dashboard',
    to: '/dashboard',
    icon: <CIcon icon={cilSpeedometer} customClassName="nav-icon" />,
    badge: {
      color: 'info',
      text: 'NEW',
    },
  },

  // resource -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Resource',
  },
  {
    component: CNavGroup,
    name: 'Calls',
    to: '/resources/calls',
    icon: <CIcon icon={cilEqualizer} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Calls',
        to: '/resources/calls/calls_list',
      },
      {
        component: CNavItem,
        name: 'Groupcalls',
        to: '/resources/calls/groupcalls_list',
      }
    ]
  },
  {
    component: CNavGroup,
    name: 'Customers',
    icon: <CIcon icon={cilSmile} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Customers',
        to: '/resources/customers/customers_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Queues',
    icon: <CIcon icon={cilEqualizer} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Queues',
        to: '/resources/queues/queues_list',
      },
      {
        component: CNavItem,
        name: 'Queuecalls',
        to: '/resources/queues/queuecalls_list',
      }
    ]
  },
  {
    component: CNavGroup,
    name: 'Numbers',
    icon: <CIcon icon={cilDialpad} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Active',
        to: '/resources/numbers/active_list',
      },
      {
        component: CNavItem,
        name: 'Buy',
        to: '/resources/numbers/buy_list',
      }
    ]
  },
  {
    component: CNavGroup,
    name: 'Flows',
    icon: <CIcon icon={cilFork} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Flows',
        to: '/resources/flows/flows_list',
      },
      {
        component: CNavItem,
        name: 'Activeflows',
        to: '/resources/flows/activeflows_list',
      }
    ]
  },
  {
    component: CNavGroup,
    name: 'Agents',
    icon: <CIcon icon={cilGroup} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Agents',
        to: '/resources/agents/agents_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'BillingAccounts',
    icon: <CIcon icon={cilDollar} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'BillingAccounts',
        to: '/resources/billing_accounts/billing_accounts_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Conferences',
    icon: <CIcon icon={cilBook} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Conferences',
        to: '/resources/conferences/conferences_list',
      },
      {
        component: CNavItem,
        name: 'Conferencecalls',
        to: '/resources/conferences/conferencecalls_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Chatbots',
    icon: <CIcon icon={cilVideogame} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Chatbots',
        to: '/resources/chatbots/chatbots_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Chats',
    icon: <CIcon icon={cilChatBubble} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Chats',
        to: '/resources/chats/chats_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Trunks',
    icon: <CIcon icon={cilRouter} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Trunks',
        to: '/resources/trunks/trunks_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Extensions',
    icon: <CIcon icon={cilMap} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Extensions',
        to: '/resources/extensions/extensions_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Providers',
    icon: <CIcon icon={cilLocomotive} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Providers',
        to: '/resources/providers/providers_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Routes',
    icon: <CIcon icon={cilLoopCircular} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Routes',
        to: '/resources/routes/routes_list',
      },
    ]
  },

  // outbound campaign -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Outbound Campaign',
  },
  {
    component: CNavGroup,
    name: 'Campaigns',
    to: '/resources/campaigns/campaigns_list',
    icon: <CIcon icon={cilHandPointRight} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Campaigns',
        to: '/resources/campaigns/campaigns_list',
      },
      {
        component: CNavItem,
        name: 'Campaigncalls',
        to: '/resources/campaigns/campaigncalls_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Outdials',
    icon: <CIcon icon={cilSpreadsheet} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Outdials',
        to: '/resources/outdials/outdials_list',
      },
    ]
  },
  {
    component: CNavGroup,
    name: 'Outplans',
    icon: <CIcon icon={cilListRich} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Outplans',
        to: '/resources/outplans/outplans_list',
      },
    ]
  },
]

export default _nav
