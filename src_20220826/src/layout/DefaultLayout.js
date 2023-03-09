import React from 'react'
import { AppContent, AppSidebar, AppFooter, AppHeader } from '../components/index'
import { useUserInfoState } from 'src/context/UserContext';

const Login = React.lazy(() => import('../views/pages/login/Login'));

const DefaultLayout = () => {
  console.log(`DefaultLayout`)

  const userInfoState = useUserInfoState();
  const token =userInfoState.token;
  console.log(`token : ${token}`);

  if(!token){
    return(
      <Login />
    )
  }

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
