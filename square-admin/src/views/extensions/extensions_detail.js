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

const ExtensionsDetail = () => {
  console.log("ExtensionsDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();
  const agent = JSON.parse(localStorage.getItem("agent_info"));

  const ref_id = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_extension = useRef(null);
  const ref_password = useRef(null);

  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("extensions");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    console.log("detailData", detailData);

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details about extension at <a href="https://api.voipbin.net/docs/extension.html" target="_blank">here</a>.</small>
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Extension</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_extension}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.extension}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Password</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_password}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.password}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Registration address</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.extension + "@" + agent["customer_id"] + ".registrar.voipbin.net"}
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
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.tm_update}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateBasic()}>Update</CButton>
                &nbsp;
                <CButton type="submit" color="dark" disabled={buttonDisable} onClick={() => Delete()}>Delete</CButton>

              </CCardBody>
            </CCard>
          </CCol>
        </CRow>
      </>
    )
  };

  const UpdateBasic = () => {
    console.log("Update UpdateBasic");
    setButtonDisable(true);

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "password": ref_password.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "extensions/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        const navi = "/resources/extensions/extensions_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not update the info. err: %o", e);
        alert("Could not update the info.");
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
    const target = "extensions/" + ref_id.current.value;
    console.log("Deleting extension info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        const navi = "/resources/extensions/extensions_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not delete the info. err: %o", e);
        alert("Could not delete the info.");
        setButtonDisable(false);
      });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default ExtensionsDetail
