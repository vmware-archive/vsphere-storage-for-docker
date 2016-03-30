//
// Simple C library to do VMCI / vSocket listen
//
// Based on vsocket usage example so quite clumsy.
//
// Used in python as direct calles to shared library
// Needs to be build as 32bit due to the need to run in 32bit python (due to
// VSI libs available only in 32 bits

// useful ref on ctypes:  http://starship.python.net/crew/theller/ctypes/tutorial.html

// TODO: cleanup, err mgmt. see errcheck attribute for better error checking

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <stdint.h>

#include "vmci_sockets.h"

#define MAGIC 0xbadbeef

// TODO: get shared lib for client and server
static int
vsock_get_family(void)
{
   static int af = -1;

   if (af == -1) { // Note: this may leak FDs in multi-threads. TBD: lock.
      af = VMCISock_GetAFValue();
   }
   return af;
}

// returns socket to listen to, or -1
// also fills in af and vmciFd
int
vmci_init(void)
{
   struct sockaddr_vm addr;
   socklen_t addrLen;
   int af; // socket family for VMCI communication
   int ret;
   int s; // socket to open


   /*
    * The address family for vSockets must be acquired, it is not static.
    * We hold onto the family we get back by keeping the fd to the device
    * open.
    */

   af = vsock_get_family();
   if (af == -1) {
      fprintf(stderr, "Failed to get address family.\n");
      return -1;
   }

   /*
    * Open a STREAM socket using our address family.
    */

   s = socket(af, SOCK_STREAM, 0);
   if (s == -1) {
      perror("Failed to open socket");
      return -1;
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
   ret = bind(s, (const struct sockaddr *) &addr, sizeof addr);
   if (ret == -1) {
      perror("Failed to bind socket");
      close(s);
      return -1;
   }

   /*
    * Get the address to which we were bound.
    */

   addrLen = sizeof addr;
   memset(&addr, 0, sizeof addr);
   ret = getsockname(s, (struct sockaddr *) &addr, &addrLen);
   if (ret == -1) {
      perror("Failed to get socket address");
      close(s);
      return -1;
   }

   /*
    * getsockname() returns the ANY context on which we bound, but we want
    * to log the actual context.  Try to grab it here.  If this is a guest,
    * then you can find this value in the .vmx file of this VM.  If this is
    * a host, then this value will be 2.
    */

   addr.svm_cid = VMCISock_GetLocalCID();

   return s;
}

// returns socket to communicate on (which neds to be closed later),
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
   socklen_t addrLen;
   struct sockaddr_vm addr;
   int c = -1; // connected socket
   int af = vsock_get_family();

   if (af == -1)  {
      printf("Internal error - FAMILY (af) value is not set\n");
      return -1;
   }
   /*
    * listen for client connections.
    */
   ret = listen(s, 1);
   if (ret == -1) {
      perror("Failed to listen on socket");
      return -1;
   }

   addrLen = sizeof addr;
   c = accept(s, (struct sockaddr *) &addr, &addrLen);
   if (c == -1) {
      perror("Failed to accept connection");
      return -1;
   }

   // get VMID. We really get CartelID for VM, but it will make do
   socklen_t len = sizeof(*vmid);
   if (getsockopt(c, af, SO_VMCI_PEER_HOST_VM_ID, vmid, &len) == -1
            || len != sizeof(*vmid)) {
      perror("sockopt SO_VMCI_PEER_HOST_VM_ID failed, continuing...");
      // will still try to recv message to know what was there - so no return
   }

   /*
    * Try to receive a message from the client.
    * the message has MAGIC, length, and the actual data.
    *
    */
   // magic:
   b = 0;
   ret = recv(c, &b, sizeof b, 0);
   if (ret == -1 || b != MAGIC) {
      printf("Failed to receive magic: ret %d (%s) got 0x%x (expected 0x%x)\n",
               ret, strerror(errno), b, MAGIC);
      close(c);
      return -1;
   }

   // length:
   b = 0;
   ret = recv(c, &b, sizeof b, 0);
   if (ret == -1) {
      printf("Failed to receive len: ret %d (%s) got %d\n",
               ret, strerror(errno), b);
      close(c);
      return -1;
   }

   if (b > bsize) {
      // passed in buffer too small !
      printf("Query is too large, can't handle: %d (max %d)\n", b, bsize);
      close(c);
      return -1;
   }

   memset(buf, 0, b);
   ret = recv(c, buf, b, 0);
   if (ret != b) {
      printf("Failed to receive content: ret %d (%s) expected %d\n",
             ret, strerror(errno), b);
      close(c);
      return -1;
   }
   // protocol sanity check
   if (strlen(buf) + 1 != b) {
      printf("Protocol error: len mismatch, expected %d, got %d\n",
               strlen(buf), b);
      close(c);
      return -1;
   }

   return c;
}

// sends a single reply
// returns 0 on OK and -1 on error, with "reply" send in this case being the error message
int
vmci_reply(const int c,      // socket to use
           const char *reply // (json) to send back
         )
{
   int ret; // keep return values here
   uint32_t b; // smallish buffer

   // just being paranoid
   // TODO: add error code to network protocol instead if the hacky err msg.
   // Also wire the proper error handling in both python srv and GO client
   if (reply == NULL) {
      reply = "OK";
   }
   printf("vmci_reply: got to send '%s' on socket %d\n", reply, c);

   /*
    * And send one word back.
    */

   b = MAGIC;
   ret = send(c, &b, sizeof(b), 0);
   if (ret != sizeof(b)) {
      reply = "Failed to send magic";
      printf("%s: ret %d (%s) expected size %d\n",
               reply, ret, strerror(errno), sizeof(b));
      close(c);
      return -1;
   }

   b = strlen(reply) + 1; // send the string and trailing \0
   ret = send(c, &b, sizeof(b), 0);
   if (ret != sizeof(b)) {
      reply = "Failed to send len";
      printf("%s: ret %d (%s) expected size %d\n",
               reply, ret, strerror(errno), sizeof(b));
      close(c);
      return -1;
   }

   printf("Sending reply '%s'\n", reply);
   ret = send(c, reply, b, 0);
   if (b != ret) {
      printf("Failed to send content: ret %d (%s) expected size %d\n",
               ret, strerror(errno), b);
      close(c);
      return -1;
   }

   close(c);
   return 0;
}

void
vmci_close(int s)
{

   if (s != -1) {
      close(s);
   }
}
