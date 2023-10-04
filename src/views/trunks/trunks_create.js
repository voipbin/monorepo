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

const TrunkCreate = () => {
  console.log("TrunkCreate");

  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_domain_name = useRef(null);
  const ref_auth_types = useRef(null);
  const ref_username = useRef(null);
  const ref_password = useRef(null);
  const ref_allowed_ips = useRef(null);

  const routeParams = useParams();
  const Create = () => {
    const id = routeParams.id;

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Create</strong> <small>Creating resource</small>
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
                    <CFormTextarea
                      ref={ref_auth_types}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.auth_types, null, 2)}
                      rows={5}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Username</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_username}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.username}
                    />
                  </CCol>


                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Password</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_password}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.password}
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
                      defaultValue={JSON.stringify(detailData.allowed_ips, null, 2)}
                      rows={5}
                    />
                  </CCol>
                </CRow>



                <CButton type="submit" onClick={() => CreateResource()}>Create</CButton>

          </CCardBody>
        </CCard>
      </CCol>
      </CRow>

      </>
    )
  };

  const CreateResource = () => {
    console.log("Create info");

    const tmpData = {
      "domain_name": ref_domain_name.current.value,
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "auth_types": JSON.parse(ref_auth_types.current.value),
      "username": ref_username.current.value,
      "password": ref_password.current.value,
      "allowed_ips": JSON.parse(ref_allowed_ips.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "trunks";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body).then((response) => {
      console.log("Created info.", response);
    });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default TrunkCreate
