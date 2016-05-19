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

#define CONN_SUCCESS 0

// Bad communication backend name (internal usage only)
#define CONN_BAD_BE_NAME -10  // Used internally, does not need to be be expose

// Misc failures
#define CONN_NO_VMCI_ADDRESS_FAMILY (-20)
#define CONN_FAILED_VSOCKET_OPEN (-30)
#define CONN_FAILED_TO_CONNECT (-40)
#define CONN_FAILED_MAGIC_SEND (-50)
#define CONN_FAILED_LEN_SEND (-60)
#define CONN_FAILED_CONTENT_SEND (-70)
#define CONN_FAILED_MAGIC_RECIEVE (-80)
#define CONN_FAILED_LEN_RECEIVE (-90)
#define CONN_MALLOC_FAILED (-100)
#define CONN_FAILED_CONTENT_RECIEVE (-110)


#endif // _CONNECTION_ERRORS_H_

