import React, { useMemo, useState, useEffect, useRef } from 'react'
import { useParams } from "react-router-dom";
import {
  CCard,
  CCardBody,
  CCardHeader,
  CCol,
  CFormInput,
  CFormLabel,
  CFormTextarea,
  CRow,
  CButton,
} from '@coreui/react'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Stack,
  TextField,
  Tooltip,
} from '@mui/material';
import { Delete, Edit } from '@mui/icons-material';
import store from '../../store'
import { MaterialReactTable } from 'material-react-table';
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";
import { useSelector, useDispatch } from 'react-redux'
import { ChatroommessagesSetWithChatroomID } from 'src/store';

const Roommessages = () => {
  const dispatch = useDispatch();
  const routeParams = useParams();
  const ref_message = useRef("");

  useEffect(() => {
    getList();
    return;
  }, []);

  const agentInfo = JSON.parse(localStorage.getItem("agent_info"));
  const room_id = routeParams.room_id;

  let messagesAll = useSelector(state => {
    var messages = state.resourceChatroommessagesReducer.data[room_id];

    var tmpRes = [];
    messages?.forEach(tmp => {
      if (tmp["source"].target == agentInfo["id"]) {
        tmpRes.push(
          <CFormInput
            type="text"
            floatingClassName="mb-3"
            floatingLabel="me"
            color="success"
            defaultValue={tmp["text"]}
            aria-label="Disabled input example"
            readOnly
            valid
          />
        );
      } else {
        tmpRes.push(
          <CFormInput
            type="text"
            floatingClassName="mb-3"
            floatingLabel={tmp["source"].target}
            color="primary"
            defaultValue={tmp["text"]}
            aria-label="Disabled input example"
            readOnly
          />
        );
      }
    });

    const res = tmpRes.reverse();
    console.log("Update message: ", res);

    return res;
  });

  const getList = (() => {
    const target = "chatroommessages?page_size=100&chatroom_id=" + room_id;

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        dispatch(ChatroommessagesSetWithChatroomID(room_id, data));
      })
      .catch(e => {
        console.log("Could not get list of chatroom messages. err: %o", e);
        alert("Could not get the list of chatroom messages.");
      });
  });

  const SendMessage = () => {
    console.log("SendMessage info. message: ", ref_message.current.value);

    const tmpData = {
      "chatroom_id": room_id,
      "text": ref_message.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "chatroommessages"
    console.log("Sending message info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then(response => {
        console.log("Sent message info. response: " + JSON.stringify(response));
      })
      .catch(e => {
        console.log("Could not send the chat message. err: %o", e);
        alert("Could not send the chat message.");
      });
  };

  return (
    <CRow>
      <CCol xs={12}>
        <CCard className="mb-4">
          <CCardHeader>
            <strong>Conversation detail</strong> <small>Detail of the conversation</small>
          </CCardHeader>

          <CCardBody>
            {messagesAll}

            <CRow>
              <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Message</b></CFormLabel>
              <CCol className="mb-3 align-items-auto">
                <CFormInput
                  ref={ref_message}
                  type="text"
                  id="colFormLabelSm"
                />
              </CCol>

              <CCol className="mb-3 align-items-auto">
                <CButton type="submit" onClick={() => SendMessage()}>Send</CButton>
              </CCol>
            </CRow>

          </CCardBody>
        </CCard>
      </CCol>
    </CRow>
  )
}

export default Roommessages
