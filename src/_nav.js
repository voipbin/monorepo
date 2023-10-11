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
  {
    component: CNavTitle,
    name: 'Activity',
  },
  {
    component: CNavItem,
    name: 'Calls',
    to: '/activity/calls',
    icon: <CIcon icon={cilPhone} customClassName="nav-icon" />,
  },
  {
    component: CNavItem,
    name: 'Activeflows',
    to: '/resources/flows/activeflows_list',
    icon: <CIcon icon={cilLan} customClassName="nav-icon" />,
  },
  {
    component: CNavItem,
    name: 'Messages',
    to: '/activity/messages',
    icon: <CIcon icon={cilSpeech} customClassName="nav-icon" />,
  },

  {
    component: CNavTitle,
    name: 'Resources',
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
    component: CNavItem,
    name: 'Customers',
    to: '/resources/customers/customers_list',
    icon: <CIcon icon={cilSmile} customClassName="nav-icon" />,
  },
  {
    component: CNavGroup,
    name: 'Queues',
    to: '/resources/queues',
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
    to: '/numbers',
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
    to: '/resources/flows/flows_list',
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
    to: '/resources/agents/agents_list',
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
    component: CNavItem,
    name: 'BillingAccounts',
    to: '/resources/billing_accounts/billing_accounts_list',
    icon: <CIcon icon={cilDollar} customClassName="nav-icon" />,
  },
  {
    component: CNavGroup,
    name: 'Conferences',
    to: '/resources/conferences/conferences_list',
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
    to: '/resources/chatbots/chatbots',
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
    to: '/resources/trunks/trunks_list',
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
    to: '/resources/extensions/extensions_list',
    icon: <CIcon icon={cilMap} customClassName="nav-icon" />,
    items: [
      {
        component: CNavItem,
        name: 'Extensions',
        to: '/resources/extensions/extensions_list',
      },
    ]
  },

  // -----------------------------------------------------------------------
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
        to: '/resources/campaigns/campaigns_list',
      },
      // {
      //   component: CNavItem,
      //   name: 'Campaigncalls',
      // },
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


  // {
  //   component: CNavTitle,
  //   name: 'Theme',
  // },
  // {
  //   component: CNavItem,
  //   name: 'Colors',
  //   to: '/theme/colors',
  //   icon: <CIcon icon={cilDrop} customClassName="nav-icon" />,
  // },
  // {
  //   component: CNavItem,
  //   name: 'Typography',
  //   to: '/theme/typography',
  //   icon: <CIcon icon={cilPencil} customClassName="nav-icon" />,
  // },
  // {
  //   component: CNavTitle,
  //   name: 'Components',
  // },
  // {
  //   component: CNavGroup,
  //   name: 'Base',
  //   to: '/base',
  //   icon: <CIcon icon={cilPuzzle} customClassName="nav-icon" />,
  //   items: [
  //     {
  //       component: CNavItem,
  //       name: 'Accordion',
  //       to: '/base/accordion',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Breadcrumb',
  //       to: '/base/breadcrumbs',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Cards',
  //       to: '/base/cards',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Carousel',
  //       to: '/base/carousels',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Collapse',
  //       to: '/base/collapses',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'List group',
  //       to: '/base/list-groups',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Navs & Tabs',
  //       to: '/base/navs',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Pagination',
  //       to: '/base/paginations',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Placeholders',
  //       to: '/base/placeholders',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Popovers',
  //       to: '/base/popovers',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Progress',
  //       to: '/base/progress',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Spinners',
  //       to: '/base/spinners',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Tables',
  //       to: '/base/tables',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Tooltips',
  //       to: '/base/tooltips',
  //     },
  //   ],
  // },
  // {
  //   component: CNavGroup,
  //   name: 'Buttons',
  //   to: '/buttons',
  //   icon: <CIcon icon={cilCursor} customClassName="nav-icon" />,
  //   items: [
  //     {
  //       component: CNavItem,
  //       name: 'Buttons',
  //       to: '/buttons/buttons',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Buttons groups',
  //       to: '/buttons/button-groups',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Dropdowns',
  //       to: '/buttons/dropdowns',
  //     },
  //   ],
  // },
  // {
  //   component: CNavGroup,
  //   name: 'Forms',
  //   icon: <CIcon icon={cilNotes} customClassName="nav-icon" />,
  //   items: [
  //     {
  //       component: CNavItem,
  //       name: 'Form Control',
  //       to: '/forms/form-control',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Select',
  //       to: '/forms/select',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Checks & Radios',
  //       to: '/forms/checks-radios',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Range',
  //       to: '/forms/range',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Input Group',
  //       to: '/forms/input-group',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Floating Labels',
  //       to: '/forms/floating-labels',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Layout',
  //       to: '/forms/layout',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Validation',
  //       to: '/forms/validation',
  //     },
  //   ],
  // },
  // {
  //   component: CNavItem,
  //   name: 'Charts',
  //   to: '/charts',
  //   icon: <CIcon icon={cilChartPie} customClassName="nav-icon" />,
  // },
  // {
  //   component: CNavGroup,
  //   name: 'Icons',
  //   icon: <CIcon icon={cilStar} customClassName="nav-icon" />,
  //   items: [
  //     {
  //       component: CNavItem,
  //       name: 'CoreUI Free',
  //       to: '/icons/coreui-icons',
  //       badge: {
  //         color: 'success',
  //         text: 'NEW',
  //       },
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'CoreUI Flags',
  //       to: '/icons/flags',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'CoreUI Brands',
  //       to: '/icons/brands',
  //     },
  //   ],
  // },
  // {
  //   component: CNavGroup,
  //   name: 'Notifications',
  //   icon: <CIcon icon={cilBell} customClassName="nav-icon" />,
  //   items: [
  //     {
  //       component: CNavItem,
  //       name: 'Alerts',
  //       to: '/notifications/alerts',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Badges',
  //       to: '/notifications/badges',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Modal',
  //       to: '/notifications/modals',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Toasts',
  //       to: '/notifications/toasts',
  //     },
  //   ],
  // },
  // {
  //   component: CNavItem,
  //   name: 'Widgets',
  //   to: '/widgets',
  //   icon: <CIcon icon={cilCalculator} customClassName="nav-icon" />,
  //   badge: {
  //     color: 'info',
  //     text: 'NEW',
  //   },
  // },
  // {
  //   component: CNavTitle,
  //   name: 'Extras',
  // },
  // {
  //   component: CNavGroup,
  //   name: 'Pages',
  //   icon: <CIcon icon={cilStar} customClassName="nav-icon" />,
  //   items: [
  //     {
  //       component: CNavItem,
  //       name: 'Login',
  //       to: '/login',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Register',
  //       to: '/register',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Error 404',
  //       to: '/404',
  //     },
  //     {
  //       component: CNavItem,
  //       name: 'Error 500',
  //       to: '/500',
  //     },
  //   ],
  // },
  // {
  //   component: CNavItem,
  //   name: 'Docs',
  //   href: 'https://coreui.io/react/docs/templates/installation/',
  //   icon: <CIcon icon={cilDescription} customClassName="nav-icon" />,
  // },
]

export default _nav
