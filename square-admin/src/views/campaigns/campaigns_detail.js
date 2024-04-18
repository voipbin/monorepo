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

const CampaignsDetail = () => {
  console.log("CampaignsDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();
  const location = useLocation();

  const ref_id = useRef(null);
  const ref_status = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_service_level = useRef(null);
  const ref_type = useRef(null);
  const ref_end_handle = useRef(null);
  const ref_next_campaign_id = useRef(null);
  const ref_outdial_id = useRef(null);
  const ref_outplan_id = useRef(null);
  const ref_queue_id = useRef(null);
  const ref_actions = useRef(null);

  const id = routeParams.id;

  const tmp = localStorage.getItem("campaigns");
  const datas = JSON.parse(tmp);
  const detailData = datas[id];
  console.log("detailData", detailData);

  // parse the location
  console.log("Debug. uselocation: %o", location);
  if (location.state != null && location.state.actions != null) {
    detailData.actions = location.state.actions;
  }
  
  const GetDetail = () => {

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/campaign.html" target="_blank">here</a>.</small>
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Type</b></CFormLabel>
                  <CCol>
                    <CFormSelect
                      ref={ref_type}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.type}
                      options={[
                        { label: 'call', value: 'call' },
                        { label: 'flow', value: 'flow' },
                      ]}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Service Level</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_service_level}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.service_level}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>End Handle</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_end_handle}
                      type="text"
                      id="colFormLabelSm"
                      options={[
                        { label: 'stop', value: 'stop' },
                        { label: 'continue', value: 'continue' },
                      ]}
                    />
                  </CCol>
                </CRow>


                <CButton type="submit" disabled={buttonDisable} onClick={() => Update()}>Update</CButton>
                <br />
                <br />


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Outplan ID</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_outplan_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.outplan_id}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Outdial ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_outdial_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.outdial_id}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Queue ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_queue_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.queue_id}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Next Campaign ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_next_campaign_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.next_campaign_id}
                    />
                  </CCol>
                </CRow>


                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateResource()}>Update Resource</CButton>
                <br />
                <br />
                

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
                            defaultValue={JSON.stringify(detailData.actions, null, 2)}
                            rows={20}
                          />
                        </CAccordionBody>
                      </CAccordionItem>
                    </CAccordion>
                  </CCol>
                </CRow>



                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateActions()}>Update Actions</CButton>
                <br />
                <br />


                <CRow>
                  <CFormLabel className="col-sm-2 col-form-label"><b>Status</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                  <CFormSelect
                      ref={ref_status}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.status}
                      options={[
                        { label: 'run', value: 'run' },
                        { label: 'stop', value: 'stop' },
                      ]}
                    />
                  </CCol>

                  <CCol>
                    <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateStatus()}>Update Status</CButton>
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


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Delete Timestamp</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.tm_delete}
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

  const navigateList = () => {
    const navi = "/resources/campaigns/campaigns_list";
    navigate(navi);
  }

  const Update = () => {
    console.log("Update info");
    setButtonDisable(true);

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "service_level": Number(ref_service_level.current.value),
      "type": ref_type.current.value,
      "end_handle": ref_end_handle.current.value,
      "next_campaign_id": ref_next_campaign_id.current.value,
      "outdial_id": ref_outdial_id.current.value,
      "outplan_id": ref_outplan_id.current.value,
      "queue_id": ref_queue_id.current.value,
      "actions": JSON.parse(ref_actions.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "campaigns/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateList();
      })
      .catch(e => {
        console.log("Could not update the basic info. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdateResource = () => {
    console.log("UpdateResource");
    setButtonDisable(true);

    const tmpData = {
      "outplan_id": ref_outplan_id.current.value,
      "outdial_id": ref_outdial_id.current.value,
      "queue_id": ref_queue_id.current.value,
      "next_campaign_id": ref_next_campaign_id.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "campaigns/" + ref_id.current.value + "/resource_info";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateList();
      })
      .catch(e => {
        console.log("Could not update the id info. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdateActions = () => {
    console.log("UpdateActions");
    setButtonDisable(true);

    const tmpData = {
      "actions": JSON.parse(ref_actions.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "campaigns/" + ref_id.current.value + "/actions";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateList();
      })
      .catch(e => {
        console.log("Could not update the action info. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdateStatus = () => {
    console.log("UpdateStatus");
    setButtonDisable(true);

    const tmpData = {
      "status": ref_status.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "campaigns/" + ref_id.current.value + "/status";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        navigateList();
      })
      .catch(e => {
        console.log("Could not update the status info. err: %o", e);
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
    const target = "campaigns/" + ref_id.current.value;
    console.log("Deleting campaign info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        navigateList();
      })
      .catch(e => {
        console.log("Could not delete the campaign info. err: %o", e);
        alert("Could not delete the campaign info.");
        setButtonDisable(false);
      });
  }

  const EditActions = () => {
    console.log("Edit actions info");

    const navi = "/resources/actiongraphs/actiongraph";
    const state = {
      state: {
        "referer": location.pathname,
        "target": "actions",
        "actions": detailData.actions,
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

export default CampaignsDetail
