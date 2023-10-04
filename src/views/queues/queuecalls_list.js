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

const QueuecallsList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  },[]);

  const getList = (() => {
    const tmp = JSON.parse(localStorage.getItem("queuecalls"));
    const data = Object.values(tmp);
    setListData(data);
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
        accessorKey: 'duration_service',
        header: 'Duration Service(ms)',
        size: 150,
      },
      {
        accessorKey: 'duration_waiting',
        header: 'Duration Waiting(ms)',
        size: 150,
      },
      {
        accessorKey: 'reference_type',
        header: 'Reference Type',
        size: 100,
      },
      {
        accessorKey: 'reference_id',
        header: 'Reference ID',
        size: 100,
      },
      {
        accessorKey: 'service_agent_id',
        header: 'Service Agent ID',
        size: 250,
      },
      {
        accessorKey: 'status',
        header: 'Status',
        size: 250,
      },
      {
        accessorKey: 'tm_create',
        header: 'create time',
        enableEditing: false,
        size: 250,
      },
      {
        accessorKey: 'tm_update',
        header: 'update time',
        enableEditing: false,
        size: 250,
      },
      {
        accessorKey: 'tm_delete',
        header: 'delete time',
        enableEditing: false,
        size: 250,
      }
    ],
    [],
  );

  const columnVisibility = {
    id: false,
    reference_type: false,
    reference_id: false,
    tm_create: false,
    tm_delete: false,
  };


  const handleDeleteRow = (row) => {
    console.log("Deleting row. ", row)

    if (
      !confirm(`Are you sure you want to delete ${row.getValue('name')}`)
    ) {
      return;
    }

    // const target = "queuecalls/" + row.getValue('id');
    // ProviderDelete(target).then(() => {
    //   console.log("Deleted queue.");
    // });
  }

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/queues/queuecalls_detail/" + row.original.id;
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
            {/* <Tooltip arrow placement="right" title="Delete">
              <IconButton color="error" onClick={() => handleDeleteRow(row)}>
                <Delete />
              </IconButton>
            </Tooltip> */}
          </Box>
        )}

        state={{
          // isLoading: isLoading,
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
        renderTopToolbarCustomActions={() => (
          <Button
            color="secondary"
            onClick={() =>
              Create()
            }
            variant="contained"
          >
            Create
          </Button>
        )}
      />
    </>
  )
}

export default QueuecallsList
