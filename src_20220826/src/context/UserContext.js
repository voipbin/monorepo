import React, { createContext, useContext, useReducer } from 'react'
import Realm from "realm"

function userInfoReducer(state, action) {
  switch (action.type) {
    case 'LOGIN':
      state = {
        ...state,
        ...action.obj,
      }
      return state
    default:
      throw new Error('Unhandle action type : ${action.type}')
  }
}

const UserInfoStateContext = createContext()
const UserInfoDispatchContext = createContext()

export function UserInfoProvider({ children }) {
  const [userInfo, dispatch] = useReducer(userInfoReducer, {})
  return (
    <UserInfoStateContext.Provider value={userInfo}>
      <UserInfoDispatchContext.Provider value={dispatch}>
        {children}
      </UserInfoDispatchContext.Provider>
    </UserInfoStateContext.Provider>
  )
}

export function useUserInfoState() {
  const context = useContext(UserInfoStateContext)
  if (!context) {
    throw new Error('Cannot find UserInfoProvider')
  }
  return context
}

export function useUserInfoDispatch() {
  const context = useContext(UserInfoDispatchContext)
  if (!context) {
    throw new Error('Cannot find UserInfoProvider')
  }
  return context
}
