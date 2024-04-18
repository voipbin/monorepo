import React, { useMemo, useState, useEffect } from 'react'
import {
  CButton,
  CModal,
  CModalBody,
  CModalFooter,
  CModalHeader,
  CModalTitle,
} from '@coreui/react'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  MenuItem,
  Stack,
  TextField,
  Tooltip,
  Typography,
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


const CustomersList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  }, []);

  const getList = (() => {

    const target = "customers?page_size=100";

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("customers", tmpData);
      })
      .catch(e => {
        console.log("Could not get the list. err: %o", e);
        alert("Could not get the list.");
      });
  });

  // show list
  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'id',
        header: 'id',
        enableEditing: false,
      },
      {
        accessorKey: 'name',
        header: 'name',
        size: 100,
      },
      {
        accessorKey: 'detail',
        header: 'detail',
        size: 100,
      },
      {
        accessorKey: 'email',
        header: 'email',
        size: 100,
      },
      {
        accessorKey: 'webhook_method',
        header: 'webhook method',
        size: 100,
      },
      {
        accessorKey: 'webhook_uri',
        header: 'webhook uri',
        size: 100,
      },
      {
        accessorKey: 'permission_ids',
        header: 'permission ids',
        size: 100,
      },
      {
        accessorKey: 'billing_account_id',
        header: 'billing account id',
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
    webhook_method: false,
    webhook_uri: false,
    permission_ids: false,
    tm_create: false,
    tm_delete: false,
  };

  const navigate = useNavigate();
  const Detail = (row) => {
    const target = "/resources/customers/customers_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }
  const Create = (row) => {
    const target = "/resources/customers/customers_create";
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

export default CustomersList
