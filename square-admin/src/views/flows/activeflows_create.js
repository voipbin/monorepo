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

const ActiveflowsCreate = () => {
  console.log("ActiveflowsCreate");

  const [buttonDisable, setButtonDisable] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  const ref_id = useRef(null);
  const ref_flow_id = useRef(null);
  const ref_actions = useRef(null);

  console.log("Debug. uselocation: %o", location);
  var actions = JSON.parse('[]');
  if (location.state != null && location.state.actions != null) {
    actions = location.state.actions;
  }

  const Create = () => {

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Create</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/flow.html" target="_blank">here</a>.</small>
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
                    <CButton type="submit" disabled={buttonDisable} onClick={() => EditActions()}>Edit Actions</CButton>
                  </CCol>

                  <CCol className="mb-3 align-items-auto">
                    <CAccordion activeItemKey={2}>
                      <CAccordionItem itemKey={1}>
                        <CAccordionHeader>
                          <b>Actions(Raw)</b>
                        </CAccordionHeader>
                        <CAccordionBody>
                          <CFormTextarea
                            ref={ref_actions}
                            type="text"
                            id="colFormLabelSm"
                            defaultValue={JSON.stringify(actions, null, 2)}
                            rows={20}
                          />
                        </CAccordionBody>
                      </CAccordionItem>
                    </CAccordion>
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
      "id": ref_id.current.value,
      "flow_id": ref_flow_id.current.value,
      "actions": JSON.parse(ref_actions.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "activeflows";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created info.", JSON.stringify(response));
        const navi = "/resources/flows/activeflows_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not create a new activeflow. err: %o", e);
        alert("Could not not create a new activeflow.");
        setButtonDisable(false);
      });
  };

  const EditActions = () => {
    console.log("Edit actions info. ", ref_actions.current.value);

    const navi = "/resources/actiongraphs/actiongraph";
    const state = {
      state: {
        "referer": location.pathname,
        "target": "actions",
        "actions": JSON.parse(ref_actions.current.value),
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

export default ActiveflowsCreate
