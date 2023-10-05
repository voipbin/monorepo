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

const ExtensionsCreate = () => {
  console.log("CustomersCreate");

  const ref_extension = useRef(null);
  const ref_password = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_email = useRef(null);
  const ref_phone_number = useRef(null);
  const ref_address = useRef(null);
  const ref_webhook_uri = useRef(null);
  const ref_webhook_method = useRef(null);
  const ref_permission_ids = useRef(null);

  const routeParams = useParams();
  const Create = () => {
    const id = routeParams.id;

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>Detail of the resource</small>
              </CCardHeader>

              <CCardBody>

              <CRow>
                  <CFormLabel className="col-sm-2 col-form-label"><b>*Extension</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_extension}
                      type="text"
                      id="colFormLabelSm"
                    />
                  </CCol>

                  <CFormLabel className="col-sm-2 col-form-label"><b>*Password</b></CFormLabel>
                  <CCol>
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
                  <CCol>
                    <CFormInput
                      ref={ref_detail}
                      type="text"
                      id="colFormLabelSm"
                    />
                  </CCol>
                </CRow>


              </CCardBody>
            </CCard>
          </CCol>
        </CRow>

        <CButton type="submit" onClick={() => CreateResource()}>Create</CButton>
      </>
    )
  };

  const CreateResource = () => {
    console.log("Create info");

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "extension": ref_extension.current.value,
      "password": ref_password.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "extensions";
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

export default ExtensionsCreate
