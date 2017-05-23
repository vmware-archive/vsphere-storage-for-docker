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


//
// VMCI sockets communication - client side.
//
// Called mainly from Go code.
//
// API: Exposes only Vmci_GetReply. The call is blocking.
//
//
#pragma once

#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <stdint.h>
#include <assert.h>

#include "vmci_sockets.h"
#include "connection_types.h"

#define ERR_BUF_LEN 512

// operations status. 0 is OK
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

#define MAXBUF 1024 * 1024 // Safety limit. We do not expect json string > 1M
#define MAX_CLIENT_PORT 1023 // Last privileged port
#define START_CLIENT_PORT 100 // Where to start client port

// Retry entire range on bind failures
#define BIND_RETRY_COUNT (MAX_CLIENT_PORT - START_CLIENT_PORT)

typedef struct be_answer {
   char *buf;                  // response buffer
   char errBuf[ERR_BUF_LEN];   // error response buffer
} be_answer;

//
// Interface for communication to "command execution" server.
//
typedef struct be_funcs {
   const char *shortName; // name of the interaface (key to access it)
   const char *name;      // longer explanation (human help)

   // init the channel, return status and ID
   be_sock_status
   (*init_sock)(be_sock_id *id, int cid, int port);
   // release the channel - clean up
   void
   (*release_sock)(be_sock_id *id);

   // send a request and get  reply - blocking
   be_sock_status
   (*get_reply)(be_sock_id *id, be_request *r, be_answer* a);
} be_funcs;

// support communication interfaces
#define VSOCKET_BE_NAME "vsocket" // backend to communicate via vSocket
#define ESX_VMCI_CID    2  		  // ESX host VMCI CID ("address")
#define DUMMY_BE_NAME "dummy"     // backend which only returns OK, for unit test


// Get backend by name
static be_funcs *
get_backend(const char *shortName);

// "dummy" interface implementation
// Used for manual testing mainly,
// to make sure data arrives to backend
//----------------------------------
static be_sock_status
dummy_init(be_sock_id *id, int cid, int port);

static void
dummy_release(be_sock_id *id);

static be_sock_status
dummy_get_reply(be_sock_id *id, be_request *r, be_answer* a);


// vsocket interface implementation
//---------------------------------



// Create and connect VMCI socket.
// return CONN_SUCCESS (0) or CONN_FAILURE (-1)
static be_sock_status
vsock_init(be_sock_id *id, int cid, int port);

//
// Send request (r->msg) and wait for reply.
// returns 0 on success , -1 (or potentially errno) on error
// On success , allocates a->buf ( caller needs to free it) and placed reply there
// Expects r and a to be allocated by the caller.
//
//
static be_sock_status
vsock_get_reply(be_sock_id *s, be_request *r, be_answer* a);

// release socket and vmci info
static void
vsock_release(be_sock_id *id);

//
// Handle one request using BE interface
// Yes,  we DO create and bind socket for each request - it's management
// so we can afford overhead, and it allows connection to be stateless.
//
static be_sock_status
host_request(be_funcs *be, be_request* req, be_answer* ans, int cid, int port);

//
//
// Entry point for vsocket requests.
// Returns NULL for success, -1 for err, and sets errno if needed
// <ans> is allocated upstairs
//
const be_sock_status
Vmci_GetReply(int port, const char* json_request, const char* be_name,
              be_answer* ans);

void
Vmci_FreeBuf(be_answer *ans);
