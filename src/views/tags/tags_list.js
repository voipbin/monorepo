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

const TagsList = () => {
  console.log("TagsList");

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  }, []);

  const getList = (() => {
    const target = "tags?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("tags", tmpData);
      })
      .catch(e => {
        console.log("Could not get a resource list. err: %o", e);
        alert("Could not not get a resource list.");
      });
  });

  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'id',
        header: 'id',
        size: 250,
      },
      {
        accessorKey: 'name',
        header: 'name',
        size: 150,
      },
      {
        accessorKey: 'detail',
        header: 'detail',
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
    const target = "/resources/tags/tags_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }

  const Create = () => {
    const target = "/resources/tags/tags_create";
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

export default TagsList
