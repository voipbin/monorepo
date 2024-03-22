import React, { useRef, useState } from 'react'
import { useParams, useLocation } from "react-router-dom";
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

const ConferencesDetail = () => {
  console.log("ConferencesDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  
  const ref_id = useRef(null);
  const ref_balance = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_timeout = useRef(null);
  const ref_data = useRef(null);
  const ref_status = useRef(null);
  const ref_pre_actions = useRef(null);
  const ref_post_actions = useRef(null);
  const ref_conferencecall_ids = useRef(null);
  const ref_recording_ids = useRef(null);

  const id = routeParams.id;
  const tmp_item_name = "conferences_" + id + "_tmp";

  const GetDetail = () => {

    const tmp = localStorage.getItem("conferences");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    console.log("detailData", detailData);

    const tmpItem = localStorage.getItem(tmp_item_name);
    if (tmp != null) {
      try {
        console.log("Check variable. tmpItem: %o", tmpItem);
        const tmpData = JSON.parse(tmpItem);
        console.log("detailData", tmpData);
    
        detailData.pre_actions = (tmpData.pre_actions);
        detailData.post_actions = (tmpData.post_actions);
      }
      catch(e) {
        console.log("could not parse the data.");
      }
    }

    console.log("Debug. uselocation: %o", location);
    if (location.state != null && location.state.actions != null) {
      if (location.state.target == "pre_actions") {
        detailData.pre_actions = location.state.actions;
      } else if (location.state.target == "post_actions") {
        detailData.post_actions = location.state.actions;
      }
    }


    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/conference.html" target="_blank">here</a>.</small>
              </CCardHeader>

              <CCardBody>
                <CRow>
                  <CFormLabel className="col-sm-2 col-form-label"><b>ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_id}
                      type="text"
                      id="id"
                      defaultValue={detailData.id}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel className="col-sm-2 col-form-label"><b>Type</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_balance}
                      type="text"
                      id="id"
                      defaultValue={detailData.type}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Status</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_status}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.status}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Timeout</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_timeout}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.timeout}
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
                      defaultValue={detailData.name}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Detail</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_detail}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.detail}
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
                            defaultValue={JSON.stringify(detailData.pre_actions, null, 2)}
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
                            defaultValue={JSON.stringify(detailData.post_actions, null, 2)}
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
                      defaultValue={JSON.stringify(detailData.data, null, 2)}
                      rows={5}
                    />
                  </CCol>
                </CRow>


                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateBasicInfo()}>Update</CButton>
                <br />
                <br />

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Conferencecall IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_conferencecall_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.conferencecall_ids, null, 2)}
                      rows={5}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Recording IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_recording_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.recording_ids, null, 2)}
                      rows={5}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Create Timestamp</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.tm_create}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Update Timestamp</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.tm_update}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" color="dark" disabled={buttonDisable} onClick={() => Delete()}>Delete</CButton>

              </CCardBody>
            </CCard>
          </CCol>
        </CRow>

      </>
    )
  };

  const UpdateBasicInfo = () => {
    console.log("Update info");
    setButtonDisable(true);

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "timeout": Number(ref_timeout.current.value),
      "pre_actions": JSON.parse(ref_pre_actions.current.value),
      "post_actions": JSON.parse(ref_post_actions.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "conferences/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then((response) => {
        console.log("Updated info.", response);

        localStorage.setItem(tmp_item_name, null);

        const navi = "/resources/conferences/conferences_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not update the info. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const Delete = () => {
    console.log("Delete info");

    if (!confirm(`Are you sure you want to delete?`)) {
      return;
    }
    setButtonDisable(true);

    const body = JSON.stringify("");
    const target = "conferences/" + ref_id.current.value;
    console.log("Deleting conference info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));

        localStorage.setItem(tmp_item_name, null);

        const navi = "/resources/conferences/conferences_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not delete the info. err: %o", e);
        alert("Could not delete the info.");
        setButtonDisable(false);
      });
  }

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
      <GetDetail/>
    </>
  )
}

export default ConferencesDetail
