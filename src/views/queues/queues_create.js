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

const QueuesCreate = () => {
  console.log("QueuesCreate");

  const ref_id = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_routing_method = useRef(null);
  const ref_service_timeout = useRef(null);
  const ref_wait_timeout = useRef(null);
  const ref_service_queuecall_ids = useRef(null);
  const ref_wait_queuecall_ids = useRef(null);
  const ref_wait_actions = useRef(null);
  const ref_tag_ids = useRef(null);
  const ref_total_abandoned_count = useRef(null);
  const ref_total_incoming_count = useRef(null);
  const ref_total_serviced_count = useRef(null);

  const routeParams = useParams();
  const Create = () => {

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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Routing Method</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_routing_method}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="random"
                      options={[
                        { label: 'random', value: 'random' },
                      ]}
                    />
                  </CCol>
                </CRow>



                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Wait Timeout(ms)</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_wait_timeout}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={0}
                      pattern="[0-9]*"
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Service Timeout(ms)</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_service_timeout}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={0}
                      pattern="[0-9]*"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Wait Actions</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_wait_actions}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="[]"
                      rows={5}
                    />
                  </CCol>


                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Tag IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_tag_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="[]"
                      rows={5}
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
      "routing_method": ref_routing_method.current.value,
      "service_timeout": Number(ref_service_timeout.current.value),
      "wait_timeout": Number(ref_wait_timeout.current.value),
      "wait_actions": JSON.parse(ref_wait_actions.current.value),
      "tag_ids": JSON.parse(ref_tag_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "queues";
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

export default QueuesCreate
