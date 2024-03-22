import React, { useMemo, useState, useEffect } from 'react'
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

const Calls = () => {
  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  }, []);

  const getList = (() => {
    const target = "calls?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("calls", tmpData);
      })
      .catch(e => {
        console.log("Could not get the list of calls. err: %o", e);
        alert("Could not get the list of calls.");
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

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/calls/calls_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }
  const Create = (row) => {
    const target = "/resources/calls/calls_create";
    console.log("navigate target: ", target);
    navigate(target);
  }

  return (
    <>
      <MaterialReactTable
        columns={listColumns}
        data={listData}
        enableRowNumbers
        state={{
          isLoading: isLoading,
        }}
        enableRowActions
        renderRowActions={({ row, table }) => (
          <Box sx={{ display: 'flex' }}>
            <Tooltip arrow placement="left" title="Edit">
              <IconButton onClick={() => Detail(row)}>
                <Edit />
              </IconButton>
            </Tooltip>
          </Box>
        )}
        muiTableBodyRowProps={({ row }) => ({
          onDoubleClick: (event) => {
            Detail(row);
          },
        })}
        renderTopToolbarCustomActions={() => (
          <Button
            color="secondary"
            onClick={() => {
              Create();
            }}
            variant="contained"
          >
            Create
          </Button>
        )}
      />
    </>
  )
}

export default Calls
