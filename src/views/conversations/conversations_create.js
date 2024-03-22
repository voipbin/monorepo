import React, { useRef, useState } from 'react'
import { useParams } from "react-router-dom";
import {
  CCard,
  CCardBody,
  CCardHeader,
  CCol,
  CFormInput,
  CFormLabel,
  CRow,
  CFormTextarea,
  CButton,
  CFormSelect,
  } from '@coreui/react'
import store from '../../store'
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";

const ConversationsCreate = () => {
  console.log("ConversationsCreate");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_source = useRef(null);
  const ref_destinations = useRef(null);
  const ref_actions = useRef(null);
  const ref_flow_id = useRef(null);

  const Create = () => {
    const id = routeParams.id;

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Create</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/conversation.html" target="_blank">here</a>.</small>
              </CCardHeader>

              <CCardBody>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Source</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_source}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(JSON.parse('{"type":"tel", "target":""}'), null, 2)}
                      rows={10}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destinations</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_destinations}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(JSON.parse('[{"type":"tel", "target":""}]'), null, 2)}
                      rows={10}
                    />
                  </CCol>

                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Flow ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_flow_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="00000000-0000-0000-0000-000000000000"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Actions</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_actions}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(JSON.parse('[{"type":"talk", "option":{"text": "hello world", "gender": "female", "language": "en-US"}}]'), null, 2)}
                      rows={10}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" disabled={buttonDisable} onClick={() => CreateResource()}>Create</CButton>

              </CCardBody>
            </CCard>
          </CCol>
        </CRow>
      </>
    )
  };

  const CreateResource = () => {
    console.log("Create info");
    setButtonDisable(true);

    const tmpData = {
      "source": JSON.parse(ref_source.current.value),
      "destinations": JSON.parse(ref_destinations.current.value),
      "flow_id": ref_flow_id.current.value,
      "actions": JSON.parse(ref_actions.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "calls";
    console.log("Creating call info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created call info.", JSON.stringify(response));
        const navi = "/resources/conversations/conversations_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not create a new conversation. err: %o", e);
        alert("Could not create a new conversation.");
        setButtonDisable(false);
      });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default ConversationsCreate
