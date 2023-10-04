import React, { useMemo, useState, useEffect } from 'react'
import {
  CButton,
  CModal,
  CModalBody,
  CModalFooter,
  CModalHeader,
  CModalTitle,
} from '@coreui/react'
import store from '../../store'
import { MaterialReactTable } from 'material-react-table';
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";

const Messages = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getMessages();
    return;
  }, []);

  const getMessages = (() => {
    const target = "messages?page_size=100";

    ProviderGet(target).then(result => {
      const data = result.result;
      setListData(data);
      setIsLoading(false);

      const tmp = ParseData(data);
      store.dispatch({
        type: 'messages',
        data: tmp,
      });
    });
  });

  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'type',
        header: 'type',
        size: 80,
      },
      {
        accessorKey: 'source.target',
        header: 'source number',
        size: 250,
      },
      {
        accessorKey: 'targets.length',
        header: 'targets',
        size: 150,
      },
      {
        accessorKey: 'direction',
        header: 'direction',
        size: 150,
      },
      {
        accessorKey: 'tm_update',
        header: 'last update',
        size: 250,
      }
    ],
    [],
  );


  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/activity/messages/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }

  const [detailData, setDetailData] = useState({});
  const [modalState, setModalState] = useState(false);
  const ModalDetail = () => {
    const tmp = JSON.stringify(detailData, null, 2)
    return (
      <>
        <CModal scrollable visible={modalState} size="xl" onClose={() => setModalState(false)}>
          <CModalHeader>
            <CModalTitle>Call detail</CModalTitle>
          </CModalHeader>
          <CModalBody>

            <div><pre>{tmp}</pre></div>

          </CModalBody>
          <CModalFooter>
            <CButton color="primary" onClick={() => setModalState(false)}>
              Close
            </CButton>
          </CModalFooter>
        </CModal>
      </>
    )
  }

  return (
    <>
      <ModalDetail/>
      <MaterialReactTable
        columns={listColumns}
        data={listData}
        enableRowNumbers
        state={{
          isLoading: isLoading,
        }}
        muiTableBodyRowProps={({ row }) => ({
          onDoubleClick: (event) => {
            Detail(row);
          },
        })}
      />
    </>
  )
}

export default Messages
