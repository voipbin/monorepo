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

const Campaigncalls = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getCampaigncalls();
    return;
  }, []);

  const getCampaigncalls = (() => {
    const target = "campaigncalls?page_size=100";

    ProviderGet(target).then(result => {
      const data = result.result;
      setListData(data);
      setIsLoading(false);

      const tmp = ParseData(data);
      store.dispatch({
        type: 'campaigncalls',
        data: tmp,
      });
    });
  });

  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'source.target',
        header: 'source number',
        size: 100,
      },
      {
        accessorKey: 'destination.target',
        header: 'destination number',
        size: 100,
      },
      {
        accessorKey: 'direction',
        header: 'direction',
        size: 100,
      },
      {
        accessorKey: 'status',
        header: 'status',
        size: 100,
      },
      {
        accessorKey: 'hangup_by',
        header: 'hangup by',
        size: 80,
      },
      {
        accessorKey: 'hangup_reason',
        header: 'hangup reason',
        size: 80,
      },
      {
        accessorKey: 'tm_update',
        header: 'last update',
        size: 250,
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
            <CModalTitle>Detail</CModalTitle>
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

export default Campaigncalls
