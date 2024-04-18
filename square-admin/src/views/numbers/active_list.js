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
import store from '../../store'
import {
  MaterialReactTable,
} from 'material-react-table';
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";

const ActiveList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  }, []);

  const getList = (() => {
    const target = "numbers?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("numbers", tmpData);
      })
      .catch(e => {
        console.log("Could not get a list of numbers. err: %o", e);
        alert("Could not not get a list of numbers.");
      });
  });

  // show list
  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'id',
        header: 'ID',
        enableEditing: false,
      },
      {
        accessorKey: 'name',
        header: 'Name',
      },
      {
        accessorKey: 'detail',
        header: 'Detail',
      },
      {
        accessorKey: 'number',
        header: 'Number',
        enableEditing: false,
        size: 100,
      },
      {
        accessorKey: 'call_flow_id',
        header: 'Call Flow ID',
        size: 100,
      },
      {
        accessorKey: 'message_flow_id',
        header: 'Message Flow ID',
        size: 100,
      },
      {
        accessorKey: 'status',
        header: 'Status',
        enableEditing: false,
        size: 100,
      },
      {
        accessorKey: 'emergency_enabled',
        header: 'Emergency Enabled',
        enableEditing: false,
        size: 100,
      },
      {
        accessorKey: 't38_enabled',
        header: 'T38 Enabled',
        enableEditing: false,
        size: 100,
      },
      {
        accessorKey: 'tm_create',
        header: 'Create Time',
        enableEditing: false,
        size: 250,
      },
      {
        accessorKey: 'tm_update',
        header: 'Update Time',
        enableEditing: false,
        size: 250,
      },
      {
        accessorKey: 'tm_delete',
        header: 'Delete Time',
        enableEditing: false,
        size: 250,
      },
      {
        accessorKey: 'tm_purchase',
        header: 'Purchase Time',
        enableEditing: false,
        size: 250,
      },
      {
        accessorKey: 'tm_renew',
        header: 'Renew Time',
        enableEditing: false,
        size: 250,
      }
    ],
    [],
  );

  const columnVisibility = {
    id: false,
    emergency_enabled: false,
    t38_enabled: false,
    call_flow_id: false,
    message_flow_id: false,
    tm_create: false,
    tm_delete: false,
    tm_purchase: false,
    tm_renew: false,
  };

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/numbers/active_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }

  return (
    <>
      <MaterialReactTable
        columns={listColumns}
        data={listData ?? []} // data?.data ?? []
        state={{
          isLoading: isLoading,
        }}
        enableRowNumbers
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

export default ActiveList
