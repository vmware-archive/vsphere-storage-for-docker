import unittest
import sys
import vmdkops_admin

class TestParsing(unittest.TestCase):

  def setUp(self):
    self.parser = vmdkops_admin.create_parser()

  def test_parse_ls_no_options(self):
    args = self.parser.parse_args(['ls'])
    self.assertEqual(args.func, vmdkops_admin.ls)
    self.assertEqual(args.l, False)
    self.assertEqual(args.c, None)

  def test_parse_ls_dash_l(self):
    args = self.parser.parse_args('ls -l'.split())
    self.assertEqual(args.func, vmdkops_admin.ls)
    self.assertEqual(args.l, True)
    self.assertEqual(args.c, None)

  def test_parse_ls_dash_c(self):
    args = self.parser.parse_args('ls -c created-by,created,last-attached'.split())
    self.assertEqual(args.func, vmdkops_admin.ls)
    self.assertEqual(args.l, False)
    self.assertEqual(args.c, ['created-by', 'created', 'last-attached'])

  def test_parse_ls_dash_c_invalid_argument(self):
    self.assert_parse_error('ls -c personality')

  def test_df(self):
    args = self.parser.parse_args('df'.split())
    self.assertEqual(args.func, vmdkops_admin.df)

  def test_df_badargs(self):
    self.assert_parse_error('df -l')

  def test_policy_no_args_fails(self):
    self.assert_parse_error('policy')

  def test_policy_create_no_args_fails(self):
    self.assert_parse_error('policy create')

  def test_policy_create(self):
    content = 'some policy content'
    cmd = 'policy create --name=testPolicy --content'.split()
    cmd.append("some policy content")
    args = self.parser.parse_args(cmd)
    self.assertEqual(args.content, 'some policy content')
    self.assertEqual(args.func, vmdkops_admin.policy_create)
    self.assertEqual(args.name, 'testPolicy')

  def test_policy_rm(self):
    args = self.parser.parse_args('policy rm testPolicy'.split())
    self.assertEqual(args.func, vmdkops_admin.policy_rm)
    self.assertEqual(args.name, 'testPolicy')

  def test_policy_rm_no_args_fails(self):
    self.assert_parse_error('policy rm')

  def test_policy_ls(self):
    args = self.parser.parse_args('policy ls'.split())
    self.assertEqual(args.func, vmdkops_admin.policy_ls)

  def test_policy_ls_badargs(self):
    self.assert_parse_error('policy ls --name=yo')

  def test_role_create(self):
    cmd = 'role create --name=carl --volume-maxsize=2TB ' + \
          '--matches-vm test*,qa* --rights=create,mount'
    args = self.parser.parse_args(cmd.split())
    self.assertEqual(args.func, vmdkops_admin.role_create)
    self.assertEqual(args.name, 'carl')
    self.assertEqual(args.volume_maxsize, '2TB')
    self.assertEqual(args.matches_vm, ['test*', 'qa*'])
    self.assertEqual(args.rights, ['create', 'mount'])

  def test_role_create_missing_option_fails(self):
    cmd = 'role create --name=carl --volume-maxsize=2TB --matches-vm=test*,qa*'
    self.assert_parse_error(cmd)

  def test_role_rm(self):
    args = self.parser.parse_args('role rm myRole'.split())
    self.assertEqual(args.func, vmdkops_admin.role_rm)
    self.assertEqual(args.name, 'myRole')

  def test_role_rm_missing_name(self):
    self.assert_parse_error('role rm')

  def test_role_ls(self):
    args = self.parser.parse_args('role ls'.split())
    self.assertEqual(args.func, vmdkops_admin.role_ls)

  def test_role_set(self):
    cmds = [
        'role set --name=carl --volume-maxsize=4TB',
        'role set --name=carl --rights create mount',
        'role set --name=carl --matches-vm marketing*',
        'role set --name=carl --volume-maxsize=2GB --rights create mount delete'
        ]
    for cmd in cmds:
      args = self.parser.parse_args(cmd.split())
      self.assertEqual(args.func, vmdkops_admin.role_set)
      self.assertEqual(args.name, 'carl')

  def test_role_set_missing_name_fails(self):
    self.assert_parse_error('role set --volume-maxsize=4TB')

  def test_role_get(self):
    args = self.parser.parse_args('role get testVm'.split())
    self.assertEqual(args.func, vmdkops_admin.role_get)
    self.assertEqual(args.vm_name, 'testVm')

  def test_status(self):
    args = self.parser.parse_args(['status'])
    self.assertEqual(args.func, vmdkops_admin.status)

  # Usage is always printed on a parse error. It's swallowed to prevent clutter.
  def assert_parse_error(self, command):
      with open('/dev/null', 'w') as f:
        sys.stdout = f
        sys.stderr = f
        with self.assertRaises(SystemExit):
          args = self.parser.parse_args(command.split())
        sys.stdout = sys.__stdout__
        sys.stderr = sys.__stderr__

if __name__ == '__main__':
  unittest.main()
