import React, { useRef, useState } from 'react'
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

const ProvidersDetail = () => {
  console.log("ProvidersDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_id = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_type = useRef(null);
  const ref_hostname = useRef(null);
  const ref_tech_prefix = useRef(null);
  const ref_tech_postfix = useRef(null);
  const ref_tech_headers = useRef(null);

  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("providers");
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Type</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_type}
                      type="text"
                      id="colFormLabelSm"
                      options={[
                        { label: 'sip', value: 'sip' },
                      ]}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Hostname</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_hostname}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.hostname}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Tech Prefix</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_tech_prefix}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.tech_prefix}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Tech Postfix</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_tech_postfix}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.tech_postfix}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Tech Headers</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_tech_headers}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.tech_headers, null, 2)}
                      rows={5}
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
      "type": ref_type.current.value,
      "hostname": ref_hostname.current.value,
      "tech_prefix": ref_tech_prefix.current.value,
      "tech_postfix": ref_tech_postfix.current.value,
      "tech_headers": JSON.parse(ref_tech_headers.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "providers/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then((response) => {
        console.log("Updated info.", JSON.stringify(response));
        const navi = "/resources/providers/providers_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not update the provider infno. err: %o", e);
        alert("Could not not update the provider info.");
        setButtonDisable(false);
      });
  }

  const Delete = () => {
    console.log("Delete info");

    if (!confirm(`Are you sure you want to delete?`)) {
      return;
    }
    setButtonDisable(true);

    const body = JSON.stringify("");
    const target = "providers/" + ref_id.current.value;
    console.log("Deleting provider info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted info. response: " + JSON.stringify(response));
        const navi = "/resources/providers/providers_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not delete the provider info. err: %o", e);
        alert("Could not not delete the provider info.");
        setButtonDisable(false);
      });
  }

  return (
    <>
      <GetDetail/>
    </>
  )
}

export default ProvidersDetail
