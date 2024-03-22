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
import { MaterialReactTable } from 'material-react-table';
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";

const Messages = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  }, []);

  const getList = (() => {
    const target = "messages?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("messages", tmpData);
      })
      .catch(e => {
        console.log("Could not get a list of messages. err: %o", e);
        alert("Could not not get a list of messages.");
      });
  });

  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'type',
        header: 'type',
        size: 80,
      },
      {
        accessorKey: 'source.target',
        header: 'source number',
        size: 250,
      },
      {
        accessorKey: 'targets.length',
        header: 'targets',
        size: 150,
      },
      {
        accessorKey: 'direction',
        header: 'direction',
        size: 150,
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
    const target = "/resources/messages/messages_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }

  const Create = () => {
    const target = "/resources/messages/messages_create";
    console.log("navigate target: ", target);
    navigate(target);
  }

  return (
    <>
      <MaterialReactTable
        columns={listColumns}
        data={listData}
        state={{
          isLoading: isLoading,
        }}
        enableRowNumbers
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

export default Messages
