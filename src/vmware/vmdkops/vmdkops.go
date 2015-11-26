// +build linux
package vmdkops

import (
	"encoding/json"
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

//
// * VMDK CADD (Create/Attach/Detach/Delete) operations client code.
// *
// * Requests operation from a Guest VM (sending json request over vSocket),
// * and expect the vmdkops_srv.py on the ESX host listening on vSocket.
// *
// * For each request:
// *   - Establishes a vSocket connection
// *   - Sends json string up to ESX
// *   - waits for reply and returns it
// *
// * Each requests has 4 bytes MAGIC, then 4 byts length (msg string length),
// * then 'length' + 1 null-terminated JSON string with command
// * On reply, returns MAGIC, length and message ("" or error message.)
// *
// * TODO: drop fprintf  and return better errors
// *
// * TODO: allow concurrency from multiple containers. Specifically, split into
// * Vmci_SubmitRequst() [not blocking] and Vmci_GetReply() [blocking] so the
// * goroutines can be concurrent
// *
// * TODO: add better length mgmt for ANSW_BUFSIZE
// *
// **** PREREQUISITES:
//   Build: open-vm-tools has to be installed - provided "vmci/vmci_sockets.h"
//   Run:   open-vm-tools has to be installed
//


/*

// VMCI sockets communication - client side. 

#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <stdint.h>

#include "vmci/vmci_sockets.h"

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

typedef struct {
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
   // printf releasing
   printf("dumm_release: released.\n");
}

static be_sock_status
dummy_get_reply(be_sock_id *id, be_request *r, be_answer* a)
{
   // printf asking
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

   // TBD: this needs to log or not at all , at least in prod
   printf("Communication BE: %s\n", be->name);

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
#define BAD_BE_NAME -10
be_sock_status
Vmci_GetReply(int port, const char* json_request, const char* be_name)
{
   	be_answer ans;
   	be_request req;

	be_funcs *be = get_backend(be_name);

   	if (be == NULL) {
    	  return BAD_BE_NAME;
   }
   req.mlen = strnlen(json_request, MAXBUF);
   req.msg = json_request;

   return host_request(be, &req, &ans, ESX_VMCI_CID, port);
}

*/
import "C"

const (
	vmciEsxPort int = 15000 // port we are connecting on. TBD: config?
)

var commBackendName string  = "vsocket" // could be changed in test

// Info we get about the volume from upstairs
type VolumeInfo struct {
	Name    string            `json:"Name"`
	Options map[string]string `json:"Opts,omitempty"`
}

// A request to be passed to ESX service
type requestToVmci struct {
	Ops     string  `json:"cmd"`
	Details VolumeInfo `json:"details"`
}

// Send a command 'cmd' to VMCI, via C API
func vmdkCmd(cmd string, name string, opts map[string]string)  string {

	json_str, err := json.Marshal(&requestToVmci{
			Ops: cmd,
			Details: VolumeInfo{Name: name, Options: opts,}}); if err != nil {
				return fmt.Sprintf("Failed to marshal json: %s", err)
			}

	cmd_s := C.CString(string(json_str))
	defer C.free(unsafe.Pointer(cmd_s))

	be_s := C.CString(commBackendName)
	defer C.free(unsafe.Pointer(be_s))

	// connect, send command, get reply, disconnect - all in one shot
	ret := C.Vmci_GetReply(C.int(vmciEsxPort), cmd_s, be_s)

	if ret != 0 {
		log.Print("Warning - no connection to ESX over vsocket, trace only")
		return fmt.Sprintf("vmdkCmd err: %d (%s)", ret, syscall.Errno(ret).Error())
	}

	return ""
}

// public API
func (v VolumeInfo) Create() string {
	return vmdkCmd("create", v.Name, v.Options)
}
func (v VolumeInfo) Remove() string {
	return vmdkCmd("create", v.Name, v.Options)
}
func (v VolumeInfo) Attach() string {
	return vmdkCmd("create", v.Name, v.Options)
}
func (v VolumeInfo) Detach() string {
	return vmdkCmd("create", v.Name, v.Options)
}
func (v VolumeInfo) List() string {
	return vmdkCmd("list", v.Name, v.Options)
}

func VmdkCreate(name string, opts map[string]string) string {
	return vmdkCmd("create", name, opts)
}
func VmdkRemove(name string, opts map[string]string) string {
	return vmdkCmd("remove", name, opts)
}
func VmdkAttach(name string, opts map[string]string) string {
	return vmdkCmd("attach", name, opts)
}
func VmdkDetach(name string, opts map[string]string) string {
	return vmdkCmd("detach", name, opts)
}
func VmdkList(name string, opts map[string]string) string {
	return vmdkCmd("list", name, opts)
}


func TestSetDummyBackend() {
	commBackendName = "dummy"
	}
