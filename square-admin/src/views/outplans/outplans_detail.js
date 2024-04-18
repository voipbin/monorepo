import React, { useMemo, useRef, useState, useEffect } from 'react'
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
import { useNavigate } from "react-router-dom";

const OutplansDetail = () => {
  console.log("OutplansDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_id = useRef(null);
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

  const id = routeParams.id;

  const GetDetail = () => {

    const tmp = localStorage.getItem("outplans");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    console.log("detailData", detailData);

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs" target="_blank">here</a>.</small>
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Source</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_source}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.source, null, 2)}
                      rows={7}
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
                      defaultValue={detailData.dial_timeout}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Try Interval(ms)</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_try_interval}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.try_interval}
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
                      defaultValue={detailData.max_try_count_0}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Max Try Count 1</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_max_try_count_1}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.max_try_count_1}
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
                      defaultValue={detailData.max_try_count_2}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Max Try Count 3</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_max_try_count_3}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.max_try_count_3}
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
                      defaultValue={detailData.max_try_count_4}
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


                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateBasicInfo()}>Update</CButton>
                &nbsp;
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
      "source": JSON.parse(ref_source.current.value),
      "dial_timeout": ref_dial_timeout.current.value,
      "try_interval": ref_try_interval.current.value,
      "max_try_count_0": Number(ref_max_try_count_0.current.value),
      "max_try_count_1": Number(ref_max_try_count_1.current.value),
      "max_try_count_2": Number(ref_max_try_count_2.current.value),
      "max_try_count_3": Number(ref_max_try_count_3.current.value),
      "max_try_count_4": Number(ref_max_try_count_4.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "outplans/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then((response) => {
        console.log("Updated info.", JSON.stringify(response));
        const navi = "/resources/outplans/outplans_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not update the outplan info. err: %o", e);
        alert("Could not not update the outplan info.");
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
    const target = "outplans/" + ref_id.current.value;
    console.log("Deleting outplan info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        const navi = "/resources/outplans/outplans_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not delete the outplan info. err: %o", e);
        alert("Could not not delete the ouplan info.");
        setButtonDisable(false);
      });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default OutplansDetail
