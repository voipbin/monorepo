import React from 'react'
import { Link } from 'react-router-dom'
import {
  CButton,
  CCard,
  CCardBody,
  CCardGroup,
  CCol,
  CContainer,
  CForm,
  CFormInput,
  CInputGroup,
  CInputGroupText,
  CRow,
} from '@coreui/react'
import CIcon from '@coreui/icons-react'
import { cilLockLocked, cilUser } from '@coreui/icons'
import { useState } from 'react';
import { useNavigate } from "react-router-dom";
import store from '../../../store'
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
  LoadResourcesAll as ProviderLoadResourcesAll,
} from '../../../provider';
import { Base64 } from "js-base64";
import { NewPhone } from "../../../phone";
import { useDispatch } from 'react-redux';
import { ChatroommessagesAddWithChatroomID, AgentInfoSet } from 'src/store';



const Login = () => {

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loginDisabled, setLoginDisabled] = useState(false);

  const navigate = useNavigate();
  const dispatch = useDispatch();

  const loginHandle = (event) => {
    event.preventDefault();

    console.log(`Logging in. username: ${username}, password: ${password}`);
    setLoginDisabled(true);

    // login
    const request = new Request('https://api.voipbin.net/auth/login', {
      method: 'POST',
      body: JSON.stringify({
          "username": username,
          "password": password,
      }),
      headers: new Headers({ 'Content-Type': 'application/json' }),
    });

    fetch(request)
      .then(response => {
        if (response.status < 200 || response.status >= 300) {
          setLoginDisabled(false);
          console.log("Could not login. response: ", JSON.stringify(response));
          return ""
        }

        console.log("Received response: ", JSON.stringify(response));
        return response.json();
      })
      .then(response => {
        if (response == "") {
          return
        }
        console.log("received data. token: ", response.token);

        // store the token in local storage
        localStorage.setItem('token', response.token);

        const splits = response.token.split('.');
        console.log("Login info. splits: ", splits);

        // agent
        const agentInfo = JSON.parse(Base64.decode(splits[1])).agent;
        const tmpAgent = JSON.stringify(agentInfo);
        localStorage.setItem('agent_info', tmpAgent);
        console.log("Agent info. splits: ", tmpAgent);

        // load all resources
        ProviderLoadResourcesAll();

        NewPhone(dispatch);
        const phone = localStorage.getItem("phone");
        console.log("Detailed phone info. phone: ", phone)

        dispatch(AgentInfoSet(agentInfo));

        // websocket
        Connect(agentInfo);

        navigate('/');
    })
  };

  const WS_URL = 'wss://api.voipbin.net/v1.0/ws';
  const Connect = (agentInfo) => {
    let authToken = localStorage.getItem("token");
    let url = WS_URL + '?token=' + authToken

    console.log("Establishing websocket connection. utl: ", url);
    var ws = new WebSocket(url);

    ws.onopen = () => {

      var topics = [];

      // generate topics. customer level
      const resourcesCustomerLevel = ["call", "activeflow"]; 
      for (var i = 0; i < resourcesCustomerLevel.length; i++) {
        const tmpResource = resourcesCustomerLevel[i];
        const tmpTopic = "\"customer_id:" + agentInfo["customer_id"] + ":" + tmpResource + "\"";
        topics.push(tmpTopic);
      }

      // generate topics. agent level
      const resourcesAgentLevel = ["messagechatroom"];
      for (var i = 0; i < resourcesAgentLevel.length; i++) {
        const tmpResource = resourcesAgentLevel[i];
        const tmpTopic = "\"agent_id:" + agentInfo["id"] + ":" + tmpResource + "\"";
        topics.push(tmpTopic);
      }
      
      // subscribe topics
      let m = '{"type": "subscribe","topics": [' + topics + ']}';
      console.log("Subscribing topics. message: ", m);
      ws.send(m);
    }

    ws.onmessage = (e) => {

      const data = JSON.parse(e.data);
      const type = data["type"];
      const message = data["data"];
      console.log("Recevied websocket type: %s, message: %s", type, message);


      switch (type) {
        case "messagechatroom_created":
          console.log("Chatroom message received. chatroom_id: %s, message: %s", message["chatroom_id"], message["text"]);
          dispatch(ChatroommessagesAddWithChatroomID(message["chatroom_id"], message));
      }
    }
  }

  const registerHandle = (event) => {
    console.log("registerHandle");
    return (
      <Link to='javascript:void(0)'
        onClick={() => window.location = 'mailto:yourmail@domain.com'}>
        Contact Me
      </Link>
    );
  };

  return (
    <div className="bg-light min-vh-100 d-flex flex-row align-items-center">
      <CContainer>
        <CRow className="justify-content-center">
          <CCol md={8}>
            <CCardGroup>
              <CCard className="p-4">
                <CCardBody>
                  <CForm>
                    <h1>Login</h1>
                    <p className="text-medium-emphasis">Sign In to your account</p>
                    <CInputGroup className="mb-3">
                      <CInputGroupText>
                        <CIcon icon={cilUser} />
                      </CInputGroupText>
                      <CFormInput
                        id="username"
                        placeholder="Username"
                        autoComplete="username"
                        value={username}
                        onChange={(event) => setUsername(event.target.value)}
                      />
                    </CInputGroup>
                    <CInputGroup className="mb-4">
                      <CInputGroupText>
                        <CIcon icon={cilLockLocked} />
                      </CInputGroupText>
                      <CFormInput
                        id="password"
                        type="password"
                        placeholder="Password"
                        autoComplete="current-password"
                        value={password}
                        onChange={(event) => setPassword(event.target.value)}
                      />
                    </CInputGroup>
                    <CRow>
                      <CCol xs={6}>
                        <CButton onClick={loginHandle} color="primary" className="px-4" disabled={loginDisabled}>
                          Login
                        </CButton>
                      </CCol>
                      <CCol xs={6} className="text-right">
                        <CButton color="link" className="px-0">
                          Forgot password?
                        </CButton>
                      </CCol>
                    </CRow>
                  </CForm>
                </CCardBody>
              </CCard>
              <CCard className="text-white bg-primary py-5" style={{ width: '44%' }}>
                <CCardBody className="text-center">
                  <div>
                    <h2>Sign up</h2>
                    <p>
                      The voipbin is currently operating in development mode.<br /> Kindly submit a sign-up request to the administrator for registration.
                    </p>

                    <Link to='javascript:void(0)' onClick={() => window.location = 'mailto:pchero21@gmail.com'}>
                      <CButton color="primary" className="mt-3" active tabIndex={-1}>
                        Register Now!
                      </CButton>
                    </Link>

                  </div>
                </CCardBody>
              </CCard>
            </CCardGroup>
          </CCol>
        </CRow>
      </CContainer>
    </div>
  )
}

export default Login
