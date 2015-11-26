/********************************************************
 * Copyright (C) 2012 VMware, Inc. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met: 
 *
 * 1. Redistributions of source code must retain the above copyright notice, this
 *    list of conditions and the following disclaimer. 
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution. 
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 * ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 * The views and conclusions contained in the software and documentation are those
 * of the authors and should not be interpreted as representing official policies, 
 * either expressed or implied, of the FreeBSD Project.
 *
 *********************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "vmci_sockets.h"


/*
 *----------------------------------------------------------------------------
 *
 * main --
 *
 *      Entry-point for the program.
 *
 * Results:
 *      0 on success, -1 on failure.
 *
 * Side effects:
 *      None.
 *
 *----------------------------------------------------------------------------
 */

int
main(int argc,       // IN
     char *argv[])   // IN
{
   int s = -1, c = -1;
   int af;
   int ret;
   int vmciFd;
   char buf[32];
   socklen_t addrLen;
   struct sockaddr_vm addr;

   /*
    * The address family for vSockets must be acquired, it is not static.
    * We hold onto the family we get back by keeping the fd to the device
    * open.
    */

   af = VMCISock_GetAFValueFd(&vmciFd);
   if (-1 == af) {
      fprintf(stderr, "Failed to get address family.\n");
      return -1;
   }

   /*
    * Open a STREAM socket using our address family.
    */

   ret = socket(af, SOCK_STREAM, 0);
   if (-1 == ret) {
      perror("Failed to open socket");
      goto exit;
   }

   s = ret;

   /*
    * Bind to an address on which we will listen for client connections.  We
    * use VMADDR_CID_ANY, which is the vSockets equivalent of INADDR_ANY, and
    * we listen on port 15000.
    */

   memset(&addr, 0, sizeof addr);
   addr.svm_family = af;
   addr.svm_cid = VMADDR_CID_ANY;
   addr.svm_port = 15000;
   ret = bind(s, (const struct sockaddr *)&addr, sizeof addr);
   if (-1 == ret) {
      perror("Failed to bind socket");
      goto exit;
   }

   /*
    * Get the address to which we were bound.
    */

   addrLen = sizeof addr;
   memset(&addr, 0, sizeof addr);
   ret = getsockname(s, (struct sockaddr *)&addr, &addrLen);
   if (-1 == ret) {
      perror("Failed to get socket address");
      goto exit;
   }

   /*
    * getsockname() returns the ANY context on which we bound, but we want
    * to log the actual context.  Try to grab it here.  If this is a guest,
    * then you can find this value in the .vmx file of this VM.  If this is
    * a host, then this value will be 2.
    */

   addr.svm_cid = VMCISock_GetLocalCID();
   printf("Listening on %d:%d...\n", addr.svm_cid, addr.svm_port);

   /*
    * And since this is the server-side, listen for client connections.
    */

   ret = listen(s, 1);
   if (-1 == ret) {
      perror("Failed to listen on socket");
      goto exit;
   }

   /*
    * Accept a connection from the client.
    */

   addrLen = sizeof addr;
   ret = accept(s, (struct sockaddr *)&addr, &addrLen);
   if (-1 == ret) {
      perror("Failed to accept connection");
      goto exit;
   }

   c = ret;

   printf("Connected to %d:%d.\n", addr.svm_cid, addr.svm_port);

   /*
    * Try to receive a message from the client.
    */ 

   memset(buf, 0, sizeof buf);
   ret = recv(c, buf, sizeof buf, 0);
   if (-1 == ret) {
      perror("Failed to receive");
      goto exit;
   }

   printf("Received '%s'.\n", buf);

   /*
    * And send one back.
    */

   memset(buf, 0, sizeof buf);
   strncpy(buf, "world", sizeof buf);
   ret = send(c, buf, strlen(buf) + 1, 0);
   if (strlen(buf) + 1 != ret) {
      perror("Failed to send");
      goto exit;
   }

   ret = 0;

exit:
   if (-1 != c) {
      close(c);
   }
   if (-1 != s) {
      close(s);
   }
   VMCISock_ReleaseAFValueFd(vmciFd);
   return ret;
}
