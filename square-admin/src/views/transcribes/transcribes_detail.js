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
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";

const TranscribesDetail = () => {
  console.log("TranscribesDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_id = useRef(null);

  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("transcribes");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];

    var stopDisabled = false;
    if (detailData["status"] == "done") {
      stopDisabled = true;
    }

    return (
      <CRow>
        <CCol xs={12}>
          <CCard className="mb-4">
            <CCardHeader>
              <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/flow.html" target="_blank">here</a>.</small>
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
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Language</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.language}
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
                <CButton type="submit" onClick={() => Transcripts()}>Transcripts</CButton>
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
              <CButton type="submit" disabled={stopDisabled} onClick={() => Stop()}>Stop</CButton>
              &nbsp;
              <CButton type="submit" color="dark" disabled={buttonDisable} onClick={() => Delete()}>Delete</CButton>

            </CCardBody>
          </CCard>
        </CCol>
      </CRow>
    )
  };

  const navigateList = () => {
    const navi = "/resources/transcribes/transcribes_list";
    navigate(navi);
  }

  const Stop = () => {
    console.log("Stop");
    const body = JSON.stringify("");
    const target = "transcribes/" + ref_id.current.value + "/stop";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateList();
      })
      .catch(e => {
        console.log("Could not stop the transcribe. err: %o", e);
        alert("Could not not stop the transcribes.");
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
    const target = "transcribes/" + ref_id.current.value;
    console.log("Deleting transcribe info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        navigateList();
      })
      .catch(e => {
        console.log("Could not delete the transcribe. err: %o", e);
        alert("Could not not delete the transcribe.");
        setButtonDisable(false);
      });
  }

  const Transcripts = () => {
    console.log("Transcripts info");

    const target = "/resources/transcribes/" + ref_id.current.value + "/transcripts_list";
    console.log("navigate target: ", target);
    navigate(target);
  };


  return (
    <>
      <GetDetail/>
    </>
  )
}

export default TranscribesDetail
