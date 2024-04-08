import React, { useState } from 'react'
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
  cilCommentBubble,
  cilCommentSquare,
  cilLocomotive,
  cilLoopCircular,
  cilPaperclip,
  cilMediaPlay,
  cilFont,
} from '@coreui/icons'
import { CNavGroup, CNavItem, CNavTitle } from '@coreui/react'
import { useEffect } from 'react'
import { useNavigate } from "react-router-dom";
import { useSelector, useDispatch } from 'react-redux'

const dashboard = {
  component: CNavItem,
  name: 'Dashboard',
  to: '/dashboard',
  icon: <CIcon icon={cilSpeedometer} customClassName="nav-icon" />,
  badge: {
    color: 'info',
    text: 'NEW',
  }
}

const calls = {
  component: CNavGroup,
  name: 'Calls',
  to: '/resources/calls',
  icon: <CIcon icon={cilPhone} customClassName="nav-icon" />,
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
    },
    {
      component: CNavItem,
      name: 'Currentcall',
      to: '/resources/calls/currentcall_detail',
    }
  ]
};

const customers = {
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
};

const queues = {
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
};

const tags = {
  component: CNavGroup,
  name: 'Tags',
  icon: <CIcon icon={cilPaperclip} customClassName="nav-icon" />,
  items: [
    {
      component: CNavItem,
      name: 'Tags',
      to: '/resources/tags/tags_list',
    },
  ]
};

const numbers = {
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
};

const flows = {
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
};

const agents = {
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
};

const billings = {
  component: CNavGroup,
  name: 'Billings',
  icon: <CIcon icon={cilDollar} customClassName="nav-icon" />,
  items: [
    {
      component: CNavItem,
      name: 'Billing Accounts',
      to: '/resources/billing_accounts/billing_accounts_list',
    },
    {
      component: CNavItem,
      name: 'Billings',
      to: '/resources/billings/billings_list',
    },
  ]
};

const conferences = {
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
};

const chatbots = {
  component: CNavGroup,
  name: 'Chatbots(AI)',
  icon: <CIcon icon={cilVideogame} customClassName="nav-icon" />,
  items: [
    {
      component: CNavItem,
      name: 'Chatbots(AI)',
      to: '/resources/chatbots/chatbots_list',
    },
  ]
};

const conversations = {
  component: CNavGroup,
  name: 'Conversations',
  icon: <CIcon icon={cilChatBubble} customClassName="nav-icon" />,
  items: [
    {
      component: CNavItem,
      name: 'Conversations',
      to: '/resources/conversations/conversations_list',
    },
    {
      component: CNavItem,
      name: 'Accounts',
      to: '/resources/conversations/accounts_list',
    },  ]
};

const chats = {
  component: CNavGroup,
  name: 'Chats',
  icon: <CIcon icon={cilCommentBubble} customClassName="nav-icon" />,
  items: [
    // {
    //   component: CNavItem,
    //   name: 'Chats',
    //   to: '/resources/chats/chats_list',
    // },
    {
      component: CNavItem,
      name: 'Rooms',
      to: '/resources/chats/rooms_list',
    },
  ]
};

const messages = {
  component: CNavGroup,
  name: 'Messages',
  icon: <CIcon icon={cilCommentSquare} customClassName="nav-icon" />,
  items: [
    {
      component: CNavItem,
      name: 'Messages',
      to: '/resources/messages/messages_list',
    },
  ]
};


const trunks = {
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
};

const extensions = {
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
};

const providers = {
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
};

const routes = {
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
};

const recordings = {
  component: CNavGroup,
  name: 'Recordings',
  icon: <CIcon icon={cilMediaPlay} customClassName="nav-icon" />,
  items: [
    {
      component: CNavItem,
      name: 'Recordings',
      to: '/resources/recordings/recordings_list',
    },
  ]
};

const transcribes = {
  component: CNavGroup,
  name: 'Transcribes',
  icon: <CIcon icon={cilFont} customClassName="nav-icon" />,
  items: [
    {
      component: CNavItem,
      name: 'Transcribes',
      to: '/resources/transcribes/transcribes_list',
    },
  ]
};



//// Campaign ////////////////////////////////////////////////////////////////////////////////

const campaigns = {
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
};

const outdials = {
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
};

const outplans = {
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
};

// groupProject defines menu groups for project resource management.
const groupProject = [
  {
    component: CNavTitle,
    name: 'Project Admin',
  },

  customers,
  providers,
  routes,
];

// groupAdmin defines menu groups for admin resource management.
const groupAdmin = [
  {
    component: CNavTitle,
    name: 'Admin',
  },

  numbers,
  billings,
];

// groupManager defines menu groups for normal resource management.
const groupResourceManager = [
  {
    component: CNavTitle,
    name: 'Resource',
  },

  flows,
  agents,
  queues,
  tags,
  conferences,
  chatbots,
  recordings,
  transcribes,
  trunks,
  extensions,
];

// groupCommnunication defines menu groups for communication.
const groupCommnunication = [
  {
    component: CNavTitle,
    name: 'Communication',
  },

  calls,
  conversations,
  chats,
  messages,
];

// groupCampaign defines menu groups for campaign management.
const groupCampaign = [
  {
    component: CNavTitle,
    name: 'Outbound Campaign',
  },

  campaigns,
  outdials,
  outplans,
];

// groupAgent defines menu groups for agent permission user.
const groupAgent = [
  {
    component: CNavTitle,
    name: 'Resource',
  },

  calls,
  conferences,
  conversations,
  chats,
  messages,
];

// navCustomerAdmin defines navigation menu for customer admin permission.
export const navCustomerAdmin = [
  dashboard,
  ...groupCommnunication,
  ...groupResourceManager,
  ...groupAdmin,
  ...groupCampaign,
]

// navCustomerManager defines navigation menu for customer manager permission.
export const navCustomerManager = [
  dashboard,
  ...groupCommnunication,
  ...groupResourceManager,
  ...groupCampaign,
]

// navCustomerAgent defines navigation menu for customer agent permission.
export const navCustomerAgent = [
  dashboard,
  ...groupCommnunication,
]

// navProjectAdmin defines navigation menu for project admin permission.
export const navProjectAdmin = [
  dashboard,
  ...groupCommnunication,
  ...groupResourceManager,
  ...groupAdmin,
  ...groupCampaign,
  ...groupProject,
];
