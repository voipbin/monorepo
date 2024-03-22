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


const Conversationmessages = () => {

  const routeParams = useParams();

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [listMessages, setMessages] = useState([]);

  const ref_message = useRef("");


  useEffect(() => {
    getList();
    return;
  }, []);

  const conversation_id = routeParams.conversation_id;

  const getList = (() => {
    const target = "conversations/" + conversation_id + "/messages?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("conversations/" + conversation_id + "/messages", tmpData);

        var tmpMessages = [];
        data.forEach(tmp => {

          if (tmp["status"] == "sent") {
            console.log("Outbound message. status: %s, message: %s", tmp["status"], tmp["text"]);
            tmpMessages.push(
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
          } else if (tmp["status"] == "received") {
            console.log("Inbound message. status: %s, message: %s", tmp["status"], tmp["text"]);
            tmpMessages.push(
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

        setMessages(tmpMessages.reverse());
      })
      .catch(e => {
        console.log("Could not get list. err: %o", e);
        alert("Could not get list.");
      });
  });

  const SendMessage = () => {
    console.log("SendMessage info. message: ", ref_message.current.value);

    const tmpData = {
      "text": ref_message.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "conversations/" + conversation_id + "/messages";
    console.log("Sending message info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then(response => {
        console.log("Sent message info. response: " + JSON.stringify(response));
      })
      .catch(e => {
        console.log("Could not send the message. err: %o", e);
        alert("Could not send the message.");
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
            {listMessages}

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

export default Conversationmessages
