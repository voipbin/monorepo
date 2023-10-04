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
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    getList();
    return;
  }, []);

  const getList = (() => {
    const tmp = JSON.parse(localStorage.getItem("groupcalls"));
    const data = Object.values(tmp);
    setListData(data);
  });

  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'source.target',
        header: 'source',
        size: 100,
      },
      {
        accessorKey: 'destinations.length',
        header: 'destinations',
        size: 50,
      },
      {
        accessorKey: 'flow_id',
        header: 'flow id',
        size: 250,
      },
      {
        accessorKey: 'ring_method',
        header: 'Ring Method',
        size: 100,
      },
      {
        accessorKey: 'call_ids.length',
        header: 'calls',
        size: 100,
      },
      {
        accessorKey: 'groupcall_ids.length',
        header: 'groupcalls',
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

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/calls/groupcalls_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }
  const Create = (row) => {
    const target = "/resources/calls/groupcalls_create";
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
