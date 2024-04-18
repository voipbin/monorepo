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
import { useSelector } from 'react-redux'

const AgentsProfile = () => {
  console.log("AgentsProfile");

  const [buttonDisable, setButtonDisable] = useState(false);
  const navigate = useNavigate();

  const ref_id = useRef(null);
  const ref_username = useRef(null);
  const ref_status = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_password = useRef("");
  const ref_password_check = useRef("");
  const ref_ring_method = useRef(null);
  const ref_addresses = useRef(null);
  const ref_tag_ids = useRef(null);

  var permission_val = 0;

  const routeParams = useParams();
  const GetDetail = () => {

    const detailData = JSON.parse(localStorage.getItem("agent_info"));
    console.log("Detailed agent info. agent_info: ", detailData);
  
    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/agent.html" target="_blank">here</a>.</small>
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
                  <CFormLabel className="col-sm-2 col-form-label"><b>Username</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_username}
                      type="text"
                      defaultValue={detailData.username}
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Ring Method</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_ring_method}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.ring_method}
                      options={[
                        { label: 'ringall', value: 'ringall' },
                      ]}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateBasic()}>Update</CButton>
                <br />
                <br/>


                <CRow>
                  <CFormLabel className="col-sm-2 col-form-label"><b>Status</b></CFormLabel>
                  <CCol>
                    <CFormSelect
                      ref={ref_status}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.status}
                      options={[
                        { label: 'available', value: 'available' },
                        { label: 'away', value: 'away' },
                        { label: 'busy', value: 'busy' },
                        { label: 'offline', value: 'offline' },
                        { label: 'ringing', value: 'ringing' },
                      ]}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateStatus()}>Update Status</CButton>
                <br />
                <br/>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Password</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                  <CFormInput
                      ref={ref_password}
                      type="password"
                      id="colFormLabelSm"
                      defaultValue={"dummy_default_password_with_long_length_0"}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Password Check</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                  <CFormInput
                    ref={ref_password_check}
                    type="password"
                    id="colFormLabelSm"
                    defaultValue={"dummy_default_password_with_long_length_1"}
                    />
                  </CCol>
                </CRow>


                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdatePassword()}>Update Password</CButton>
                <br />
                <br/>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Permission</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData["permission"]}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Addresses</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_addresses}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.addresses, null, 2)}
                      rows={15}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Tag IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_tag_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.tag_ids, null, 2)}
                      rows={5}
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

  const UpdateBasic = () => {
    console.log("Update info");
    setButtonDisable(true);

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "ring_method": ref_ring_method.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the basic info. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdateStatus = () => {
    console.log("Update status info");
    setButtonDisable(true);

    const tmpData = {
      "status": ref_status.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value + "/status";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the statue. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdatePassword = () => {
    console.log("Update password info");
    
    if (ref_password.current.value !== ref_password_check.current.value) {
      confirm(`Please check the password.`)
      return;
    }
    setButtonDisable(true);

    const tmpData = {
      "password": ref_password.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value + "/password";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the password. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default AgentsProfile
