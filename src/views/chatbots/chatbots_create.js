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

const ChatbotsCreate = () => {
  console.log("ChatbotsCreate");

  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_engine_type = useRef(null);
  const ref_init_prompt = useRef(null);

  const routeParams = useParams();
  const Create = () => {
    const id = routeParams.id;

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
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Engine Type</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_engine_type}
                      type="text"
                      id="colFormLabelSm"
                      options={[
                        { label: 'chatGPT', value: 'chatGPT' },
                      ]}
                    />
                  </CCol>
                </CRow>



                <CRow>
                <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Init Prompt</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_init_prompt}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="Please enter your initial prompt for initialization prompt."
                      rows={15}
                    />
                  </CCol>

                </CRow>



          </CCardBody>
        </CCard>
      </CCol>
      </CRow>

      <CButton type="submit" onClick={() => CreateResource()}>Create</CButton>
      </>
    )
  };

  const CreateResource = () => {
    console.log("Create info");

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
      "engine_type": ref_engine_type.current.value,
      "init_prompt": ref_init_prompt.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "chatbots";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body).then((response) => {
      console.log("Created info.", response);
    });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default ChatbotsCreate
