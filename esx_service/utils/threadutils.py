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
        self._lock = threading.Lock()
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
                lock = _get_lock(reentrant)
                self._lock_store[lockname] = lock
                logging.debug("LockManager.get_lock: new lock: %s, %s",
                              lockname, lock)
        return lock


def get_lock_decorator(reentrant=False):
    """
    Create a locking decorator to be used in modules
    """
    def lock_decorator(func):
        """
        Locking decorator
        """
        def protected(*args, **kwargs):
            """
            Locking wrapper
            """
            lock = _get_lock(reentrant)
            logging.debug("lock: %s, caller: %s, args: %s %s, will try to acquire",
                          lock, func.__name__, args, kwargs)
            with lock:
                logging.debug("lock: %s acquired for caller: %s, args: %s %s",
                              lock, func.__name__, args, kwargs)
                return func(*args, **kwargs)
            logging.debug("lock: %s released by caller: %s, args: %s %s",
                          lock, func.__name__, args, kwargs)
        return protected
    return lock_decorator

def get_local_storage():
    """Return a thread local storage object"""
    return threading.local()

def _get_lock(reentrant=False):
    """Return a thread Lock or Rlock"""
    if reentrant:
        return threading.RLock()
    else:
        return threading.Lock()
