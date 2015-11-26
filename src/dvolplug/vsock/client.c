
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


// returns socket id or -1
int
init_vsocket(int *vmciFd, int *af)
{
   int ret;
   /*
    * The address family for vSockets is not static and must be acquired.
    * We hold onto the family we get back by keeping the fd to the device
    * open.
    */

   *af = VMCISock_GetAFValueFd(vmciFd);
   if (-1 == *af) {
      fprintf(stderr, "Failed to get address family.\n");
      return -1;
   }

   ret = socket(*af, SOCK_STREAM, 0);
   if (-1 == ret) {
      perror("Failed to open socket");
      return -1;
   }

   return ret;
}

/*
 * Open a STREAM socket using our address family.
 */
// returns 0 for OK, -1 for error
int
connect_and_send_msg(int s, int af, int cid, int port)
{
   char buf[32];
   struct sockaddr_vm addr;
   int ret;

   /*
    * Connect to the server.
    */

   memset(&addr, 0, sizeof addr);
   addr.svm_family = af;
   addr.svm_cid = cid;
   addr.svm_port = port;
   ret = connect(s, (const struct sockaddr *) &addr, sizeof addr);
   if (-1 == ret) {
      perror("Failed to connect");
      return -1;
   }

   printf("Connected to %d:%d.\n", addr.svm_cid, addr.svm_port);

   /*
    * Try to send a message to the server.
    */

   memset(buf, 0, sizeof buf);
   strncpy(buf, "Hello", sizeof buf);
   ret = send(s, buf, strlen(buf) + 1, 0);
   if (strlen(buf) + 1 != ret) {
      perror("Failed to send");
      return -1;
   }

   printf("Sent %d bytes.\n", ret);
   return 0;
}

// returns 0 for success, -1 for failure
int
get_reply(int s)
{
   char buf[32];
   /*
    * And try to get a reply back.
    */

   memset(buf, 0, sizeof buf);
   if (recv(s, buf, sizeof buf, 0) == -1) {
      perror("Failed to receive");
      return -1;
   }

   printf("Received '%s'.\n", buf);
   return 0;
}

void
close_and_release(int s, int vmciFd)
{
   close(s);
   VMCISock_ReleaseAFValueFd(vmciFd);
}

int
doit(int cid, int port)
{
   int s = -1;
   int vmciFd;
   int af;

   if ((s = init_vsocket(&vmciFd, &af)) == -1) {
      return -1;
   }

   if (connect_and_send_msg(s, af, cid, port) == -1) {
      return -2;
   }

   if (get_reply(s) == -1) {
      return -3;
   }

   close_and_release(s, vmciFd);
   return 0;
}

#ifdef __DEFINE_MAIN
int
main(int argc,       // IN
     char *argv[])   // IN
{
   int ret;

   /*
    * We take the context ID as the first argument and the port as the second.
    */

   if (argc < 3) {
      fprintf(stderr, "Usage: %s <cid> <port>.\n", argv[0]);
      return -1;
   }
   ret = doit(atoi(argv[1]), atoi(argv[2]));
   exit(ret);
}

#endif /* __DEFINE_MAIN */
