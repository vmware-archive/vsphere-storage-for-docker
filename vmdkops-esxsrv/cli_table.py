import unittest

# A small module for creating ascii based tables

def create(header, data):
    """
    Create a printable ascii table as a string and return it
    @param header - a list of strings for naming columns
    @param data - a list of rows (a list of strings) containing data

    Note that the length of each row must equal the length of the header list

    Warning: This function does not attempt to fit any fixed width terminal. It
    only formats the data given into columns and does not do any truncation.
    """
    sizes = max_column_sizes(header, data)
    return format(header, data, sizes)

def format(header, data, sizes):
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
        s = s + values[i].ljust(sizes[i]+2)
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


class TestTableLogic(unittest.TestCase):
    """ Test any functions that don't require visual inspection """

    def test_max_column_sizes(self):
        header = ['Name', 'Greeting']
        data = [['Bill', 'Hello'],
                ['Jennifer', 'Sup'],
                ['Dave', 'How\'s it hangin?']]
        self.assertEqual([8, 16], max_column_sizes(header, data))

if __name__ == '__main__':
  unittest.main()
