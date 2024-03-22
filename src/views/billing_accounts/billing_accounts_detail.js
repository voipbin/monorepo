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

const BillingAccountsDetail = () => {
  console.log("BillingAccountsDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const navigate = useNavigate();

  const ref_id = useRef(null);
  const ref_balance = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_payment_method = useRef(null);
  const ref_payment_type = useRef(null);

  const routeParams = useParams();
  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("billing_accounts");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/billing_account.html" target="_blank">here</a>.</small>
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

                  <CFormLabel className="col-sm-2 col-form-label"><b>Balance(USD)</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_balance}
                      type="text"
                      id="id"
                      defaultValue={detailData.balance}
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


                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdateBasicInfo()}>Update</CButton>
                <br />
                <br />


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Payment Method</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_payment_method}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.payment_method}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Payment Type</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_payment_type}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.payment_type}
                    />
                  </CCol>
                </CRow>


                <CButton type="submit" disabled={buttonDisable} onClick={() => UpdatePaymentInfo()}>Update Payment</CButton>
                <br />
                <br />


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

  const navigateBack = () => {
    navigate(-1);
  }

  const UpdateBasicInfo = () => {
    console.log("Update info");
    setButtonDisable(true);

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "payment_method": ref_payment_method.current.value,
      "payment_type": Number(ref_payment_type.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "billing_accounts/" + ref_id.current.value;
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then((response) => {
        console.log("Updated info.", JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the basic info. err: %o", e);
        alert("Could not update the info.");
        setButtonDisable(false);
      });
  };

  const UpdatePaymentInfo = () => {
    console.log("Update info");
    setButtonDisable(true);

    const tmpData = {
      "payment_method": ref_payment_method.current.value,
      "payment_type": Number(ref_payment_type.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "billing_accounts/" + ref_id.current.value + "/payment_info";
    console.log("Update info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then((response) => {
        console.log("Updated info.", JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not update the payment info. err: %o", e);
        alert("Could not update the info.");
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
    const target = "billing_accounts/" + ref_id.current.value;
    console.log("Deleting billing account info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
      .then(response => {
        console.log("Deleted billing account. response: " + JSON.stringify(response));
        navigateBack();
      })
      .catch(e => {
        console.log("Could not delete the billing account. err: %o", e);
        alert("Could not delete the info.");
        setButtonDisable(false);
      });
  }


  return (
    <>
      <GetDetail/>
    </>
  )
}

export default BillingAccountsDetail
