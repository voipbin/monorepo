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

const RoutesCreate = () => {
  console.log("RoutesCreate");

  const routeParams = useParams();

  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_customer_id = useRef(routeParams.id);
  const ref_provider_id = useRef(null);
  const ref_priority = useRef(null);
  const ref_target = useRef(null);


  const Create = () => {

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Create</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs" target="_blank">here</a>.</small>
              </CCardHeader>

              <CCardBody>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Name</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_name}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue=""
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Detail</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_detail}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue=""
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Customer ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_customer_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="00000000-0000-0000-0000-000000000001"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Provider ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_provider_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue=""
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Priority</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_priority}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="0"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Target</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_target}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="all"
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
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "customer_id": ref.customer_id.current.value,
      "provider_id": ref_provider_id.current.value,
      "priority": Number(ref_priority.current.value),
      "target": ref_target.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "routes";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created info.", JSON.stringify(response));
      })
      .catch(e => {
        console.log("Could not create a new route. err: %o", e);
        alert("Could not not create a new route.");
      });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default RoutesCreate
