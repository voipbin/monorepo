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

const ExtensionsDetail = () => {
  console.log("ExtensionsDetail");

  const ref_id = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_email = useRef(null);
  const ref_phone_number = useRef(null);
  const ref_address = useRef(null);
  const ref_billingaccount_id = useRef(null);
  const ref_webhook_uri = useRef(null);
  const ref_webhook_method = useRef(null);
  const ref_permission_ids = useRef(null);

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("extensions");
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

                  <CFormLabel className="col-sm-2 col-form-label"><b>Username</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.username}
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Email</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_email}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.email}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Phone Number</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_phone_number}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.phone_number}
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Address</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_address}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.address}
                    />
                  </CCol>
                </CRow>







                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Webhook URI</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_webhook_uri}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.webhook_uri}
                    />
                  </CCol>


                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Webhook Method</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_webhook_method}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.webhook_method}
                      options={[
                        detailData.webhook_method,
                        { label: 'GET', value: 'GET' },
                        { label: 'POST', value: 'POST' },
                        { label: 'PUT', value: 'PUT' },
                        { label: 'DELETE', value: 'DELETE' },
                      ]}
                    />
                  </CCol>
                </CRow>



                <CButton type="submit" onClick={() => UpdateBasic()}>Update</CButton>
                <br/>
                <br/>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Billing Account ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_billingaccount_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.billing_account_id}
                    />
                  </CCol>
                  <CCol className="mb-3 align-items-auto">
                    <CButton type="submit" onClick={() => UpdateBillingAccountID()}>Update</CButton>
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Permission IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_permission_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={JSON.stringify(detailData.permission_ids, null, 2)}
                      rows={5}
                    />
                  </CCol>
                  <CCol className="mb-3 align-items-auto">
                    <CButton type="submit" onClick={() => UpdatePermissionIDs()}>Update</CButton>
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
                  <CCol className="mb-3 align-items-auto">
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

      <CButton type="submit" onClick={() => UpdateBasic()}>Update</CButton>
      </>
    )
  };

  const UpdateBasic = () => {
    console.log("Update UpdateBasic");

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "email": ref_email.current.value,
      "phone_number": ref_phone_number.current.value,
      "address": ref_address.current.value,
      "webhook_method": ref_webhook_method.current.value,
      "webhook_uri": ref_webhook_uri.current.value,
      "permission_ids": JSON.parse(ref_permission_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "extensions/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(() => {
      console.log("Updated info.");
    });
  };

  const UpdateBillingAccountID = () => {
    console.log("Update UpdateBillingAccountID");

    const tmpData = {
      "billing_account_id": ref_billingaccount_id.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "customers/" + ref_id.current.value +"/billing_account_id";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(() => {
      console.log("Updated info.");
    });
  };

  const UpdatePermissionIDs = () => {
    console.log("Update UpdatePermissionIDs");

    const tmpData = {
      "permission_ids": JSON.parse(ref_permission_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "customers/" + ref_id.current.value +"/permission_ids";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body).then(() => {
      console.log("Updated info.");
    });
  };



  return (
    <>
      <GetDetail/>
    </>
  )
}

export default ExtensionsDetail
