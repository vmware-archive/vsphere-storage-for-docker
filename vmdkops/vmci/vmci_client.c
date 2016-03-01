//
// VMCI sockets communication - client side.
//
// Called mainly from Go code.
//
// API: Exposes only Vmci_GetReply. The call is blocking.
//
// TODO: clean up messages and error handling
// TODO: split into  Vmci_IssueRequest and (blocking) Vmci_GetReply
//

#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <stdint.h>

#include "vmci_sockets.h"

// operations status. 0 is OK
typedef int be_sock_status;

//
// Booking structure for opened VMCI / vSocket
//
typedef struct {
   int sock_id; // socket id for socket APIs
   int vmci_id; // vmci id to track close()
   struct sockaddr_vm addr; // held here for bookkeeping and reporting
} be_sock_id;

//
// Protocol message structure: request and reply
//
#define MAGIC 0xbadbeef
typedef struct {
   uint32_t mlen;  // length of message (including trailing \0)
   const char *msg;     // expect null-term string there. Immutable.
} be_request;

#define ANSW_BUFSIZE 1024  // fixed size for json reply or specific errmsg
#define MAXBUF ANSW_BUFSIZE + 1 // Safety limit

typedef struct be_answer {
   int status; // TBD: OK, parse error, access denied, etc...
   char buf[ANSW_BUFSIZE];
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
   // TBD: split in request (asyn) , get_reply(sync)
   be_sock_status
   (*get_reply)(be_sock_id *id, be_request *r, be_answer* a);
} be_funcs;

// Vsocket communication implementation forward declarations
static be_sock_status
vsock_init(be_sock_id *id, int cid, int port);
static be_sock_status
vsock_get_reply(be_sock_id *id, be_request *r, be_answer* a);
static void
vsock_release(be_sock_id *id);

// Dummy communication declaration - dummy just prints stuff, for test
static be_sock_status
dummy_init(be_sock_id *id, int cid, int port);
static be_sock_status
dummy_get_reply(be_sock_id *id, be_request *r, be_answer* a);
static void
dummy_release(be_sock_id *id);

// support communication interfaces
#define VSOCKET_BE_NAME "vsocket" // backend to communicate via vSocket
#define ESX_VMCI_CID    2  		  // ESX host VMCI CID ("address")
#define DUMMY_BE_NAME "dummy"     // backend which only returns OK, for unit test

be_funcs backends[] =
   {
      {
      VSOCKET_BE_NAME, "vSocket Communication Backend v0.1", vsock_init,
      vsock_release, vsock_get_reply },
      {
      DUMMY_BE_NAME, "Dummy Communication Backend", dummy_init, dummy_release,
      dummy_get_reply },

      {
      0 } };

#define BAD_BE_NAME -10  // Used internally, does not need to be be expose

// Get backend by name
static be_funcs *
get_backend(const char *shortName)
{
   be_funcs *be = backends;
   while (be && be->shortName && *be->shortName) {
      if (strcmp(shortName, be->shortName) == 0) {
         return be;
      }
      be++;
   }
   return NULL;
}


// "dummy" interface implementation
// Used for manual testing mainly,
// to make sure data arrives to backend
//----------------------------------
static be_sock_status
dummy_init(be_sock_id *id, int cid, int port)
{
   // printf connecting
   printf("dummy_init: connected.\n");
   return 0;
}

static void
dummy_release(be_sock_id *id)
{
   printf("dumm_release: released.\n");
}

static be_sock_status
dummy_get_reply(be_sock_id *id, be_request *r, be_answer* a)
{
   printf("dummy_get_reply: got request %s.\n", r->msg);
   printf("dummy_get_reply: replying empty (for now).\n");
   a->buf[0] = '\0';
   a->status = 0;

   return 0;
}


// vsocket interface implementation
//---------------------------------

// Create and connect VMCI socket.
static be_sock_status
vsock_init(be_sock_id *id, int cid, int port)
{
   int ret;
   int af;    // family id
   int sock;     // socket id

   // The address family for vSockets is not static and must be acquired.
   // We hold onto the family we get back by keeping the fd to the device
   // open.

   af = VMCISock_GetAFValueFd(&id->vmci_id);
   if (af == -1) {
      fprintf(stderr, "Failed to get address family.\n");
      return -1;
   }

   sock = socket(af, SOCK_STREAM, 0);
   if (sock == -1) {
      perror("Failed to open socket");
      return errno;
   }

   id->sock_id = sock;

   // Connect to the server.
   memset(&id->addr, 0, sizeof id->addr);
   id->addr.svm_family = af;
   id->addr.svm_cid = cid;
   id->addr.svm_port = port;
   ret = connect(sock, (const struct sockaddr *) &id->addr, sizeof id->addr);
   if (ret == -1) {
      perror("Failed to connect");
      return errno;
   }

   fprintf(stderr, "Connected to %d:%d.\n", id->addr.svm_cid,
            id->addr.svm_port);


   return ret;
}

// Send request (r->msg) and wait for reply. Return reply in a->buf.
// Expect r and a to be allocated by the caller
static be_sock_status
vsock_get_reply(be_sock_id *s, be_request *r, be_answer* a)
{
   int ret;
   uint32_t b; // smallish buffer

 	// Try to send a message to the server.
   b = MAGIC;
   ret = send(s->sock_id, &b, sizeof b, 0);
   ret += send(s->sock_id, &r->mlen, sizeof(r->mlen), 0);
   ret += send(s->sock_id, r->msg, r->mlen + 1, 0);
   if (ret != (sizeof(b) + sizeof(r->mlen) + r->mlen + 1)) {
      perror("Failed to send ");
      return errno;
   }

   // magic:
   b = 0;
   ret = recv(s->sock_id, &b, sizeof b, 0);
   if (ret == -1 || b != MAGIC) {
      printf("Failed to receive magic: %d (%s) 0x%x\n", errno, strerror(errno),
               b);
      return ret;
   }

   // length, just in case:
   b = 0;
   ret = recv(s->sock_id, &b, sizeof b, 0);
   if (ret == -1) {
      printf("Failed to receive len: %d (%s)\n", errno, strerror(errno));
      return ret;
   }

   memset(a->buf, 0, sizeof a->buf); // pure paranoia
   ret = recv(s->sock_id, a->buf, sizeof(a->buf) - 1, 0);
   if (ret == -1) {
      printf("Failed to receive msg: %d (%s)", errno, strerror(errno));
      return ret;
   }
   if (strlen(a->buf) != b) {
      printf("Warning: len mismatch, strlen %zd, len %d\n", strlen(a->buf), b);
      return -1;
   }

   printf("Received '%s'.\n", a->buf);
   return 0;
}

// release socket and vmci info
static void
vsock_release(be_sock_id *id)
{
   close(id->sock_id);
   VMCISock_ReleaseAFValueFd(id->vmci_id);
}

//
// Handle one request using BE interface
// Yes,  we DO create and bind socket for each request - it's management
// so we can afford overhead, and it allows connection to be stateless.
//
static be_sock_status
host_request(be_funcs *be, be_request* req, be_answer* ans, int cid, int port)
{

   int vmciFd;
   int af;
   be_sock_id id;
   be_sock_status ret;

   if ((ret = be->init_sock(&id, cid, port)) != 0) {
      return ret;
   }

   if ((ret = be->get_reply(&id, req, ans)) != 0) {
      return ret;
   }

   be->release_sock(&id);
   return 0;
}


//
//
// Entry point for vsocket requests
//
be_sock_status
Vmci_GetReply(int port, const char* json_request, const char* be_name, be_answer* ans)
{
   	be_request req;

	be_funcs *be = get_backend(be_name);

   	if (be == NULL) {
    	  return BAD_BE_NAME;
   }
   req.mlen = strnlen(json_request, MAXBUF);
   req.msg = json_request;

   return host_request(be, &req, ans, ESX_VMCI_CID, port);
}
