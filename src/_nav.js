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
  cilCommentSquare,
  cilLocomotive,
  cilLoopCircular,
} from '@coreui/icons'
import { CNavGroup, CNavItem, CNavTitle } from '@coreui/react'
import { useEffect } from 'react'

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

const billing_accounts = {
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
  name: 'Chatbots',
  icon: <CIcon icon={cilVideogame} customClassName="nav-icon" />,
  items: [
    {
      component: CNavItem,
      name: 'Chatbots',
      to: '/resources/chatbots/chatbots_list',
    },
  ]
};

const chats = {
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



const navProjectAdmin = [
  dashboard,

  // resource -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Resource',
  },
  calls,
  customers,
  queues,
  numbers,
  flows,
  agents,
  billing_accounts,
  conferences,
  chatbots,
  chats,
  messages,
  trunks,
  extensions,
  providers,
  routes,

  // outbound campaign -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Outbound Campaign',
  },
  campaigns,
  outdials,
  outplans,
]

const navCustomerAdmin = [
  dashboard,

  // resource -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Resource',
  },
  calls,
  queues,
  numbers,
  flows,
  agents,
  billing_accounts,
  conferences,
  chatbots,
  chats,
  messages,
  trunks,
  extensions,

  // outbound campaign -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Outbound Campaign',
  },
  campaigns,
  outdials,
  outplans,
]

const navCustomerManager = [
  dashboard,

  // resource -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Resource',
  },
  calls,
  queues,
  flows,
  agents,
  conferences,
  chatbots,
  chats,
  messages,
  trunks,
  extensions,

  // outbound campaign -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Outbound Campaign',
  },
  campaigns,
  outdials,
  outplans,
]

const navCustomerAgent = [
  dashboard,

  // resource -----------------------------------------------------------------------
  {
    component: CNavTitle,
    name: 'Resource',
  },
  calls,
  conferences,
  chats,
  messages,
]


var _nav = [];

const agentInfo = JSON.parse(localStorage.getItem("agent_info"));
if (agentInfo === null) {
  console.log("The customer has no agent info.");
  _nav = navCustomerAgent;
} else if (agentInfo["permission"] & 0x0001) {  // project super admin
  console.log("The customer has project admin permission." + 0x0001);
  _nav = navProjectAdmin;
} else if (agentInfo["permission"] & 0x0010) {
  console.log("The customer has customer agent permission.");
  _nav = navCustomerAgent;
} else if (agentInfo["permission"] & 0x0020) {
  console.log("The customer has customer admin permission.");
  _nav = navCustomerAdmin;
} else if (agentInfo["permission"] & 0x0040) {
  console.log("The customer has customer manager permission.");
  _nav = navCustomerManager;
} else {
  console.log("The customer has no permission.");
  _nav = navCustomerAgent;
}

export default _nav
