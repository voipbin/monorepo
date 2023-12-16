import React, { useRef } from 'react'
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
import {
  phone,
  CallDial as PhoneDial,
  CallAnswer as PhoneAnswer,
  CallHangup as PhoneHangup,
  CallGetInfo,
} from '../../phone'

const CurrentcallDetail = () => {
  console.log("CurrentcallDetail");

  // get extension
  const tmp = localStorage.getItem("extension_info");
  const extension = JSON.parse(tmp);
  console.log("Debug info. extension info: ", extension);


  const ref_source = useRef(null);
  const ref_destination = useRef(null);


  const ref_destinations = useRef(null);
  const ref_actions = useRef(null);
  const ref_flow_id = useRef(null);

  const call_info = CallGetInfo();

  const Detail = () => {
    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Current call</strong> <small>Showing the current call detail info</small>
              </CCardHeader>

              <CCardBody>
                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Source</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_source}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={extension["extension"]}
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    {"+E164 number => normal telephone call."} <br />
                    {"agent:<agent-id> => Dial to the agent."} <br />
                    {"conference:<conference-id> => Join to the conference."} <br />
                    {"other wise => extension number."}
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destination</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_destination}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue=""
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" onClick={() => Dial()}>Dial</CButton>
                <br />
                <br />

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Source</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={call_info["from"]}
                      rows={10}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destinations</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={call_info["to"]}
                      rows={10}
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Status</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={call_info["status"]}
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Direction</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={call_info["direction"]}
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CCol>
                    <CButton type="submit" onClick={() => Answer()}>Answer</CButton>
                    &nbsp; &nbsp;
                    <CButton type="submit" onClick={() => Hangup()}>Hangup</CButton>
                  </CCol>
                </CRow>

              </CCardBody>
            </CCard>
          </CCol>
        </CRow>
      </>
    )
  };

  const Dial = () => {
    console.log("Dialing to the destination.");

    PhoneDial(ref_destination.current.value);
  };

  const Answer = () => {
    console.log("Answer the current call.");

    PhoneAnswer();
  };

  const Hangup = () => {
    console.log("Hangup the current call.");

    PhoneHangup();
  };

  const CreateResource = () => {
    console.log("Create info");

    const tmpData = {
      "source": JSON.parse(ref_source.current.value),
      "destinations": JSON.parse(ref_destinations.current.value),
      "flow_id": ref_flow_id.current.value,
      "actions": JSON.parse(ref_actions.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "calls";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body).then(() => {
      console.log("Created info.");
    });
  };

  return (
    <>
      <Detail/>
    </>
  )
}

export default CurrentcallDetail
