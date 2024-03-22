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
import {
  CFormInput,
  CRow,
  CCol,
} from '@coreui/react';
import { Delete, Edit, ShoppingCart, } from '@mui/icons-material';
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

const BuyList = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchCountry, setSearchCountry] = useState("us");

  useEffect(() => {
    searchAvailableNumbers("us");
    return;
  }, []);

  const searchAvailableNumbers = ((tmpCountry) => {
    const target = "available_numbers?page_size=10&country_code=" + tmpCountry;

    ProviderGet(target)
      .then(result => {
        const data = result.result;
        setListData(data);
        setIsLoading(false);
      })
      .catch(e => {
        console.log("Could not get a list of available numbers. err: %o", e);
        alert("Could not not get a list of available numbers.");
      });
  });


  // show list
  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'number',
        header: 'Number',
        enableEditing: false,
      },
      {
        accessorKey: 'country',
        header: 'Country',
        enableEditing: false,
      },
      {
        accessorKey: 'postal_code',
        header: 'Postal Code',
        enableEditing: false,
      },
      {
        accessorKey: 'features',
        header: 'Features',
        enableEditing: false,
        accessorFn: (row) => {
          return JSON.stringify(row.features);
        },
      },
    ],
    [],
  );

  const columnVisibility = {};

  const navigate = useNavigate();
  const Create = (row) => {
    const target = "/resources/numbers/buy_create/" + row.original.number;
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
            <Tooltip arrow placement="right" title="Buy">
              <IconButton color="error" onClick={() =>
                Create(row)
              }>
                <ShoppingCart />
              </IconButton>
            </Tooltip>
          </Box>
        )}

        state={{
          isLoading: isLoading,
        }}

        muiTableBodyRowProps={({ row, table }) => ({
          onDoubleClick: (event) => {
            Create(row);
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
        muiSearchTextFieldProps={{
          placeholder: 'Search All Props',
          sx: { minWidth: '18rem' },
          variant: 'outlined',
        }}
        renderTopToolbarCustomActions={() => (
          <CRow>
            <CCol xs>
              <CFormInput
                type="text"
                placeholder="US"
                aria-label="default input example"
                onChange={(evt) => {
                  const value = evt.nativeEvent.target.value;
                  setSearchCountry(value);
                }}
              />
            </CCol>
            <CCol xs>
              <Button
                color="secondary"
                onClick={() => {
                  setIsLoading(true);
                  searchAvailableNumbers(searchCountry.toLowerCase());
                }}
                variant="contained"
              >
                SEARCH
              </Button>
            </CCol>
          </CRow>
        )}
      />
    </>
  )
}

export default BuyList
