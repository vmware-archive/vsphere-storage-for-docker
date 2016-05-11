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
# limitations under the License

# Tests for cli_table.py

import unittest
import cli_table


class TestTableLogic(unittest.TestCase):
    """ Test any functions that don't require visual inspection
        Note that the number literals are the longest strings in each column before and after
        shrinking"""

    def test_max_column_sizes(self):
        header = ['Name', 'Greeting']
        data = [['Bill', 'Hello'], ['Jennifer', 'Sup'],
                ['Dave', 'How\'s it hangin?']]
        self.assertEqual([8, 16], cli_table.max_column_sizes(header, data))

    def test_long_values_shrink_column1(self):
        """ We only shrink the first column since it's much larger than the second """
        header = ['Name', 'Some Attribute']
        data = [[
            '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb22a8aa22f096b0b',
            'blahblahblah'
        ], ['Volume1', 'blah']]
        sizes = cli_table.max_column_sizes(header, data)
        self.assertEqual([64, 14], sizes)
        self.assertEqual([46, 14], cli_table.shrink_to_fit(sizes, 60))

    def test_long_values_shrink_column2(self):
        """ We only shrink the second column since it's much larger than the first """
        header = ['a', 'b']
        data = [
            ['x',
             '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb22a8aa22f096b0b'
             ], ['x', 'afdadsfdasfasfdasdfdasfadsfadsfdas']
        ]
        sizes = cli_table.max_column_sizes(header, data)
        self.assertEqual([1, 64], sizes)
        self.assertEqual([1, 49], cli_table.shrink_to_fit(sizes, 50))

    def test_shrink_multiple_equal_columns(self):
        """ We have 2 columns of equal sizes so we shrink them equally """
        header = ['a', 'b']
        data = [[
            '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb22a8aa22f096b0b',
            '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb22a8aa22f096b0b'
        ], ['x', 'afdadsfdasfasfdasdfdasfadsfadsfdas']]
        sizes = cli_table.max_column_sizes(header, data)
        self.assertEqual([64, 64], sizes)
        self.assertEqual([30, 30], cli_table.shrink_to_fit(sizes, 60))

    def test_shrink_2_equal_one_smaller_columns_equally(self):
        """
        Since the shrinking required is greater than the reduction of the first 2 columns to the
        size of the 3rd, we end up shrinking all the columns until they fit.
        """
        header = ['a', 'b', 'c']
        data = [[
            '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb22a8aa22f096b0b',
            '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb22a8aa22f096b0b',
            '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb2'
        ]]
        sizes = cli_table.max_column_sizes(header, data)
        self.assertEqual([64, 64, 50], sizes)
        self.assertEqual([20, 20, 20], cli_table.shrink_to_fit(sizes, 60))

    def test_shrink_2_columns_but_not_3rd(self):
        """ We can shrink the 2 larger first columns by themselves until they fit and are still
           larger than the 3rd column """
        header = ['a', 'b', 'c']
        data = [[
            '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb22a8aa22f096b0b',
            '2ff76be0906d860c7b7dc79cb6321a3f6a7fb9addf01233eb22a8aa22f096b0b',
            '2ff76be0906d860c7b7dc79cb6321a'
        ]]
        sizes = cli_table.max_column_sizes(header, data)
        self.assertEqual([64, 64, 30], sizes)
        self.assertEqual([45, 45, 30], cli_table.shrink_to_fit(sizes, 120))


if __name__ == '__main__':
    unittest.main()
