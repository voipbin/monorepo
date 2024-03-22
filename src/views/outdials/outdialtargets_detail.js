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

const OutdialsDetail = () => {
  console.log("OutdialsDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_id = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_outdial_id = useRef(null);
  const ref_status = useRef(null);
  const ref_destination_0 = useRef(null);
  const ref_destination_1 = useRef(null);
  const ref_destination_2 = useRef(null);
  const ref_destination_3 = useRef(null);
  const ref_destination_4 = useRef(null);
  const ref_try_count_0 = useRef(null);
  const ref_try_count_1 = useRef(null);
  const ref_try_count_2 = useRef(null);
  const ref_try_count_3 = useRef(null);
  const ref_try_count_4 = useRef(null);
  const ref_data = useRef(null);

  const GetDetail = () => {
    const outdial_id = routeParams.outdial_id;
    const id = routeParams.id;
    const tmp = localStorage.getItem("outdials/" + outdial_id + "/targets");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    console.log("detailData", detailData);

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
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Detail</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_detail}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.detail}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel className="col-sm-2 col-form-label"><b>Outdial ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_outdial_id}
                      type="text"
                      id="id"
                      defaultValue={detailData.outdial_id}
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
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destination 0</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_destination_0}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.destination_0, null, 2)}
                      rows={7}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Try Count 0</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_try_count_0}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.try_count_0}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destination 1</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_destination_1}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.destination_1, null, 2)}
                      rows={7}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Try Count 1</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_try_count_1}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.try_count_1}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destination 2</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_destination_2}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.destination_2, null, 2)}
                      rows={7}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Try Count 2</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_try_count_2}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.try_count_2}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destination 3</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_destination_3}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.destination_3, null, 2)}
                      rows={7}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Try Count 3</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_try_count_3}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.try_count_3}
                      readOnly plainText
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Destination 4</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_destination_4}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.destination_4, null, 2)}
                      rows={7}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Try Count 4</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_try_count_4}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.try_count_4}
                      readOnly plainText
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
                      defaultValue={detailData.data}
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
      "outdial_id": ref_outdial_id.current.value,
      "destination_0": JSON.parse(ref_destination_0.current.value),
      "retry_count_0": ref_try_count_0.current.value,
      "destination_1": JSON.parse(ref_destination_1.current.value),
      "retry_count_1": ref_try_count_1.current.value,
      "destination_2": JSON.parse(ref_destination_2.current.value),
      "retry_count_2": ref_try_count_2.current.value,
      "destination_3": JSON.parse(ref_destination_3.current.value),
      "retry_count_3": ref_try_count_3.current.value,
      "destination_4": JSON.parse(ref_destination_4.current.value),
      "retry_count_4": ref_try_count_4.current.value,
      "data": ref_status.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "outdials/" + ref_outdial_id.current.value + "/targets/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then((response) => {
        console.log("Updated info.", JSON.stringify(response));
        const navi = "/resources/outdials/" + ref_outdial_id.current.value + "/outdialtargets_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not update the outdial target info. err: %o", e);
        alert("Could not not update the outdial target info.");
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
    const target = "outdials/" + outdial_id + "/targets/" + ref_id.current.value;
    console.log("Deleting outdial target info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        const navi = "/resources/outdials/" + ref_outdial_id.current.value + "/outdialtargets_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not update the outdial target info. err: %o", e);
        alert("Could not not update the outdial target info.");
        setButtonDisable(false);
      });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default OutdialsDetail
