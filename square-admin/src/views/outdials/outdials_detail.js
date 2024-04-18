import React, { useMemo, useRef, useState, useEffect } from 'react'
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

const OutdialsDetail = () => {
  console.log("OutdialsDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_id = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_data = useRef(null);

  const id = routeParams.id;

  const GetDetail = () => {

    const tmp = localStorage.getItem("outdials");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    console.log("detailData", detailData);

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/campaign.html" target="_blank">here</a>.</small>
              </CCardHeader>

              <CCardBody>
                <CRow>
                  <CFormLabel className="col-sm-2 col-form-label"><b>ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_id}
                      type="text"
                      id="id"
                      defaultValue={detailData.id}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Name</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_name}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.name}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Detail</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_detail}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.detail}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Data</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_data}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.data}
                      rows={5}
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


                <CRow>
                  <CButton type="submit" color="info" disabled={buttonDisable} onClick={() => ListOutdialtargets()}>Outdial target list</CButton>
                </CRow>

                <br/>
                <CButton type="submit" onClick={() => UpdateBasicInfo()}>Update</CButton>
                &nbsp;
                <CButton type="submit" color="dark" disabled={buttonDisable} onClick={() => Delete()}>Delete</CButton>


              </CCardBody>
            </CCard>
          </CCol>
        </CRow>
      </>
    )
  };

  const navigateBack = () => {
    navigate(-1);
  }

  const navigateTargetList = () => {
    const navi = "/resources/outdials/" + id + "/outdialtargets_list";
    navigate(navi);
  }

  const UpdateBasicInfo = () => {
    console.log("Update info");
    setButtonDisable(true);

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "data": ref_data.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "outdials/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then((response) => {
        console.log("Updated info.", JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the outdial info. err: %o", e);
        alert("Could not not update the outdial info.");
        setButtonDisable(false);
      });
  };

  const ListOutdialtargets = () => {
    console.log("ListOutdialtargets");
    navigateTargetList();
  };

  const Delete = () => {
    console.log("Delete info");

    if (!confirm(`Are you sure you want to delete?`)) {
      return;
    }
    setButtonDisable(true);

    const body = JSON.stringify("");
    const target = "outdials/" + ref_id.current.value;
    console.log("Deleting outdial info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not delete the outdial. err: %o", e);
        alert("Could not not delete the outdial.");
        setButtonDisable(false);
      });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default OutdialsDetail
