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

# A counter implementation that can be shared by multiple threads.

import threading

class Counter:
    '''
    A counter object that can be shared by multiple threads.
    '''
    _lock = threading.RLock()

    def __init__(self, initial_value = 0):
        self._value = initial_value

    def incr(self, delta = 1):
        '''
        Increment the counter with locking
        '''
        with Counter._lock:
            self._value += delta

    def decr(self, delta = 1):
        '''
        Decrement the counter with locking
        '''
        with Counter._lock:
            self._value -= delta

    @property
    def value(self):
        return self._value
