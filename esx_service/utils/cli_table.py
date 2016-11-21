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

# A small module for creating ascii based tables

import subprocess

SPACES = 2


def create(header, data):
    """
    Create a printable ascii table as a string and return it
    @param header - a list of strings for naming columns
    @param data - a list of rows (a list of strings) containing data

    Note that the length of each row must equal the length of the header list

    Warning: This function does not attempt to fit any fixed width terminal. It
    only formats the data given into columns and does not do any truncation.
    """
    max_sizes = max_column_sizes(header, data)
    ### Subtract the number of spaces between columns from the term width
    width = term_width() - SPACES * (len(header) - 1)
    sizes = shrink_to_fit(max_sizes, width)
    header = truncate([header], sizes)[0]
    data = truncate(data, sizes)
    return format_table2string(header, data, sizes)


def term_width():
    """
    Return the width of the terminal. This function assumes stty exists and is runnable. If that is
    not the case than a width of 80 characters is returned as a guess/default. This should never
    happen as stty exists on ESX, Linux and OSX.
    """
    try:
        # Get the number of rows and columns seperated by a space
        output = subprocess.check_output("stty size".split())
        return int(output.split()[1])
    except subprocess.CalledProcessError:
        return 80


def shrink_to_fit(column_sizes, terminal_width):
    """
    If the total size of all columns exceeds the terminal width, then we need to shrink the
    individual column sizes to fit. In most tables, there are one or two columns that are much
    longer than the other columns. We therefore tailor the shrinking algorithm based on this
    principle. The algorithm is as follows:

    1) Truncate the longest column until either the columns all fit in the terminal width or
       the size of the truncated column is equal to the next longest column.
    2) If the columns fit from truncating we are done.
    3) If the columns do not fit, shrink the now equally sized columns 1 char at a time until the
       width fits or these columns equal size of the next smallest column.
    4) Repeat steps 2 and 3 successively until a fit is found.

    Note that there is the pathological case that the terminal is smaller than a single character
    for all columns. Ignore this for now. The only way to handle it would be to print every column
    on it's own line and truncate them to fit. This may be a useful enhancement to make later.
    """
    total_size = sum(column_sizes)
    if total_size <= terminal_width:
        return column_sizes

    # Put the columns in sorted order (largest to smallest)
    sorted_sizes = sorted(column_sizes, reverse=True)

    # Find the index of each sorted column size in the original list and store it
    # Zero out the value in the original list so that we cover duplicate values, since list.index(val)
    # only finds the first instance.
    indexes = []
    for size in sorted_sizes:
        index = column_sizes.index(size)
        indexes.append(index)
        column_sizes[index] = 0

    # Shrink the sorted columns until they fit the terminal width
    while total_size > terminal_width:
        largest = sorted_sizes[0]
        num_largest_columns = sorted_sizes.count(largest)
        if num_largest_columns != len(sorted_sizes):
            next_largest = sorted_sizes[num_largest_columns]
        else:
            # All columns are the same size, so just shrink each one until they fit
            next_largest = 0

        to_remove = total_size - terminal_width
        gap = largest - next_largest

        if gap * num_largest_columns > to_remove:
            # We can resize in this step and we are done
            to_remove_per_column = int(to_remove / num_largest_columns)
            remainder = to_remove % num_largest_columns
            for i in range(num_largest_columns):
                sorted_sizes[i] = largest - to_remove_per_column
            for i in range(remainder):
                sorted_sizes[i] = sorted_sizes[i] - 1
        else:
            # We need to remove the max number of chars until we get to the next largest size then
            # try again
            for i in range(num_largest_columns):
                sorted_sizes[i] = next_largest

        total_size = sum(sorted_sizes)

    # Put the shrunken column sizes in their proper index locations
    for i in range(len(column_sizes)):
        index = indexes[i]
        column_sizes[index] = sorted_sizes[i]

    return column_sizes


def format_table2string(header, data, sizes):
    """ Actually create the table as a string """
    s = value_row(header, sizes) + '\n'
    s = s + divider_row(sizes) + '\n'
    for row in data:
        s = s + value_row(row, sizes) + '\n'
    return s


def value_row(values, sizes):
    """ Create a one line string of left justified values using the column sizes in sizes """
    s = ''
    for i in range(len(values)):
        s = s + values[i].ljust(sizes[i] + SPACES)
    return s


def divider_row(sizes):
    """ Create a one line string of '-' characters of given length in each column """
    s = ''
    for i in range(len(sizes)):
        s = s + '-'.ljust(sizes[i], '-') + '  '
    return s


def max_column_sizes(header, data):
    """ Determine the maximum length for each column and return the lengths as a list """
    sizes = [len(h) for h in header]
    for row in data:
        for i in range(len(header)):
            if len(row[i]) > sizes[i]:
                sizes[i] = len(row[i])
    return sizes


def truncate(data, sizes):
    """
    @param data - A list of a list of values
    @param sizes - A list of max sizes

    Shrink each value in data to it's corresponding max size
    Show truncation by replacing the last 2 characters with `..`
    """
    truncated = []
    for row in data:
        truncated_row = []
        for i in range(len(sizes)):
            size = sizes[i]
            column = row[i]
            if len(column) > size:
                truncated_row.append(column[:size - 2] + '..')
            else:
                truncated_row.append(column)
        truncated.append(truncated_row)
    return truncated
