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

const ConferencecallsDetail = () => {
  console.log("ConferencesDetail");

  const ref_id = useRef(null);
  const ref_status = useRef(null);
  const ref_conference_id = useRef(null);
  const ref_reference_type = useRef(null);
  const ref_reference_id = useRef(null);

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("conferencecalls");
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

                </CRow>



                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Conference ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_conference_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.conference_id}
                      readOnly plainText
                    />
                  </CCol>

                </CRow>



                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Reference Type</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_reference_type}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.reference_type}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Reference ID</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_reference_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.reference_id}
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


                <CButton type="submit" onClick={() => Kick()}>Kick</CButton>
                <br />
                <br />

          </CCardBody>
        </CCard>
      </CCol>
      </CRow>

      </>
    )
  };

  const Kick = () => {
    console.log("Kick info");

    const body = JSON.stringify('');
    const target = "conferencecalls/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderDelete(target, body).then((response) => {
      console.log("Kick info.", response);
    });
  };

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default ConferencecallsDetail
