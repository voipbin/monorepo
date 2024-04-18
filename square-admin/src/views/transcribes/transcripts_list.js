import React, { useMemo, useState, useEffect, useRef } from 'react'
import { useParams } from "react-router-dom";
import {
  CCard,
  CCardBody,
  CCardHeader,
  CCol,
  CFormInput,
  CFormLabel,
  CFormTextarea,
  CRow,
  CButton,
} from '@coreui/react'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Stack,
  TextField,
  Tooltip,
} from '@mui/material';
import { Delete, Edit } from '@mui/icons-material';
import store from '../../store'
import { MaterialReactTable } from 'material-react-table';
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';

const Transcripts = () => {
  const routeParams = useParams();
  const ref_message = useRef("");

  const [transcripts, setTranscripts] = useState([]);

  useEffect(() => {
    getList();
    return;
  }, []);

  const transcribe_id = routeParams.id;

  const getList = (() => {
    const target = "transcripts?page_size=1000&transcribe_id=" + transcribe_id;

    ProviderGet(target)
      .then(result => {
        console.log("Received transcripts. result: %o", result);

        const tmps = result.result;

        var tmpRes = [];
        tmps?.forEach(tmp => {
          console.log("transcript detail. transcript: %o", tmp);
          if (tmp["direction"] == "in") {
            tmpRes.push(
              <CFormInput
                type="text"
                floatingClassName="mb-3"
                floatingLabel="in"
                color="success"
                defaultValue={tmp["message"]}
                aria-label="Disabled input example"
                readOnly
                valid
              />
            );
          } else {
            tmpRes.push(
              <CFormInput
                type="text"
                floatingClassName="mb-3"
                floatingLabel={"out"}
                color="primary"
                defaultValue={tmp["message"]}
                aria-label="Disabled input example"
                readOnly
              />
            );
          }
        })

        console.log("res: %o", tmpRes);
        setTranscripts(tmpRes);
      })
      .catch(e => {
        console.log("Could not get list of transcript messages. err: %o", e);
        alert("Could not get the list of transcript messages.");
      });
  });

  return (
    <CRow>
      <CCol xs={12}>
        <CCard className="mb-4">
          <CCardHeader>
            <strong>Transcript detail</strong> <small>Detail of the transcript</small>
          </CCardHeader>

          <CCardBody>
            {transcripts}
          </CCardBody>
        </CCard>
      </CCol>
    </CRow>
  )
}

export default Transcripts
