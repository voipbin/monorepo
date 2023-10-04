import React, { useMemo, useState, useEffect } from 'react'
import {
  CButton,
  CModal,
  CModalBody,
  CModalFooter,
  CModalHeader,
  CModalTitle,
} from '@coreui/react'
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


const Activeflows = () => {

  const [listData, setListData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getActiveflows();
    return;
  }, []);

  const getActiveflows = (() => {
    const tmp = JSON.parse(localStorage.getItem("activeflows"));
    const data = Object.values(tmp);
    setListData(data);
  });

  const listColumns = useMemo(
    () => [
      {
        accessorKey: 'reference_type',
        header: 'reference type',
        size: 150,
      },
      {
        accessorKey: 'reference_id',
        header: 'reference id',
        size: 250,
      },
      {
        accessorKey: 'status',
        header: 'status',
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
    const target = "/resources/flows/activeflows_detail/" + row.original.id;
    console.log("navigate target: ", target);
    navigate(target);
  }

  return (
    <>
      <MaterialReactTable
        columns={listColumns}
        data={listData}
        enableRowNumbers
        muiTableBodyRowProps={({ row }) => ({
          onDoubleClick: (event) => {
            Detail(row);
          },
        })}
      />
    </>
  )
}

export default Activeflows
