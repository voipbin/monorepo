/* eslint-disable */
import { createContext } from 'react'

let Service = {
  count: 0,
  count2: 0,
}


var ServiceContext = createContext(Service);



let service = {
  count: 0,
  count2: 0,
}

function IncreaseCount() {
  service.count++
  return service.count
}

export {
  service,
  IncreaseCount,
  ServiceContext,
}
