#!/usr/bin/env python
#
#  Copyright 2017 VMware, Inc. All Rights Reserved.
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
Unit  test invocation for local_sh.py
"""

import os
import unittest
import tempfile

import log_config
import local_sh
import shutil

class TestLocalShInfo(unittest.TestCase):
    """ Basic test for saving config DB link using local.sh. The test checks saving
    data into the sh-like fie """

    # Basic test: add new content. Replace it. Remove it.
    # Compare with original content - should be the same.
    # Also, on each step check some pattern in the current file
    def test_fileops(self):
        """Basic unit test - validates file operation on a tmp fake file """

        test_content = """
#!/bin/bash some
#stuff for rc.local.d/local.sh

# more
Some more code() !

exit 0

"""

        name = tempfile.mktemp()
        with open(name, "w") as f:
            f.write(test_content)
        for ds_name in ["DSTest1", "DSTest2", "DSTest3"]:
            local_sh.update_content(content=local_sh.CONFIG_DB_INFO.format(ds_name),
                                    tag=local_sh.CONFIG_DB_TAG, file=name)
            with open(name) as file_id:
                if file_id.read().find(ds_name) == -1:
                    self.assertEqual("failed with {}".format(ds_name), None)
        local_sh.update_content(content=None, tag=local_sh.CONFIG_DB_TAG, file=name, add=False)
        with open(name) as file_id:
            final_content = file_id.read()
        if final_content != test_content:
            self.assertEqual("result:\n{}\n expected :\n{}\n".format(final_content, test_content),
                             None)
        else:
            print("local.sh update/remove test - All good")
        os.remove(name)


if __name__ == "__main__":
    log_config.configure()
    unittest.main()