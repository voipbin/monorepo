import React, { useState } from 'react'
import { useSelector, useDispatch } from 'react-redux'

import { CSidebar, CSidebarBrand, CSidebarNav, CSidebarToggler } from '@coreui/react'
import CIcon from '@coreui/icons-react'

import { AppSidebarNav } from './AppSidebarNav'

import { logoNegative } from 'src/assets/brand/logo-negative'
import { sygnet } from 'src/assets/brand/sygnet'

import SimpleBar from 'simplebar-react'
import 'simplebar/dist/simplebar.min.css'

// sidebar nav config
import {navProjectAdmin, navCustomerAgent, navCustomerAdmin, navCustomerManager} from '../_nav'

const AppSidebar = () => {
  const dispatch = useDispatch()
  const unfoldable = useSelector((state) => state.sidebarUnfoldable);
  const sidebarShow = useSelector((state) => state.sidebarShow);
  const menus = useSelector((state) => {
    const agentInfo = JSON.parse(localStorage.getItem("agent_info"));
    console.log("Get detailed agent info. agent_info: ", agentInfo);

    if (agentInfo === null) {
      console.log("The customer has no agent info.");
      return navCustomerAgent;
    } 
    
    if (agentInfo["permission"] & 0x0001) {  // project super admin
      console.log("The customer has project admin permission." + 0x0001);
      return navProjectAdmin;
    }  else if (agentInfo["permission"] & 0x0020) {
      console.log("The customer has customer admin permission.");
      return navCustomerAdmin;
    } else if (agentInfo["permission"] & 0x0040) {
      console.log("The customer has customer manager permission.");
      return navCustomerManager;
    } else if (agentInfo["permission"] & 0x0010) {
      console.log("The customer has customer agent permission.");
      return navCustomerAgent;
    } else {
      console.log("The customer has no permission. permission: %d?", agentInfo["permission"]);
      return navCustomerAgent;
    }
  
  });

  return (
    <CSidebar
      position="fixed"
      unfoldable={unfoldable}
      visible={sidebarShow}
      onVisibleChange={(visible) => {
        dispatch({ type: 'set', sidebarShow: visible })
      }}
    >
      <CSidebarBrand className="d-none d-md-flex" to="/">
        <CIcon className="sidebar-brand-full" icon={logoNegative} height={35} />
        <CIcon className="sidebar-brand-narrow" icon={sygnet} height={35} />
      </CSidebarBrand>
      <CSidebarNav>
        <SimpleBar>
          <AppSidebarNav items={menus} />
        </SimpleBar>
      </CSidebarNav>
      <CSidebarToggler
        className="d-none d-lg-flex"
        onClick={() => dispatch({ type: 'set', sidebarUnfoldable: !unfoldable })}
      />
    </CSidebar>
  )
}

export default React.memo(AppSidebar)
