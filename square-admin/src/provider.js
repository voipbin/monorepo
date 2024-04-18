import React, { useState } from 'react'
import { useNavigate } from "react-router-dom";
import FileSaver from "file-saver";



const hostname = 'https://api.voipbin.net/v1.0/'

export const Get = (target) => {
    return new Promise((resolve, reject) => {
        let authToken = localStorage.getItem("token");

        let requestURI = hostname + target;
        if (!requestURI.includes('?')) {
            requestURI = requestURI + '?token=' + authToken
        } else {
            requestURI = requestURI + '&token=' + authToken
        }

        let request = new Request(requestURI, {
            method: 'GET',
            headers: new Headers({ 'Content-Type': 'application/json' }),
        });

        fetch(request)
            .then(response => {
                if (response.status == 401) {
                    localStorage.setItem("token", "");
                } else if (response.status < 200 || response.status >= 300) {
                    throw new Error(response.statusText);
                }
                return resolve(response.json());
            })
            .catch(e => {
                console.log("Could not fetch the data. err: ", e);
                reject(new Error("Could not fetch the data. err: %o", e));
            });
    })
}

export const Post = (target, data) => {
    return new Promise((resolve, reject) => {
        let authToken = localStorage.getItem("token");
        let request = new Request(hostname + target + '?token=' + authToken, {
            method: 'POST',
            headers: new Headers({ 'Content-Type': 'application/json' }),
            body: data,
        });

        fetch(request)
            .then(response => {
                if (response.status == 401) {
                    localStorage.setItem("token", "");
                } else if (response.status < 200 || response.status >= 300) {
                    throw new Error(response.statusText);
                }
                return resolve(response.json());
            })
            .catch(e => {
                console.log("Could not fetch the data. err: ", e);
                reject(new Error("Could not fetch the data. err: %o", e));
            });
    });
}


export const Put = (target, data) => {
    return new Promise((resolve, reject) => {
        let authToken = localStorage.getItem("token");
        let request = new Request(hostname + target + '?token=' + authToken, {
            method: 'PUT',
            headers: new Headers({ 'Content-Type': 'application/json' }),
            body: data,
        });

        fetch(request)
            .then(response => {
                if (response.status == 401) {
                    localStorage.setItem("token", "");
                } else if (response.status < 200 || response.status >= 300) {
                    throw new Error(response.statusText);
                }
                return resolve(response.json());
            })
            .catch(e => {
                console.log("Could not fetch the data. err: ", e);
                reject(new Error("Could not fetch the data. err: %o", e));
            });
    });
}

export const Delete = (target) => {
    return new Promise((resolve, reject) => {
        let authToken = localStorage.getItem("token");
        let request = new Request(hostname + target + '?token=' + authToken, {
            method: 'DELETE',
        });

        fetch(request)
            .then(response => {
                if (response.status == 401) {
                    localStorage.setItem("token", "");
                } else if (response.status < 200 || response.status >= 300) {
                    throw new Error(response.statusText);
                }
                return resolve(response.json());
            })
            .catch(e => {
                console.log("Could not fetch the data. err: ", e);
                reject(new Error("Could not fetch the data. err: %o", e));
            });
    });
}

export const Download = (target, filename) => {
  console.log("Downloading the file. target: %s, filename: %s", target, filename);
  let authToken = localStorage.getItem("token");
  const tmpTarget = hostname + target + '?token=' + authToken;
  saveAs(tmpTarget, filename);  
}

export const ParseData = (data) => {
    const res = {};
    data.forEach(tmp => {
        res[tmp.id] = tmp
    });

    return res;
}

export const LoadResource = (resource) => {
  const target = resource + "?page_size=100";
  Get(target).then(result => {
    const data = result.result;
    const tmp = ParseData(data);

    const tmpData = JSON.stringify(tmp);
    localStorage.setItem(resource, tmpData);
  });
}

const resources = [
    // "activeflows",
    // "agents",
    // "billing_accounts",
    "calls",
    // "campaigns",
    // "chatbots",
    // "chats",
    // "conferences",
    // "conferencecalls",
    // "customers",
    // "domains",
    // "extensions",
    // "flows",
    // "groupcalls",
    // "messages",
    // "numbers",
    // "queuecalls",
    // "queues",
    // "trunks",
]

export const LoadResourcesAll = (resource) => {
  // load the resources
  resources.forEach(r => {
    LoadResource(r);
  })
}
