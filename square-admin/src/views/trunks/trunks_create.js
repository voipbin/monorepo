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
  CListGroup,
  CListGroupItem,
  CFormCheck,
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

const TrunkCreate = () => {
  console.log("TrunkCreate");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_domain_name = useRef(null);
  const ref_auth_types = useRef(null);
  const ref_auth_types_basic = useRef(null);
  const ref_auth_types_ip = useRef(null);
  const ref_username = useRef(null);
  const ref_password = useRef(null);
  const ref_allowed_ips = useRef(null);

  const Create = () => {
    const id = routeParams.id;

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Create</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/trunk.html" target="_blank">here</a>.</small>
              </CCardHeader>

              <CCardBody>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Domain Name</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_domain_name}
                      type="text"
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
                  <CCol>
                    <CFormInput
                      ref={ref_detail}
                      type="text"
                      id="colFormLabelSm"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Auth Types</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CListGroup ref={ref_auth_types}>
                      <CListGroupItem>
                        <CFormCheck ref={ref_auth_types_basic} hitArea="full" label="basic" id="ref_auth_types_basic" value="" defaultChecked/>
                      </CListGroupItem>
                      <CListGroupItem>
                        <CFormCheck ref={ref_auth_types_ip} hitArea="full" label="ip" id="ref_auth_types_ip" value=""/>
                      </CListGroupItem>
                    </CListGroup>
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Username</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_username}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue=""
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Password</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_password}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue=""
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Allowed IPs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_allowed_ips}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="[]"
                      rows={5}
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

  const CreateResource = () => {
    console.log("Create info");
    setButtonDisable(true);

    let tmpAuth = []
    {
      if (ref_auth_types_basic.current.checked == true) {
        tmpAuth.push("basic");
      }
      if (ref_auth_types_ip.current.checked == true) {
        tmpAuth.push("ip");
      }
    }

    const tmpData = {
      "domain_name": ref_domain_name.current.value,
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "auth_types": JSON.parse(JSON.stringify(tmpAuth)),
      "username": ref_username.current.value,
      "password": ref_password.current.value,
      "allowed_ips": JSON.parse(ref_allowed_ips.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "trunks";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created info.", JSON.stringify(response));
        const navi = "/resources/trunks/trunks_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not create a new trunk. err: %o", e);
        alert("Could not not create a new trunk.");
        setButtonDisable(false);
      });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default TrunkCreate
