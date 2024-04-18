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

const ChatbotsList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  },[]);

  const getList = (() => {
    const target = "chatbots?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("chatbots", tmpData);
      })
      .catch(e => {
        console.log("Could not get the list of chatbots info. err: %o", e);
        alert("Could not get the list of AI chatbots info.");
        setButtonDisable(false);
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
        accessorKey: 'engine_type',
        header: 'Engine Type',
      },
      {
        accessorKey: 'init_prompt',
        header: 'Init Prompt',
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
    init_prompt: false,
    tm_create: false,
    tm_delete: false,
  };

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/chatbots/chatbots_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }
  const Create = () => {
    const target = "/resources/chatbots/chatbots_create";
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

export default ChatbotsList
