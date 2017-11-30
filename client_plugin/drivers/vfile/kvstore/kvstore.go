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

// KV Store interface
//
// Provide functionalities of Key-Value store for volume driver usage.

package kvstore

// VolStatus: Datatype for keeping status of a vFile volume
type VolStatus string

/*
   Constants:
   VolStateCreating:     vFile volume is being created. Not ready to be mounted.
   VolStateReady:        vFile volume is ready to be mounted but
                         no Samba service running right now.
   VolStateMounted:      Samba service already running. Volume mounted
                         on at least one host VM.
   VolStateMounting:     Volume is being mounted, File sharing service is being
                         started for this volume.
   VolStateUnmounting:   Volume is being unmounted, File sharing service is being
                         stopped for this volume.
   VolStateDeleting:     vFile volume is being deleted. Not ready to be mounted.
   VolStateError:        vFile volume in error state.

   VolPrefixState:       Each volume has three metadata keys. Each such
                         key terminates in the name of the volume, but
                         is preceded by a prefix. This is the prefix
                         for "State" key
   VolPrefixGRef:        The prefix for GRef key (Global refcount)
   VolPrefixInfo:        The prefix for info key. This key holds all
                         other metadata fields squashed into one

   VolumeDoesNotExistError:    Error indicating that there is no such volume
*/
const (
	VolStateCreating        VolStatus = "Creating"
	VolStateReady           VolStatus = "Ready"
	VolStateMounted         VolStatus = "Mounted"
	VolStateMounting        VolStatus = "Mounting"
	VolStateUnmounting      VolStatus = "Unmounting"
	VolStateDeleting        VolStatus = "Deleting"
	VolStateError           VolStatus = "Error"
	VolPrefixState                    = "SVOLS_stat_"
	VolPrefixGRef                     = "SVOLS_gref_"
	VolPrefixInfo                     = "SVOLS_info_"
	VolPrefixClient                   = "SVOLS_client_"
	VolPrefixStartTrigger             = "SVOLS_start_trigger_"
	VolPrefixStopTrigger              = "SVOLS_stop_trigger_"
	VolPrefixStartMarker              = "SVOLS_start_marker_"
	VolPrefixStopMarker               = "SVOLS_stop_marker_"
	VolumeDoesNotExistError           = "No such volume"
	OpGet                             = "Get"
	OpPut                             = "Put"
	OpDelete                          = "Delete"
)

// KvPair : Key Value pair holder
type KvPair struct {
	Key    string
	Value  string
	OpType string
}

// KvLoc: generic lock for a key
type KvLock interface {
	// BlockingLockWithLease - Try to blocking wait to get a lock on a key
	BlockingLockWithLease() error

	// TryLock - try a lock
	TryLock() error

	// ReleaseLock - releasing a lock
	ReleaseLock()

	// ClearLock - clean a lock
	ClearLock()
}

// KvStore is the interface for VolumeDriver to access a plugin-level KV store
type KvStore interface {
	// WriteMetaData - Update or Create volume metadata in KV store
	WriteMetaData(entries []KvPair) error

	// ReadMetaData - Read volume metadata in KV store
	ReadMetaData(keys []string) ([]KvPair, error)

	// UpdateMetaData - Read/Write/Delete metadata according to given key-value pairs
	UpdateMetaData(entries []KvPair) ([]KvPair, error)

	// DeleteMetaData - Delete volume metadata in KV store
	DeleteMetaData(name string) error

	// CompareAndPut - Compare the value of key with oldVal, if equal, replace with newVal
	CompareAndPut(key string, oldVal string, newVal string) bool

	// CompareAndPutStateOrBusywait - Compare the volume state with oldVal
	// if equal, replace with newVal and return true; or else, return false;
	// waits if volume is in a state from where it can reach the ready state
	CompareAndPutStateOrBusywait(key string, oldVal string, newVal string) bool

	// List - List all the different portion of keys with a given prefix
	List(prefix string) ([]string, error)

	// AtomicIncr - Increase a key value by one
	AtomicIncr(key string) error

	// AtomicDecr - Decrease a key value by one
	AtomicDecr(key string) error

	// BlockingWaitAndGet - Blocking wait until a key value becomes equal to a specific value
	// then read the value of another key
	BlockingWaitAndGet(key string, value string, newKey string) (string, error)

	// KvMapFromPrefix -  Create key-value pairs according to a given prefix
	KvMapFromPrefix(prefix string) (map[string]string, error)

	// DeleteClientMetaData - Delete volume client metadata in KV store
	DeleteClientMetaData(name string, nodeID string) error

	// CreateLock - Create a new lock based on a given key
	CreateLock(key string) (KvLock, error)
}
