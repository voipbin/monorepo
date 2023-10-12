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
  CListGroup,
  CListGroupItem,
  CFormCheck,
  } from '@coreui/react'
import store from '../../store'
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';

const OutplansCreate = () => {
  console.log("OutplansCreate");

  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_source = useRef(null);
  const ref_dial_timeout = useRef(null);
  const ref_try_interval = useRef(null);
  const ref_max_try_count_0 = useRef(null);
  const ref_max_try_count_1 = useRef(null);
  const ref_max_try_count_2 = useRef(null);
  const ref_max_try_count_3 = useRef(null);
  const ref_max_try_count_4 = useRef(null);

  const Create = () => {

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Create</strong> <small>Creating resource</small>
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Source</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_source}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(JSON.parse('{"type":"tel", "target":""}'), null, 2)}
                      rows={5}
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Dial Timeout(ms)</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_dial_timeout}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="30000"
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Try Interval(ms)</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_try_interval}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="300000"
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Max Try Count 0</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_max_try_count_0}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="5"
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Max Try Count 1</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_max_try_count_1}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="5"
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Max Try Count 2</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_max_try_count_2}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="5"
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Max Try Count 3</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_max_try_count_3}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="5"
                    />
                  </CCol>
                </CRow>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Max Try Count 4</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_max_try_count_4}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="5"
                    />
                  </CCol>

                </CRow>

                <CButton type="submit" onClick={() => CreateResource()}>Create</CButton>

              </CCardBody>
            </CCard>
          </CCol>
        </CRow>
      </>
    )
  };

  const CreateResource = () => {
    console.log("Create info");

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "source": JSON.parse(ref_source.current.value),
      "dial_timeout": Number(ref_dial_timeout.current.value),
      "try_interval": Number(ref_try_interval.current.value),
      "max_try_count_0": Number(ref_max_try_count_0.current.value),
      "max_try_count_1": Number(ref_max_try_count_1.current.value),
      "max_try_count_2": Number(ref_max_try_count_2.current.value),
      "max_try_count_3": Number(ref_max_try_count_3.current.value),
      "max_try_count_4": Number(ref_max_try_count_4.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "outplans";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body).then((response) => {
      console.log("Created info.", JSON.stringify(response));
    });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default OutplansCreate
