import React, { useRef, useState } from 'react'
import { useLocation } from "react-router-dom";
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
  CAccordion,
  CAccordionBody,
  CAccordionHeader,
  CAccordionItem,
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

const QueuesCreate = () => {
  console.log("QueuesCreate");

  const [buttonDisable, setButtonDisable] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_routing_method = useRef(null);
  const ref_service_timeout = useRef(null);
  const ref_wait_timeout = useRef(null);
  const ref_wait_actions = useRef(null);
  const ref_tag_ids = useRef(null);

  var wait_actions = JSON.parse('[]');
  if (location.state != null && location.state.actions != null) {
    wait_actions = location.state.actions;
  }

  const Create = () => {

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/queue.html" target="_blank">here</a>.</small>
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
                    <CButton type="submit" disabled={buttonDisable} onClick={() => EditWaitActions()}>Edit Wait Actions</CButton>
                  </CCol>

                  <CCol className="mb-3 align-items-auto">
                    <CAccordion activeItemKey={2}>
                      <CAccordionItem itemKey={1}>
                        <CAccordionHeader>
                          <b>Wait Actions(Raw)</b>
                        </CAccordionHeader>
                        <CAccordionBody>
                          <CFormTextarea
                            ref={ref_wait_actions}
                            type="text"
                            id="colFormLabelSm"
                            defaultValue={JSON.stringify(wait_actions, null, 2)}
                            rows={20}
                          />
                        </CAccordionBody>
                      </CAccordionItem>
                    </CAccordion>
                  </CCol>
                </CRow>


                <CRow>
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
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created info.", JSON.stringify(response));
        const navi = "/resources/queues/queues_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not create a new queue. err: %o", e);
        alert("Could not not create a new queue.");
        setButtonDisable(false);
      });
  };

  const EditWaitActions = () => {
    console.log("Edit actions info. ", ref_wait_actions.current.value);

    const navi = "/resources/actiongraphs/actiongraph";
    const state = {
      state: {
        "referer": location.pathname,
        "target": "wait_actions",
        "actions": JSON.parse(ref_wait_actions.current.value),
      }
    }

    console.log("move to action graph. data: %o", state);
    navigate(navi, state);
  };


  return (
    <>
      <Create/>
    </>
  )
}

export default QueuesCreate
