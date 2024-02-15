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

const ActiveflowsCreate = () => {
  console.log("ActiveflowsCreate");

  const ref_id = useRef(null);
  const ref_flow_id = useRef(null);
  const ref_actions = useRef(null);

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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="00000000-0000-0000-0000-000000000000"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Flow ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_flow_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="00000000-0000-0000-0000-000000000000"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Actions</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_actions}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="[]"
                      rows={20}
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
      "id": ref_id.current.value,
      "flow_id": ref_flow_id.current.value,
      "actions": JSON.parse(ref_actions.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "activeflows";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body).then((response) => {
      console.log("Created info.", JSON.stringify(response));
    });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default ActiveflowsCreate
