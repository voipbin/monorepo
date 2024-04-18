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

const ChatsCreate = () => {
  console.log("ChatsCreate");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_type = useRef(null);
  const ref_owner_id = useRef(null);
  const ref_participant_ids = useRef(null);

  const Create = () => {
    const id = routeParams.id;

    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Create</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/chat.html" target="_blank">here</a>.</small>
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Type</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormSelect
                      ref={ref_type}
                      type="text"
                      id="colFormLabelSm"
                      options={[
                        { label: 'normal', value: 'normal' },
                        { label: 'group', value: 'group' },
                      ]}
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Owner ID</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_owner_id}
                      type="text"
                      id="colFormLabelSm"
                    />
                  </CCol>
                </CRow>


                <CRow>
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Participant IDs</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormTextarea
                      ref={ref_participant_ids}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue="[]"
                      rows={15}
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
      "type": ref_type.current.value,
      "owner_id": ref_owner_id.current.value,
      "participant_ids": JSON.parse(ref_participant_ids.current.value),
    };

    const body = JSON.stringify(tmpData);
    const target = "chats";
    console.log("Create info. target: " + target + ", body: " + body);
    ProviderPost(target, body)
      .then((response) => {
        console.log("Created info.", JSON.stringify(response));
        const navi = "/resources/chats/chats_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not create a new chat. err: %o", e);
        alert("Could not create a new chat.");
        setButtonDisable(false);
      });
  };

  return (
    <>
      <Create/>
    </>
  )
}

export default ChatsCreate
