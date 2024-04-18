import React, { useRef, useState, useEffect } from 'react'
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
import { useSelector, useDispatch } from 'react-redux'
import { useNavigate } from "react-router-dom";

const CurrentcallDetail = () => {
  console.log("CurrentcallDetail");
  const navigate = useNavigate();

  const ref_source = useRef(null);
  const ref_destination = useRef(null);
  const call_info = CallGetInfo();

  let currentCall = useSelector((state) => {
    // note: i don't know why this is needed, but this makes possible to
    // update the call info on the fly.
    let res = [state.resourceCurrentcallReducer];
    return res;
  });
  console.log("currentcall info: ", currentCall);

  // get extension
  const tmp = localStorage.getItem("extension_info");
  var extension = JSON.parse(tmp);
  console.log("Debug info. extension info: ", extension);
  if (extension == null) {
    extension = {};
  }

  useEffect(() => {

    // validate the extension info  
    if (Object.keys(extension).length === 0) {
      alert("You have no extension in the address. Please add the extension to your address.");

      // go to main page
      const navi = "/";
      navigate(navi);
    }
  });

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
                      readOnly
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destinations</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={call_info["to"]}
                      rows={10}
                      readOnly
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
                      readOnly
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
                      readOnly
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

  return (
    <>
      <Detail/>
    </>
  )
}

export default CurrentcallDetail
