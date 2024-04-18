import React, { useRef, useState } from 'react'
import { useNavigate } from "react-router-dom";
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

const RoomsDetail = () => {
  console.log("RoomsDetail");

  const [buttonDisable, setButtonDisable] = useState(false);
  const routeParams = useParams();
  const navigate = useNavigate();

  const ref_id = useRef(null);
  const ref_name = useRef(null);
  const ref_detail = useRef(null);
  const ref_owner_id = useRef(null);
  const ref_type = useRef(null);
  const ref_participant_ids = useRef(null);

  const GetDetail = () => {
    const id = routeParams.id;

    const tmp = localStorage.getItem("chatrooms");
    const datas = JSON.parse(tmp);
    const detailData = datas[id];
    return (
      <>
        <CRow>
          <CCol xs={12}>
            <CCard className="mb-4">
              <CCardHeader>
                <strong>Detail</strong> <small>You can find more details at <a href="https://api.voipbin.net/docs/chat.html" target="_blank">here</a>.</small>
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
                  <CFormLabel className="col-sm-2 col-form-label"><b>Chat ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      type="text"
                      id="id"
                      defaultValue={detailData.chat_id}
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
                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Owner ID</b></CFormLabel>
                  <CCol className="mb-3 align-items-auto">
                    <CFormInput
                      ref={ref_owner_id}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.owner_id}
                      readOnly plainText
                    />
                  </CCol>

                  <CFormLabel htmlFor="colFormLabelSm" className="col-sm-2 col-form-label"><b>Type</b></CFormLabel>
                  <CCol>
                    <CFormInput
                      ref={ref_type}
                      type="text"
                      id="colFormLabelSm"
                      defaultValue={detailData.type}
                      readOnly plainText
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
                      defaultValue={JSON.stringify(detailData.participant_ids, null, 2)}
                      rows={10}
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


                <CRow>
                  <CButton type="submit" onClick={() => Messages()}>Messages</CButton>
                </CRow>

                <br/>
                <CButton type="submit" onClick={() => Update()}>Update</CButton>
                &nbsp;
                <CButton type="submit" color="dark" onClick={() => Delete()}>Delete</CButton>

              </CCardBody>
            </CCard>
          </CCol>
        </CRow>

      </>
    )
  };

  const Messages = () => {
    console.log("Messages info");

    const target = "/resources/chats/" + ref_id.current.value + "/messages_list";
    console.log("navigate target: ", target);
    navigate(target);
  };

  const Update = () => {
    console.log("Update info");
    setButtonDisable(true);

    const tmpData = {
      "name": ref_name.current.value,
      "detail": ref_detail.current.value,
    };

    const body = JSON.stringify(tmpData);
    const target = "chatrooms/" + ref_id.current.value;
    console.log("Updating conversation info. target: " + target + ", body: " + body);
    ProviderPut(target, body)
      .then(response => {
        console.log("Updated info. response: " + JSON.stringify(response));
        const navi = "/resources/chats/rooms_list";
        navigate(navi);
      })
      .catch(e => {
        console.log("Could not update the info. err: %o", e);
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
    const target = "chatrooms/" + ref_id.current.value;
    console.log("Deleting chat info. target: " + target + ", body: " + body);
    ProviderDelete(target, body)
    .then(response => {
      console.log("Deleted info. response: " + JSON.stringify(response));
      const navi = "/resources/chats/rooms_list";
      navigate(navi);
    })
    .catch(e => {
      console.log("Could not delete the info. err: %o", e);
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

export default RoomsDetail
