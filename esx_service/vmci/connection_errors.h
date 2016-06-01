// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Errors on vmci command communication channel

#ifndef _CONNECTION_ERRORS_H_
#define _CONNECTION_ERRORS_H_

typedef enum VMCI_ConnectionError {
   CONN_SUCCESS = 0,

   // Misc. protocol errors. When changing, also update info[] below
   CONN_FAILED_VMCI_ADDRESS_FAMILY_GET,
   CONN_FAILED_VMCI_ADDRESS_FAMILY_MISSING,
   CONN_FAILED_VSOCKET_OPEN,
   CONN_FAILED_VSOCKET_BIND,
   CONN_FAILED_VSOCKET_LISTEN,
   CONN_FAILED_VSOCKET_ACCEPT,
   CONN_FAILED_TO_CONNECT,
   CONN_FAILED_SOCKADDR_GET,
   CONN_FAILED_MAGIC_SEND,
   CONN_FAILED_LEN_SEND,
   CONN_FAILED_CONTENT_SEND,
   CONN_FAILED_MAGIC_RECIEVE,
   CONN_FAILED_LEN_RECEIVE,
   CONN_FAILED_CONTENT_RECIEVE,
   CONN_LEN_MISMATCH,
   CONN_MALLOC_FAILED,
   CONN_BUF_TOO_SMALL,
   CONN_FAILED_LEN_MISMATCH,

   // Bad communication back end name (internal usage only)
   CONN_BAD_BE_NAME
} VMCI_ConnectionError;

// gets error string for a connection error
static const char*
VMCI_ErrorStr(VMCI_ConnectionError id)
{
   int i;
   struct {
      VMCI_ConnectionError id;
      const char *msg;
   } info[] = {
      { CONN_FAILED_VMCI_ADDRESS_FAMILY_GET, "failed to get VMCI Address Family"},
      { CONN_FAILED_VMCI_ADDRESS_FAMILY_MISSING, "missing VMCI AF (internal error)"},
      { CONN_FAILED_VSOCKET_OPEN, "failed to open vSocket"},
      { CONN_FAILED_VSOCKET_BIND, "failed to bind vSocket"},
      { CONN_FAILED_VSOCKET_LISTEN, "failed to listen on vSocket"},
      { CONN_FAILED_VSOCKET_ACCEPT, "failed on accept on vSocket"},
      { CONN_FAILED_TO_CONNECT, "failed to connect to vSocket"},
      { CONN_FAILED_SOCKADDR_GET, "failed sockadr get"},
      { CONN_FAILED_MAGIC_SEND, "failed MAGIC send"},
      { CONN_FAILED_LEN_SEND, "failed LEN send"},
      { CONN_FAILED_CONTENT_SEND, "failed content send"},
      { CONN_FAILED_MAGIC_RECIEVE, "failed MAGIC receive"},
      { CONN_FAILED_LEN_RECEIVE, "failed LEN receive"},
      { CONN_FAILED_CONTENT_RECIEVE, "failed to receive content"},
      { CONN_MALLOC_FAILED, "failed malloc"},
      { CONN_BUF_TOO_SMALL, "request buffer is too small"},
      { CONN_LEN_MISMATCH, "message length mismatch"},
      { CONN_BAD_BE_NAME, "bad back end name"},

      { CONN_SUCCESS, NULL}
   };

   for (i=0; info[i].msg != NULL;  i++) {
      if (info[i].id == id) {
         return info[i].msg;
      }
   }
   return "Unknown error";
}

#endif // _CONNECTION_ERRORS_H_

