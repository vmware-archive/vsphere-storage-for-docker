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
// Simple C library to do VMCI / vSocket listen
//
// Based on vsocket usage example so quite clumsy.

// TODO: return meaningful error codes. Issue #206

#pragma once

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <stdint.h>

#include "vmci_sockets.h"
#include "connection_types.h"


// SO_QSIZE maximum number of connections (requests) in socket queue.
int SO_QSIZE = 128;

// Returns vSocket to listen on, or -1.
// errno indicates the reason for a failure, if any.
int
vmci_init(unsigned int port);

// Returns vSocket to communicate on (which needs to be closed later),
// or -1 on error
int
vmci_get_one_op(const int s,    // socket to listen on
         uint32_t *vmid, // cartel ID for VM
         char *buf,      // external buffer to return json string
         const int bsize // buffer size
     );

// Sends a single reply on a socket.
// Returns 0 on OK and -1 on error (errno is set in this case.
// For errors, "reply" contains extra error info (specific for vmci_reply)
int
vmci_reply(const int client_socket,      // socket to use
         const char *reply // (json) to send back
     );

// Closes a socket.
void
vmci_close(int s);
