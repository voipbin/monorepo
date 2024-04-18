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
} from '@coreui/react'
import store from '../../store'
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  Download as ProviderDownload,
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";

const SubscriptionsDetail = () => {
  console.log("SubscriptionsDetail");

  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_id = useRef(null);

  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("recordings");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];

    const startTime = Date.parse(detailData.tm_start);
    const endTime = Date.parse(detailData.tm_end);
    const recordingTime = endTime - startTime;

    console.log("Start recording time: %o, time: %o, timestamp: %s", recordingTime, startTime, detailData.tm_start);

    return (
      <CRow>
        <CCol xs={12}>
          <CCard className="mb-4">
            <CCardHeader>
              <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/recording.html" target="_blank">here</a>.</small>
            </CCardHeader>

            <CCardBody>
              <CRow>
                <CFormLabel className="col-sm-2 col-form-label"><b>ID</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    ref={ref_id}
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.id}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel className="col-sm-2 col-form-label"><b>Status</b></CFormLabel>
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
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Reference ID</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.reference_id}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Reference Type</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.reference_type}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Format</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.format}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Recording duration(Sec)</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={recordingTime / 1000}
                    readOnly plainText
                  />
                </CCol>
              </CRow>


              <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Start Timestamp</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.tm_start}
                    readOnly plainText
                  />
                </CCol>

                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>End Timestamp</b></CFormLabel>
                <CCol>
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.tm_end}
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
              <CButton type="submit" onClick={() => Download()}>Download</CButton>
              &nbsp;
              <CButton type="submit" color="dark" onClick={() => Delete()}>Delete</CButton>

            </CCardBody>
          </CCard>
        </CCol>
      </CRow>
    )
  };

  const navigateList = () => {
    const navi = "/resources/recordings/recordings_list";
    navigate(navi);
  }

  const Download = () => {
    console.log("Download");
    const body = JSON.stringify("");
    const target = "recordingfiles/" + ref_id.current.value;

    const filename = ref_id.current.value + ".zip"
    console.log("Download recording file. target: %s, filename: %s", target, filename);

    ProviderDownload(target, filename);
  };

  const Delete = () => {
    console.log("Delete info");

    if (!confirm(`Are you sure you want to delete?`)) {
      return;
    }

    const body = JSON.stringify("");
    const target = "recordings/" + ref_id.current.value;
    console.log("Deleting recording info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        navigateList();
      })
      .catch(e => {
        console.log("Could not delete the recording. err: %o", e);
        alert("Could not not delete the recording.");
      });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default SubscriptionsDetail
