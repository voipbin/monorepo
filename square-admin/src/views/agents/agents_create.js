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

const AgentsCreate = () => {
  console.log("AgentsCreate");

  const [buttonDisable, setButtonDisable] = useState(false);
  const navigate = useNavigate();

  const ref_username = useRef(null);
  const ref_password = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_ring_method = useRef(null);
  const ref_addresses = useRef(null);
  const ref_tag_ids = useRef(null);

  var permission_val = 16;

  const Create = () => {
    var selectedAgent = true;
    var selectedManager = false;
    var selectedAdmin = false;

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
                <strong>Create</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/agent.html" target="_blank">here</a>.</small>
              </CCardHeader>

              <CCardBody>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Username</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_username}
                      type="text"
                      id="colFormLabelSm"
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Password</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_password}
                      type="password"
                      id="colFormLabelSm"
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
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Detail</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_detail}
                      type="text"
                      id="colFormLabelSm"
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
                      options={[
                        { label: 'ringall', value: 'ringall' },
                      ]}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Permission</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <select class="form-select" multiple aria-label="multiple select example" size="3" onChange={onChangeSelect}>
                      <option value="16" selected={selectedAgent}>Agent</option>
                      <option value="32" selected={selectedManager}>Manager</option>
                      <option value="64" selected={selectedAdmin}>Admin</option>
                    </select>
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Addresses</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_addresses}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="[]"
                      rows={20}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Tag IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_tag_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="[]"
                      rows={20}
                    />
                  </CCol>
                </CRow>


                <CButton type="submit" disabled={buttonDisable} onClick={() => CreateResource()}>Create</CButton>

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

  const CreateResource = () => {
    console.log("Create info");
    setButtonDisable(true);

    const tmpData = {
      "username": ref_username.current.value,
      "password": ref_password.current.value,
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "ring_method": ref_ring_method.current.value,
      "permission": Number(permission_val),
      "addresses": JSON.parse(ref_addresses.current.value),
      "tag_ids": JSON.parse(ref_tag_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created info.", JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not create the agent. err: %o", e);
        alert("Could not create the agent.");
        setButtonDisable(false);
      });

  };

  return (
    <>
      <Create/>
    </>
  )
}

export default AgentsCreate
