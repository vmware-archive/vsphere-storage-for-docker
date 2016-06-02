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
//
// Used in python as direct calles to shared library
// Needs to be build as 32bit due to the need to run in 32bit python (due to
// VSI libs available only in 32 bits

// useful ref on ctypes:  http://starship.python.net/crew/theller/ctypes/tutorial.html

// TODO: return meaningful error codes. Issue #206

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <stdint.h>

#include "vmci_sockets.h"
#include "connection_types.h"


// Returns vSocket to listen on, or -1.
// errno indicates the reason for a failure, if any.
int
vmci_init(void)
{
   struct sockaddr_vm addr;
   socklen_t addrLen;
   int ret;
   int socket_fd; // socket to open
   int saved_errno; // buffer for retaining errno
   int af = vsock_get_family(); // socket family for vSockets communication

   if (af == -1) {
      return CONN_FAILURE;
   }

   /*
    * Open a STREAM socket using our address family.
    */

   socket_fd = socket(af, SOCK_STREAM, 0);
   if (socket_fd == -1) {
      perror("Failed to open socket");
      return CONN_FAILURE;
   }

   /*
    * Bind to an address on which we will listen for client connections.  We
    * use VMADDR_CID_ANY, which is the vSockets equivalent of INADDR_ANY, and
    * we listen on port 15000.
    */

   memset(&addr, 0, sizeof addr);
   addr.svm_family = af;
   addr.svm_cid = VMADDR_CID_ANY;
   addr.svm_port = 15000;
   ret = bind(socket_fd, (const struct sockaddr *) &addr, sizeof addr);
   if (ret == -1) {
      saved_errno = errno;
      perror("Failed to bind socket");
      close(socket_fd);
      errno = saved_errno;
      return CONN_FAILURE;
   }

   return socket_fd;
}

// Returns vSocket to communicate on (which needs to be closed later),
// or -1 on error
int
vmci_get_one_op(const int s,    // socket to listen on
         uint32_t *vmid, // cartel ID for VM
         char *buf,      // external buffer to return json string
         const int bsize // buffer size
         )
{
   int ret; // keep return values here
   uint32_t b; // smallish buffer
   int saved_errno; // retain errno when needed
   socklen_t addrLen;
   struct sockaddr_vm addr;
   int client_socket = -1; // connected socket to talk to client
   int af = vsock_get_family(); // socket family for vSockets communication

   if (af == -1) {
      return CONN_FAILURE;
   }

   /*
    * listen for client connections.
    */
   ret = listen(s, 1);
   if (ret == -1) {
      perror("Failed to listen on socket");
      return CONN_FAILURE;
   }

   addrLen = sizeof addr;
   client_socket = accept(s, (struct sockaddr *) &addr, &addrLen);
   if (client_socket == -1) {
      perror("Failed to accept connection");
      return CONN_FAILURE;
   }

   // get VMID. We really get CartelID for VM, but it will make do
   socklen_t len = sizeof(*vmid);
   if (getsockopt(client_socket, af, SO_VMCI_PEER_HOST_VM_ID, vmid, &len) == -1 || len
            != sizeof(*vmid)) {
      perror("sockopt SO_VMCI_PEER_HOST_VM_ID failed, continuing...");
      // will still try to recv message to know what was there - so no return
   }

   /*
    * Try to receive a message from the client.
    * the message has MAGIC, length, and the actual data.
    *
    */

   // get magic:
   b = 0;
   ret = recv(client_socket, &b, sizeof b, 0);
   if (ret == -1 || b != MAGIC) {
      saved_errno = errno;
      fprintf(stderr,
               "Failed to receive magic: ret %d (%s) got 0x%x (expected 0x%x)\n",
               ret, strerror(errno), b, MAGIC);
      close(client_socket);
      errno = saved_errno;
      return CONN_FAILURE;
   }

   // get length:
   b = 0;
   ret = recv(client_socket, &b, sizeof b, 0);
   if (ret == -1) {
      saved_errno = errno;
      fprintf(stderr, "Failed to receive len: ret %d (%s) got %d\n", ret,
               strerror(errno), b);
      close(client_socket);
      errno = saved_errno;
      return CONN_FAILURE;
   }

   if (b > bsize) {
      fprintf(stderr, "Query is too large: %d (max %d)\n", b, bsize);
      close(client_socket);
      errno = ERANGE; // result too large for the buffer
      return CONN_FAILURE;
   }

   memset(buf, 0, b);
   ret = recv(client_socket, buf, b, 0);
   if (ret != b) {
      saved_errno = errno;
      fprintf(stderr, "Failed to receive content: ret %d (%s) expected %d\n",
               ret, strerror(errno), b);
      close(client_socket);
      errno = saved_errno;
      return CONN_FAILURE;
   }
   // do protocol sanity check
   if (strlen(buf) + 1 != b) {
      fprintf(stderr, "Protocol error: len mismatch, expected %d, got %d\n",
               strlen(buf), b);
      close(client_socket);
      errno = EBADMSG;
      return CONN_FAILURE;
   }

   return client_socket;
}

// Sends a single reply on a socket.
// Returns 0 on OK and -1 on error (errno is set in this case.
// For errors, "reply" contains extra error info (specific for vmci_reply)
int
vmci_reply(const int client_socket,      // socket to use
         const char *reply // (json) to send back
         )
{
   int ret; // keep return values here
   int saved_errno; // retain errno when needed
   uint32_t b; // smallish buffer

   // Just being paranoid...
   if (reply == NULL) {
      reply = "OK";
   }

   /*
    * And send one word back.
    */

   b = MAGIC;
   ret = send(client_socket, &b, sizeof(b), 0);
   if (ret != sizeof(b)) {
      saved_errno = errno;
      reply = "Failed to send magic";
      fprintf(stderr, "%s: ret %d (%s) expected size %d\n", reply, ret,
               strerror(errno), sizeof(b));
      goto failed;
   }

   b = strlen(reply) + 1; // send the string and trailing \0
   ret = send(client_socket, &b, sizeof(b), 0);
   if (ret != sizeof(b)) {
      saved_errno = errno;
      reply = "Failed to send len";
      fprintf(stderr, "%s: ret %d (%s) expected size %d\n", reply, ret,
               strerror(errno), sizeof(b));
      goto failed;
   }

   ret = send(client_socket, reply, b, 0);
   if (b != ret) {
      saved_errno = errno;
      fprintf(stderr, "Failed to send content: ret %d (%s) expected size %d\n",
               ret, strerror(errno), b);
      goto failed;
   }

   // success
   close(client_socket);
   return CONN_SUCCESS;

   // failure
failed:
   close(client_socket);
   errno = saved_errno;
   return CONN_FAILURE;
}

// Closes a socket.
void
vmci_close(int s)
{

   if (s != -1) {
      close(s);
   }
}
