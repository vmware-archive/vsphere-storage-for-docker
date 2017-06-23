// Copyright 2017 VMware, Inc. All Rights Reserved.
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


//
// VMCI sockets communication - client side.
//
// API: Exposes Vmci_GetReply and Vmci_FreeBuf.
//

#include <stdlib.h>
#include <errno.h>
#include <stdint.h>

#include "vmci_sockets.h"
#include "connection_types.h"

#define ERR_BUF_LEN 512
#define MAXBUF 1024 * 1024 // Safety limit. We do not expect json string > 1M
#define MAX_CLIENT_PORT 1023 // Last privileged port
#define START_CLIENT_PORT 100 // Where to start client port
#define BIND_RETRY_COUNT (MAX_CLIENT_PORT - START_CLIENT_PORT) // Retry entire range on bind failures

// Operations status. 0 is OK
typedef int be_sock_status;

//
// Booking structure for opened VMCI / vSocket
//
typedef struct {
   int sock_id; // socket id for socket APIs
   struct sockaddr_vm addr; // held here for bookkeeping and reporting
} be_sock_id;

//
// Protocol message structure: request and reply
//
typedef struct be_request {
   uint32_t mlen;   // length of message (including trailing \0)
   const char *msg; // null-terminated immutable JSON string.
} be_request;

typedef struct be_answer {
   char *buf;                  // response buffer
   char errBuf[ERR_BUF_LEN];   // error response buffer
} be_answer;

//
// Entry point for vsocket requests.
// Returns NULL for success, -1 for err, and sets errno if needed
// <ans> is allocated upstairs
//
const be_sock_status
Vmci_GetReply(int port, const char* json_request, const char* be_name, be_answer* ans);

//
// Frees a be_answer instance.
//
void
Vmci_FreeBuf(be_answer *ans);
