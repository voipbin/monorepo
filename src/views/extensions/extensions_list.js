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


const ExtensionsList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getList();
    return;
  }, []);

  const getList = (() => {
    const tmp = JSON.parse(localStorage.getItem("extensions"));
    const data = Object.values(tmp);
    setListData(data);
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
        accessorKey: 'extension',
        header: 'Extension',
        size: 100,
      },
      {
        accessorKey: 'domain_id',
        header: 'Domain ID',
        size: 250,
      },
      {
        accessorKey: 'tm_update',
        header: 'update time',
        enableEditing: false,
        size: 250,
      },
    ],
    [],
  );

  const columnVisibility = {
    id: false,
  };

  const handleDeleteRow = (row) => {
    console.log("Deleting row. ", row)

    if (
      !confirm(`Are you sure you want to delete ${row.getValue('username')}`)
    ) {
      return;
    }

    const target = "customers/" + row.getValue('id');
    ProviderDelete(target).then(() => {
      console.log("Deleted customer.");
    });
  }

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
        enableRowNumbers
        enableRowActions
        renderRowActions={({ row, table }) => (
          <Box sx={{ display: 'flex' }}>
            <Tooltip arrow placement="left" title="Edit">
              <IconButton
                onClick={() => {
                  Detail(row);
                }
              }>
                <Edit />
              </IconButton>
            </Tooltip>
            <Tooltip arrow placement="right" title="Delete">
              <IconButton
                color="error"
                onClick={() =>
                  handleDeleteRow(row)
                }>
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

export default ExtensionsList
