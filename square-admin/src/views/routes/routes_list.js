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
  CFormInput,
  CRow,
  CCol,
} from '@coreui/react';
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';
import { useNavigate } from "react-router-dom";

const RoutesList = () => {
  console.log("RoutesList");

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchCustomerID, setSearchCustomerID] = useState("00000000-0000-0000-0000-000000000001");

  useEffect(() => {
    getList();
    return;
  }, []);

  const searchRoutes = ((tmpCustomerID) => {
    const target = "routes?customer_id=" + tmpCustomerID;
    console.log("searchRoutes. target: ", target);

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);
      })
      .catch(e => {
        console.log("Could not get a resource list. err: %o", e);
        alert("Could not not get a resource list.");
      });
  });

  const getList = (() => {
    const target = "routes?page_size=100";
    console.log("getList. target: ", target);

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);

        const tmp = ParseData(data);
        const tmpData = JSON.stringify(tmp);
        localStorage.setItem("routes", tmpData);
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
        accessorKey: 'customer_id',
        header: 'Customer ID',
        size: 350,
      },
      {
        accessorKey: 'priority',
        header: 'Priority',
      },
      {
        accessorKey: 'target',
        header: 'Target',
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
    const target = "/resources/routes/routes_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }
  const Create = () => {
    const target = "/resources/routes/routes_create/" + searchCustomerID;
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
          <CRow>
            <CCol xs={8}>
              <CFormInput
                type="text"
                placeholder="00000000-0000-0000-0000-000000000001"
                aria-label="default input example"
                size="sm"
                onChange={(evt) => {
                  const value = evt.nativeEvent.target.value;
                  setSearchCustomerID(value);
                }}
              />
            </CCol>
            <CCol xs>
              <Button
                color="primary"
                size="sm"
                onClick={() => {
                  setIsLoading(true);
                  searchRoutes(searchCustomerID.toLowerCase());
                }}
                variant="contained"
              >
                Search
              </Button>
            </CCol>
            <CCol xs>
              <Button
                color="secondary"
                onClick={() => {
                  Create(true);
                }}
                variant="contained"
              >
                Create
              </Button>
            </CCol>
          </CRow>
        )}
      />
    </>
  )
}

export default RoutesList
