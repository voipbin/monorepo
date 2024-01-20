import React from 'react'
import { AppContent, AppSidebar, AppFooter, AppHeader } from '../components/index'

import { useEffect } from 'react'
import { useNavigate } from "react-router-dom";

const DefaultLayout = () => {

  const navigate = useNavigate();
  useEffect(() => {

    const agentInfo = JSON.parse(localStorage.getItem("agent_info"));
    console.log("check agent info. agent: ", agentInfo);

    if (agentInfo === null) {
      navigate("/login");
    }
  }, []);

  return (
    <div>
      <AppSidebar />
      <div className="wrapper d-flex flex-column min-vh-100 bg-light">
        <AppHeader />
        <div className="body flex-grow-1 px-3">
          <AppContent />
        </div>
        <AppFooter />
      </div>
    </div>
  )
}

export default DefaultLayout
