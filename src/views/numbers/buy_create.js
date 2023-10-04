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

const BuyCreate = () => {
  console.log("BuyCreate");

  const ref_number = useRef(null);
  const ref_password = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_billingaccount_id = useRef(null);
  const ref_call_flow_id = useRef(null);
  const ref_message_flow_id = useRef(null);
  const ref_permission_ids = useRef(null);

  const routeParams = useParams();
  const Create = () => {
    const number = routeParams.id;

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
                  <CFormLabel className="col-sm-2 col-form-label"><b>*Number</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_number}
                      type="text"
                      id="id"
                      defaultValue={number}
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Call Flow ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_call_flow_id}
                      type="text"
                      id="colFormLabelSm"
                    />
                  </CCol>


                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Message Flow ID</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_message_flow_id}
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
      "number": ref_number.current.value,
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "message_flow_id": ref_message_flow_id.current.value,
      "call_flow_id": ref_call_flow_id.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "customers";
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

export default BuyCreate
