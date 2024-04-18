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

const TagsCreate = () => {
  console.log("TagsCreate");

  const [buttonDisable, setButtonDisable] = useState(false);
  const navigate = useNavigate();

  const ref_name = useRef("");
  const ref_detail = useRef("");

  const Create = () => {

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details about tag at <a href="https://api.voipbin.net/docs/tag.html" target="_blank">here</a>.</small>
              </CCardHeader>

              <CCardBody>

                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Name</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_name}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue=""
                    />
                  </CCol>
                </CRow>


                <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Detail</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_detail}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue=""
                    />
                  </CCol>
                </CRow>

                <CButton type="submit" disabled={buttonDisable} onClick={() => CreateResource()}>Create</CButton>
              </CCardBody>
            </CCard>
          </CCol>
        </CRow>

      </>
    )
  };

  const CreateResource = () => {
    console.log("Create info");
    setButtonDisable(true);

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "tags";
    console.log("Creating tag info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created tag info.", JSON.stringify(response));
        const navi = "/resources/tags/tags_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not create a new tag. err: %o", e);
        alert("Could not not create a new tag.");
        setButtonDisable(false);
      });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default TagsCreate
