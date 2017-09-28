#!/usr/bin/env python
# Copyright 2017 VMware, Inc. All Rights Reserved.
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

# A counter implementation that can be shared by multiple threads to
# keep track of the in-flight operations.

import logging
import threading

class OpsCounter:
    '''
    A counter object that can be shared by multiple threads to
    keep track of the in-flight operations.
    '''

    def __init__(self, initial_value = 0):
        self._value = initial_value
        self._event = threading.Event()
        self._lock = threading.RLock()

    def incr(self, delta = 1):
        '''
        Increment the counter with locking
        '''
        with self._lock:
            self._value += delta
            logging.debug("After incr: counter.value=%d", self._value)

            if self._event.is_set():
                logging.debug("Clearing counter event")
                self._event.clear()

    def decr(self, delta = 1):
        '''
        Decrement the counter with locking
        '''
        with self._lock:
            self._value -= delta
            logging.debug("After decr: counter.value=%d", self._value)

            if (self._value == 0):
                logging.debug("Setting counter event")
                self._event.set()

    @property
    def value(self):
        '''
        Return the current value of the counter
        '''
        return self._value

    def wait(self, timeout=None):
        '''
        Block until the counter value decreased to 0
        '''
        return self._event.wait(timeout)
