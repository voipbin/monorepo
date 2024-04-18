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

const QueuesList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  },[]);

  const getList = (() => {
    const target = "queues?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("queues", tmpData);
      })
      .catch(e => {
        console.log("Could not get a resource list. err: %o", e);
        alert("Could not not get a resource list.");
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
        size: 100,
      },
      {
        accessorKey: 'detail',
        header: 'Detail',
        size: 100,
      },
      {
        accessorKey: 'routing_method',
        header: 'Routing Method',
        size: 100,
      },
      {
        accessorKey: 'tag_ids',
        header: 'Tag IDs',
        size: 100,
      },
      {
        accessorKey: 'wait_actions',
        header: 'Wait Actions',
        accessorFn: (row) => {
          return JSON.stringify(row.wait_actions);
        },
        size: 100,
      },
      {
        accessorKey: 'wait_timeout',
        header: 'Wait Timeout',
        size: 250,
      },
      {
        accessorKey: 'service_timeout',
        header: 'Service Timeout',
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
    tag_ids: false,
    wait_actions: false,
    tm_create: false,
    tm_delete: false,
  };

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/queues/queues_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }
  const Create = (row) => {
    const target = "/resources/queues/queues_create";
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
              <IconButton onClick={() => Detail(row)}>
                <Edit />
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

export default QueuesList
