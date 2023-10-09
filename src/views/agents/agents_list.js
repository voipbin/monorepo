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

const AgentsList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  },[]);

  const getList = (() => {
    const target = "agents?page_size=100";

    ProviderGet(target).then(result => {
      const data = result.result;
      setListData(data);
      setIsLoading(false);

      const tmp = ParseData(data);
      const tmpData = JSON.stringify(tmp);
      localStorage.setItem("agents", tmpData);
    });

    // const tmp = JSON.parse(localStorage.getItem("agents"));
    // const data = Object.values(tmp);
    // setListData(data);
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
        accessorKey: 'username',
        header: 'Username',
        enableEditing: false,
        size: 100,
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
        accessorKey: 'addresses',
        header: 'Addresses',
        accessorFn: (row) => {
          return JSON.stringify(row.addresses);
        },
      },
      {
        accessorKey: 'tag_ids',
        header: 'Tag IDs',
        accessorFn: (row) => {
          return JSON.stringify(row.tag_ids);
        },
        size: 100,
      },
      {
        accessorKey: 'ring_method',
        header: 'Ring Method',
      },
      {
        accessorKey: 'status',
        header: 'Status',
        enableEditing: false,
      },
      {
        accessorKey: 'permission',
        header: 'Permission',
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
    ],
    [],
  );

  const columnVisibility = {
    id: false,
    addresses: false,
    tag_ids: false,
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

    const target = "agents/" + row.getValue('id');
    ProviderDelete(target).then(() => {
      console.log("Deleted queue.");
    });
  }

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/agents/agents_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }
  const Create = () => {
    const target = "/resources/agents/agents_create";
    console.log("navigate target: ", target);
    navigate(target);
  }

  return (
    <>
      <MaterialReactTable
        columns={listColumns}
        data={listData ?? []} // data?.data ?? []
        enableRowNumbers
        state={{
          isLoading: isLoading,
        }}
        enableRowActions
        renderRowActions={({ row, table }) => (
          <Box sx={{ display: 'flex' }}>
            <Tooltip arrow placement="left" title="Edit">
              <IconButton onClick={() => {
                Detail(row);
              }}>
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

export default AgentsList
