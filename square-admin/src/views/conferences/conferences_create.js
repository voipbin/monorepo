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


const ConferencesCreate = () => {
  console.log("ConferencesCreate");
  
  const [buttonDisable, setButtonDisable] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_type = useRef(null);
  const ref_timeout = useRef(null);
  const ref_data = useRef(null);
  const ref_pre_actions = useRef(null);
  const ref_post_actions = useRef(null);
  
  const tmp_item_name = "conferences_create_tmp";
  var pre_actions = JSON.parse('[]');
  var post_actions = JSON.parse('[]');

  const tmp = localStorage.getItem(tmp_item_name);
  if (tmp != null) {
    try {
      console.log("Check variable. tmp: %o", tmp);
      const detailData = JSON.parse(tmp);
      console.log("detailData", detailData);
  
      pre_actions = (detailData.pre_actions);
      post_actions = (detailData.post_actions);
    }
    catch(e) {
      console.log("could not parse the data.");
    }
  }

  console.log("Debug. uselocation: %o", location);
  if (location.state != null && location.state.actions != null) {
    if (location.state.target == "pre_actions") {
      pre_actions = location.state.actions;
    } else if (location.state.target == "post_actions") {
      post_actions = location.state.actions;
    }
  }

  const Create = () => {

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Create</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/conference.html" target="_blank">here</a>.</small>
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Type</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_type}
                      type="text"
                      id="colFormLabelSm"
                      options={[
                        { label: 'Conference', value: 'conference' },
                      ]}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Timeout</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_timeout}
                      type="number"
                      id="colFormLabelSm"
                      defaultValue={0}
                      pattern="[0-9]*"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Pre Actions</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CButton type="submit" disabled={buttonDisable} onClick={() => EditPreActions()}>Edit Pre Actions</CButton>
                  </CCol>

                  <CCol className="mb-3 align-items-auto">
                    <CAccordion activeItemKey={2}>
                      <CAccordionItem itemKey={1}>
                        <CAccordionHeader>
                          <b>Pre Actions(Raw)</b>
                        </CAccordionHeader>
                        <CAccordionBody>
                          <CFormTextarea
                            ref={ref_pre_actions}
                            type="text"
                            id="colFormLabelSm"
                            defaultValue={JSON.stringify(pre_actions, null, 2)}
                            rows={20}
                          />
                        </CAccordionBody>
                      </CAccordionItem>
                    </CAccordion>
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Post Actions</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CButton type="submit" disabled={buttonDisable} onClick={() => EditPostActions()}>Edit Post Actions</CButton>
                  </CCol>

                  <CCol className="mb-3 align-items-auto">
                    <CAccordion activeItemKey={2}>
                      <CAccordionItem itemKey={1}>
                        <CAccordionHeader>
                          <b>Post Actions(Raw)</b>
                        </CAccordionHeader>
                        <CAccordionBody>
                          <CFormTextarea
                            ref={ref_post_actions}
                            type="text"
                            id="colFormLabelSm"
                            defaultValue={JSON.stringify(post_actions, null, 2)}
                            rows={20}
                          />
                        </CAccordionBody>
                      </CAccordionItem>
                    </CAccordion>
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Data</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_data}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="{}"
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
      "type": ref_type.current.value,
      "timeout": Number(ref_timeout.current.value),
      "pre_actions": JSON.parse(ref_pre_actions.current.value),
      "post_actions": JSON.parse(ref_post_actions.current.value),
      "data": JSON.parse(ref_data.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "conferences";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created info.", JSON.stringify(response));

        localStorage.setItem(tmp_item_name, null);

        const navi = "/resources/conferences/conferences_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not create the info. err: %o", e);
        alert("Could not create the info.");
        setButtonDisable(false);
      });
  };

  const EditPreActions = () => {
    console.log("Edit pre actions info. pre_actions: %o, post_actions: %o", ref_pre_actions.current.value, ref_post_actions.current.value);

    const tmp = {
      "pre_actions": JSON.parse(ref_pre_actions.current.value),
      "post_actions": JSON.parse(ref_post_actions.current.value),
    };
    const tmpItem = JSON.stringify(tmp);
    localStorage.setItem(tmp_item_name, tmpItem);

    const navi = "/resources/actiongraphs/actiongraph";
    const state = {
      state: {
        "referer": location.pathname,
        "target": "pre_actions",
        "actions": JSON.parse(ref_pre_actions.current.value),
      }
    }

    console.log("move to action graph. data: %o", state);
    navigate(navi, state);
  };

  const EditPostActions = () => {
    console.log("Edit post actions info. ", ref_post_actions.current.value);

    const tmp = {
      pre_actions: JSON.parse(ref_pre_actions.current.value),
      post_actions: JSON.parse(ref_post_actions.current.value),
    }
    const tmpItem = JSON.stringify(tmp);
    localStorage.setItem(tmp_item_name, tmpItem);

    const navi = "/resources/actiongraphs/actiongraph";
    const state = {
      state: {
        "referer": location.pathname,
        "target": "post_actions",
        "actions": JSON.parse(ref_post_actions.current.value),
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

export default ConferencesCreate
