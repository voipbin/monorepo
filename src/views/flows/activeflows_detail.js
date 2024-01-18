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
} from '@coreui/react'
import store from '../../store'
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';

const ActiveflowsDetail = () => {
  console.log("ActiveflowsDetail");

  const ref_id = useRef(null);

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("activeflows");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];

    var stopDisabled = false;
    if (detailData["status"] == "ended") {
      stopDisabled = true;
    }

    return (
      <CRow>
        <CCol xs={12}>
          <CCard className="mb-4">
            <CCardHeader>
              <strong>Detail</strong> <small>Detail of the resource</small>
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
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Flow ID</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={detailData.flow_id}
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
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Current Action</b></CFormLabel>
                <CCol className="mb-3 align-items-auto">
                  <CFormTextarea
                    type="text"
                    id="colFormLabelSm"
                    defaultValue={JSON.stringify(detailData.current_action, null, 2)}
                    rows={10}
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
              <CButton type="submit" disabled={stopDisabled} onClick={() => Stop()}>Stop</CButton>
              &nbsp;
              <CButton type="submit" color="dark" onClick={() => Delete()}>Delete</CButton>

            </CCardBody>
          </CCard>
        </CCol>
      </CRow>
    )
  };

  const Stop = () => {
    console.log("Stop");
    const body = JSON.stringify("");
    const target = "activeflows/" + ref_id.current.value + "/stop";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPost(target, body).then(response => {
      console.log("Updated info. response: " + JSON.stringify(response));
    });
  };

  const Delete = () => {
    console.log("Delete info");

    if (!confirm(`Are you sure you want to delete?`)) {
      return;
    }

    const body = JSON.stringify("");
    const target = "activeflows/" + ref_id.current.value;
    console.log("Deleting activeflow info. target: " + target + ", body: " + body);
    ProviderDelete(target, body).then(response => {
      console.log("Deleted info. response: " + JSON.stringify(response));
    });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default ActiveflowsDetail
