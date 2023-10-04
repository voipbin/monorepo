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

const Detail = () => {
  console.log("Detail");

  const ref_id = useRef(null);
  const ref_type = useRef(null);
  const ref_from = useRef(null);
  const ref_direction = useRef(null);
  const ref_targets = useRef(null);
  const ref_text = useRef(null);

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const storeData = store.getState();
    const detailData = storeData["messages"][id];
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
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Type</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_type}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.type}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Direction</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_direction}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.direction}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>From</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_from}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.source.target}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>




                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Targets</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_targets}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.targets, null, 2)}
                      rows={15}
                      readOnly plainText
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
                      defaultValue={detailData.text}
                      rows={2}
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

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default Detail
