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

const MessagesCreate = () => {
  console.log("MessagesCreate");

  const ref_source = useRef(null);
  const ref_destinations = useRef(null);
  const ref_text = useRef(null);


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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Source</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_source}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(JSON.parse('{"type":"tel", "target":""}'), null, 2)}
                      rows={10}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destinations</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_destinations}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(JSON.parse('[{"type":"tel", "target":""}]'), null, 2)}
                      rows={10}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Text</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_text}
                      type="text"
                      id="colFormLabelSm"
                      placeholder="Message here"
                      defaultValue=""
                      rows={10}
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
      "source": JSON.parse(ref_source.current.value),
      "destinations": JSON.parse(ref_destinations.current.value),
      "text": ref_text.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "messages";
    console.log("Creating message info. target: " + target + ", body: " + body);
    ProviderPost(target, body).then((response) => {
      console.log("Created message info.", JSON.stringify(response));
    });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default MessagesCreate
