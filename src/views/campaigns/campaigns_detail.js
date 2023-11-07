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

const CampaignsDetail = () => {
  console.log("CampaignsDetail");

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

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("campaigns");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    console.log("detailData", detailData);

    // const storeData = store.getState();
    // const detailData = storeData["campaigns"][id];
    // console.log("detailData", detailData);

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



                <CButton type="submit" onClick={() => Update()}>Update</CButton>
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


                <CButton type="submit" onClick={() => UpdateResource()}>Update Resource</CButton>
                <br />
                <br />


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Actions</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_actions}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.actions, null, 2)}
                      rows={15}
                    />
                  </CCol>

                </CRow>

                <CButton type="submit" onClick={() => UpdateActions()}>Update Actions</CButton>
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
                    <CButton type="submit" onClick={() => UpdateStatus()}>Update Status</CButton>
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

              </CCardBody>
            </CCard>
          </CCol>
        </CRow>
      </>
    )
  };

  const Update = () => {
    console.log("Update info");

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
    ProviderPut(target, body).then(() => {
      console.log("Updated info.",);

    });
  };

  const UpdateResource = () => {
    console.log("UpdateResource");

    const tmpData = {
      "outplan_id": ref_outplan_id.current.value,
      "outdial_id": ref_outdial_id.current.value,
      "queue_id": ref_queue_id.current.value,
      "next_campaign_id": ref_next_campaign_id.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "campaigns/" + ref_id.current.value + "/resource_info";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(response => {
      console.log("Updated info. response: " + JSON.stringify(response));
    });
  };

  const UpdateActions = () => {
    console.log("UpdateActions");

    const tmpData = {
      "actions": JSON.parse(ref_actions.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "campaigns/" + ref_id.current.value + "/actions";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(response => {
      console.log("Updated info. response: " + JSON.stringify(response));
    });
  };

  const UpdateStatus = () => {
    console.log("UpdateStatus");

    const tmpData = {
      "status": ref_status.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "campaigns/" + ref_id.current.value + "/status";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(response => {
      console.log("Updated info. response: " + JSON.stringify(response));
    });
  };

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default CampaignsDetail
