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

const ConferencesDetail = () => {
  console.log("ConferencesDetail");

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

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("conferences");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    console.log("detailData", detailData);

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
                    <CFormTextarea
                      ref={ref_pre_actions}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.pre_actions, null, 2)}
                      rows={15}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Post Actions</b></CFormLabel>
                  <CCol>
                  <CFormTextarea
                      ref={ref_post_actions}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.post_actions, null, 2)}
                      rows={15}
                    />
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


                <CButton type="submit" onClick={() => UpdateBasicInfo()}>Update</CButton>
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

          </CCardBody>
        </CCard>
      </CCol>
      </CRow>

      </>
    )
  };

  const UpdateBasicInfo = () => {
    console.log("Update info");

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
    ProviderPut(target, body).then((response) => {
      console.log("Updated info.", response);
    });
  };

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default ConferencesDetail
