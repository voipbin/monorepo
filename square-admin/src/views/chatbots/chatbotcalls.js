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

const Chabotcalls = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getCalls();
    return;
  }, []);

  const getCalls = (() => {
    const target = "chatbotcalls?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        store.dispatch({
          type: 'chatbotcalls',
          data: tmp,
        });
      })
      .catch(e => {
        console.log("Could not get the list of chatbotcalls. err: %o", e);
        alert("Could not get the list of chatbotcalls.");
        setButtonDisable(false);
      });
  });

  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'id',
        header: 'ID',
        size: 200,
      },
      {
        accessorKey: 'status',
        header: 'Status',
        size: 50,
      },
      {
        accessorKey: 'reference_type',
        header: 'Reference Type',
        size: 50,
      },
      {
        accessorKey: 'reference_id',
        header: 'Reference ID',
        size: 200,
      },
      {
        accessorKey: 'chatbot_id',
        header: 'Chatbot ID',
        size: 200,
      },
      {
        accessorKey: 'tm_update',
        header: 'last update',
        size: 200,
      }
    ],
    [],
  );

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
            console.info("print modal", modalState, event, row);
            setDetailData(row.original);
            setModalState(!modalState);
          },
        })}
      />
    </>
  )
}

export default Chabotcalls
