import React from 'react'
import { CFooter } from '@coreui/react'

const AppFooter = () => {
  return (
    <CFooter>
      <div>
        <a href="https://voipbin.net" target="_blank" rel="noopener noreferrer">
          Admin
        </a>
        <span className="ms-1">&copy; 2023 project voipbin.</span>
      </div>
      <div className="ms-auto">
        <span className="me-1">Powered by</span>
        <a href="https://voipbin.net" target="_blank" rel="noopener noreferrer">
          project VoIPBin
        </a>
      </div>
    </CFooter>
  )
}

export default React.memo(AppFooter)
