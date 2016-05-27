#!/bin/bash

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


# A helper function to avoid the need for using sleep in tests. Given a long
# enough timeout, the wait_for function will deterministically validate whether
# a given condition has been met. It will immediately return (within one second)
# if the condition is true. The full timeout will only be waited for if the
# condition is never met, thus asserting that it is most likely never going to
# be met and the test has failed.

# Repeatedly try checking for a condition by calling a predicate function and
# waiting for it to return true. The function `$1` will be called every second
# for $2 seconds total. wait_for will return 0 as soon as $1 returns true, or
# return 1 on timeout.
function wait_for {
    for i in `seq 1 $2`
    do
      sleep 1
      $1
      if [ "$?" -eq 0 ] ; then
          return 0
      fi
    done
    return 1
}
