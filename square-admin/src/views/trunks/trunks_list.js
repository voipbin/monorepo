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

const TrunksList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  },[]);

  const getList = (() => {
    const target = "trunks?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("trunks", tmpData);
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
        accessorKey: 'domain_name',
        header: 'Domain Name',
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
        accessorKey: 'auth_types',
        header: 'Auth Types',
      },
      {
        accessorKey: 'tm_update',
        header: 'Update Time',
        enableEditing: false,
        size: 250,
      },
    ],
    [],
  );

  const columnVisibility = {
    id: false,
  };

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/trunks/trunks_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }
  const Create = () => {
    const target = "/resources/trunks/trunks_create";
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
              Create(true);
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

export default TrunksList
