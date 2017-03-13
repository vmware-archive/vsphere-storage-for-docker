# Copyright 2016 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Implements threading utilities
Written by Bruno Moura <brunotm@gmail.com>
"""

import threading
import logging
from weakref import WeakValueDictionary

class LockManager(object):
    """
    Thread safe lock manager class
    """
    def __init__(self):
        self._lock = get_lock()
        self._lock_store = WeakValueDictionary()

    def get_lock(self, lockname, reentrant=False):
        """
        Create or return a existing lock identified by lockname.
        """
        with self._lock:
            try:
                lock = self._lock_store[lockname]
                logging.debug("LockManager.get_lock: existing lock: %s, %s",
                              lockname, lock)
            except KeyError:
                lock = get_lock(reentrant)
                self._lock_store[lockname] = lock
                logging.debug("LockManager.get_lock: new lock: %s, %s",
                              lockname, lock)
            logging.debug("LockManager existing locks in store: %s",
                          self._list_locks())
            return lock

    def _list_locks(self):
        return self._lock_store.keys()

    def list_locks(self):
        """
        Return a list of existing lock names in lock store.
        """
        with self._lock:
            return self._list_locks()


def get_lock_decorator(reentrant=False):
    """
    Create a locking decorator to be used in modules
    """
    # Lock to be used in the decorator
    lock = get_lock(reentrant)
    def lock_decorator(func):
        """
        Locking decorator
        """
        def protected(*args, **kwargs):
            """
            Locking wrapper
            """
            # Get lock memory address for debugging
            lockaddr = hex(id(lock))
            # logging.debug("Trying to acquire lock: %s @ %s, caller: %s, args: %s %s",
            #               lock, lockaddr, func.__name__, args, kwargs)
            with lock:
                # logging.debug("Acquired lock: %s @ %s, caller: %s, args: %s %s",
                #               lock, lockaddr, func.__name__, args, kwargs)
                return func(*args, **kwargs)
        return protected
    return lock_decorator


def start_new_thread(target, args):
    """Start a new thread"""
    threading.Thread(target=target, args=args).start()
    logging.debug("Currently active threads: %s",
                  get_active_threads())


def get_active_threads():
    """Return the list of active thread objects"""
    return threading.enumerate()


def get_local_storage():
    """Return a thread local storage object"""
    return threading.local()


def set_thread_name(name):
    """Set the current thread name"""
    threading.current_thread().name = name


def get_thread_name():
    """Get the current thread name"""
    return threading.current_thread().name


def get_lock(reentrant=False):
    """Return a unmanaged thread Lock or Rlock"""
    if reentrant:
        return threading.RLock()
    else:
        return threading.Lock()
