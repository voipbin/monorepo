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

const AgentsDetail = () => {
  console.log("AgentsDetail");

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
    const id = routeParams.id;

    const tmp = localStorage.getItem("agents");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];

    var selectedAgent = false;
    var selectedManager = false;
    var selectedAdmin = false;

    if (detailData["permission"] & 16) {
      selectedAgent = true;
    }
    if (detailData["permission"] & 64) {
      selectedManager = true;
    }
    if (detailData["permission"] & 32) {
      selectedAdmin = true;
    }
    permission_val = detailData["permission"];

    const onChangeSelect = (e) => {
      var permission = 0;
      for (let i = 0; i < e.target.length; i++) {
        const tmp = e.target[i];
        if (tmp["selected"] == true) {
          permission += parseInt(tmp["value"]);
        }
      }
      permission_val = permission;
    }

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
                  <CCol className="mb-3 align-items-auto">
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
                    <select class="form-select" multiple aria-label="multiple select example" size="3" onChange={onChangeSelect}>
                      <option value="16" selected={selectedAgent}>Agent</option>
                      <option value="64" selected={selectedManager}>Manager</option>
                      <option value="32" selected={selectedAdmin}>Admin</option>
                    </select>
                  </CCol>
                </CRow>


                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdatePermission()}>Update Permission</CButton>
                <br />
                <br/>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Addresses</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_addresses}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.addresses, null, 2)}
                      rows={15}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateAddresse()}>Update Addresses</CButton>
                <br />
                <br/>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Tag IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_tag_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.tag_ids, null, 2)}
                      rows={15}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateTagIDs()}>Update Tag IDs</CButton>
                <br />
                <br/>

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
        console.log("Could not update the status. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdateAddresse = () => {
    console.log("Update addresses info");
    setButtonDisable(true);

    const tmpData = {
      "addresses": JSON.parse(ref_addresses.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value + "/addresses";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the address. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdateTagIDs = () => {
    console.log("Update tag ids info");
    setButtonDisable(true);

    const tmpData = {
      "tag_ids": JSON.parse(ref_tag_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value + "/tag_ids";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the tag ids. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdatePermission = () => {
    console.log("Update permission info");
    setButtonDisable(true);

    const tmpData = {
      "permission": JSON.parse(permission_val),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value + "/permission";
    console.log("Update info. target: " + target + ", body: " + body);

    ProviderPut(target, body)
      .then(response => {
        console.log("response: %o", response);
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the permission. err: %o", e);
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
      });;
  };

  const Delete = () => {
    console.log("Delete info");

    if (!confirm(`Are you sure you want to delete?`)) {
      return;
    }
    setButtonDisable(true);

    const body = JSON.stringify("");
    const target = "agents/" + ref_id.current.value;
    console.log("Deleting agent info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not delete the agent. err: %o", e);
        alert("Could not delete the agent.");
        setButtonDisable(false);
      });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default AgentsDetail
