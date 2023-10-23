import React, { useMemo, useState, useEffect } from 'react'
import {
  CButton,
  CModal,
  CModalBody,
  CModalFooter,
  CModalHeader,
  CModalTitle,
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
import { MaterialReactTable } from 'material-react-table';
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";

const CampaigncallsList = () => {

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
      const tmpData = JSON.stringify(tmp);
      localStorage.setItem("campaigncalls", tmpData);
    });
  });

  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'id',
        header: 'ID',
        enableEditing: false,
      },
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
        accessorKey: 'status',
        header: 'status',
        size: 100,
      },
      {
        accessorKey: 'result',
        header: 'result',
        size: 100,
      },
      {
        accessorKey: 'tm_update',
        header: 'last update',
        size: 250,
      }
    ],
    [],
  );

  const columnVisibility = {
    id: false,
  };

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

  const handleDeleteRow = (row) => {
    console.log("Deleting row. ", row)

    if (
      !confirm(`Are you sure you want to delete ${row.getValue('name')}`)
    ) {
      return;
    }

    const target = "campaigncalls/" + row.getValue('id');
    ProviderDelete(target).then((response) => {
      console.log("Deleted info.", JSON.stringify(response));
    });
  }

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/campaigns/campaigncalls_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }

  return (
    <>
      <MaterialReactTable
        columns={listColumns}
        data={listData ?? []} // data?.data ?? []

        enableRowNumbers
        enableRowActions
        renderRowActions={({ row, table }) => (
          <Box sx={{ display: 'flex' }}>
            <Tooltip arrow placement="left" title="Edit">
              <IconButton onClick={() => {
                  Detail(row);
                }
              }>
                <Edit />
              </IconButton>
            </Tooltip>
            <Tooltip arrow placement="right" title="Delete">
              <IconButton color="error" onClick={() => handleDeleteRow(row)}>
                <Delete />
              </IconButton>
            </Tooltip>
          </Box>
        )}

        state={{
          isLoading: isLoading,
        }}

        muiTableBodyRowProps={({ row, table }) => ({
          onDoubleClick: (event) => {
            Detail(row);
          },
        })}
        initialState={{
          columnVisibility: columnVisibility
        }}

        displayColumnDefOptions={{
          'mrt-row-numbers': {
            enableResizing: true,
            enableHiding: true
          }
        }}
      />
    </>
  )
}

export default CampaigncallsList
