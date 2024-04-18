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
  CButton,
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

const CallsDetail = () => {
  console.log("CallsDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const navigate = useNavigate();

  const ref_id = useRef(null);

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("calls");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];

    var hangupDisable = false;
    if (detailData["status"] == "hangup") {
      hangupDisable = true;
    }

    return (
      <CRow>
        <CCol xs={12}>
          <CCard className="mb-4">
            <CCardHeader>
              <strong>Call detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/call.html" target="_blank">here</a>.</small>
            </CCardHeader>

            <CCardBody>
              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>ID</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    ref={ref_id}
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.id}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Status</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.status}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Flow ID</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.flow_id}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Activeflow ID</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.activeflow_id}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>From</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.source.target}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>To</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.destination.target}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Direction</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.direction}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Hangup By</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.hangup_by}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Hangup Reason</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.hangup_reason}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Create Timestamp</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.tm_create}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Ringing Timestamp</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.tm_ringing}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Progressing Timestamp</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.tm_progressing}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Hangup Timestamp</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.tm_hangup}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Update Timestamp</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.tm_update}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <br />
              <CButton type="submit" disabled={hangupDisable} onClick={() => Hangup()}>Hangup</CButton>
              &nbsp;
              <CButton type="submit" color="dark" disabled={buttonDisable} onClick={() => Delete()}>Delete</CButton>
            </CCardBody>
          </CCard>
        </CCol>
      </CRow>
    )
  };

  const navigateBack = () => {
    navigate(-1);
  }

  const Hangup = () => {
    console.log("Hangup");
    setButtonDisable(true);

    const body = JSON.stringify("");
    const target = "calls/" + ref_id.current.value + "/hangup";
    console.log("Hangup call info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not hangup the call . err: %o", e);
        alert("Could not hangup the call.");
        setButtonDisable(false);
      });
  };

  const Delete = () => {
    console.log("Delete info");

    if (!confirm(`Are you sure you want to delete?`)) {
      return;
    }
    setButtonDisable(true);

    const body = JSON.stringify("");
    const target = "calls/" + ref_id.current.value;
    console.log("Deleting call info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not delete the call. err: %o", e);
        alert("Could not delete the call.");
        setButtonDisable(false);
      });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default CallsDetail
