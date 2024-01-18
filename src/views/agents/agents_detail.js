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

const AgentsDetail = () => {
  console.log("AgentsDetail");

  const ref_id = useRef(null);
  const ref_username = useRef(null);
  const ref_status = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_permission = useRef(null);
  const ref_ring_method = useRef(null);
  const ref_addresses = useRef(null);
  const ref_tag_ids = useRef(null);

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("agents");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
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
                  <CFormLabel className="col-sm-2 col-form-label"><b>Username</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_username}
                      type="text"
                      defaultValue={detailData.username}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel className="col-sm-2 col-form-label"><b>Status</b></CFormLabel>
                  <CCol>
                    <CFormSelect
                      ref={ref_status}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.status}
                      options={[
                        { label: 'available', value: 'available' },
                        { label: 'away', value: 'away' },
                        { label: 'busy', value: 'busy' },
                        { label: 'offline', value: 'offline' },
                        { label: 'ringing', value: 'ringing' },
                      ]}
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Ring Method</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_ring_method}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.ring_method}
                      options={[
                        { label: 'ringall', value: 'ringall' },
                      ]}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" onClick={() => UpdateBasic()}>Update</CButton>
                <br />
                <br/>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Permission</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_permission}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.permission}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" onClick={() => UpdatePermission()}>Update Permission</CButton>
                <br />
                <br/>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Addresses</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_addresses}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.addresses, null, 2)}
                      rows={15}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" onClick={() => UpdateAddresse()}>Update Addresses</CButton>
                <br />
                <br/>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Tag IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_tag_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.tag_ids, null, 2)}
                      rows={15}
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" onClick={() => UpdateTagIDs()}>Update Tag IDs</CButton>
                <br />
                <br/>

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

                <CButton type="submit" color="dark" onClick={() => Delete()}>Delete</CButton>

              </CCardBody>
            </CCard>
          </CCol>
        </CRow>
      </>
    )
  };

  const UpdateBasic = () => {
    console.log("Update info");

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "ring_method": ref_ring_method.current.value,
      "permission": Number(ref_permission.current.value),
      "addresses": JSON.parse(ref_addresses.current.value),
      "tag_ids": JSON.parse(ref_tag_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(response => {
      console.log("Updated info. response: " + JSON.stringify(response));
    });
  };


  const UpdateAddresse = () => {
    console.log("Update addresses info");

    const tmpData = {
      "addresses": JSON.parse(ref_addresses.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value + "/addresses";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(response => {
      console.log("Updated info. response: " + JSON.stringify(response));
    });
  };

  const UpdateTagIDs = () => {
    console.log("Update tag ids info");

    const tmpData = {
      "tag_ids": JSON.parse(ref_tag_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value + "/tag_ids";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(response => {
      console.log("Updated info. response: " + JSON.stringify(response));
    });
  };

  const UpdatePermission = () => {
    console.log("Update permission info");

    const tmpData = {
      "permission": JSON.parse(ref_permission.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "agents/" + ref_id.current.value + "/permission";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(response => {
      console.log("Updated info. response: " + JSON.stringify(response));
    });
  };

  const Delete = () => {
    console.log("Delete info");

    const body = JSON.stringify("");
    const target = "agents/" + ref_id.current.value;
    console.log("Deleting agent info. target: " + target + ", body: " + body);
    ProviderDelete(target, body).then(response => {
      console.log("Deleted info. response: " + JSON.stringify(response));
    });
  }


  return (
    <>
      <GetDetail/>
    </>
  )
}

export default AgentsDetail
