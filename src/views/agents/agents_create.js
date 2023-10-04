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

const AgentsCreate = () => {
  console.log("AgentsCreate");

  const ref_username = useRef(null);
  const ref_password = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_ring_method = useRef(null);
  const ref_permission = useRef(null);
  const ref_addresses = useRef(null);
  const ref_tag_ids = useRef(null);

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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Username</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_username}
                      type="text"
                      id="colFormLabelSm"
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Password</b></CFormLabel>
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
                  <CCol>
                    <CFormInput
                      ref={ref_permission}
                      type="text"
                      id="colFormLabelSm"
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
      "username": ref_username.current.value,
      "password": ref_password.current.value,
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "ring_method": ref_ring_method.current.value,
      "permission": Number(ref_permission.current.value),
      "addresses": JSON.parse(ref_addresses.current.value),
      "tag_ids": JSON.parse(ref_tag_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body).then(() => {
      console.log("Created info.");
    });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default AgentsCreate
